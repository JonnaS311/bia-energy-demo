package ingest

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

type fakeWriter struct {
	meters   []string
	readings []domain.Reading
	events   []domain.Event
	failOn   string
}

func (f *fakeWriter) UpsertMeters(_ context.Context, ids []string) error {
	if f.failOn == "meters" {
		return errors.New("db down")
	}
	f.meters = append(f.meters, ids...)
	return nil
}
func (f *fakeWriter) UpsertReadings(_ context.Context, rows []domain.Reading) error {
	if f.failOn == "readings" {
		return errors.New("db down")
	}
	f.readings = rows
	return nil
}
func (f *fakeWriter) UpsertEvents(_ context.Context, rows []domain.Event) error {
	f.events = rows
	return nil
}
func (f *fakeWriter) Counts(context.Context) (int, int, int, error) {
	return len(f.readings), len(f.events), len(f.meters), nil
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func logger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewJSONHandler(&buf, nil)), &buf
}

const evHeader = "meter_id,event_timestamp,event_type,description\n"

func TestRunHappyPath(t *testing.T) { // RF-D-01, RF-D-06, RF-D-07, RF-D-10
	dir := t.TempDir()
	writeFile(t, dir, "readings.csv", header+"M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\nM-101,2026-09-01 01:00:00,1.2,221.9,101.28,1.5,OK\n")
	writeFile(t, dir, "events.csv", evHeader+"M-150,2026-09-11 00:00,OPERATIONAL_CHANGE,x\n")
	writeFile(t, dir, "otro_archivo.csv", "secreto,no,leer\n")
	w := &fakeWriter{}
	log, buf := logger()
	tr := NewTracker()
	if !tr.Busy() {
		t.Fatal("tracker should start PENDING")
	}
	st := Run(context.Background(), w, dir, tr, log)
	if st.Status != StatusCompleted || st.Readings != 1 || st.Events != 1 || st.Meters != 2 || st.Rejected != 1 || tr.Busy() {
		t.Fatalf("stats %+v", st)
	}
	out := buf.String()
	for _, want := range []string{"ingest_started", "ingest_completed", "INGEST_ROW_REJECTED", `"reason":"PF_OUT_OF_RANGE"`, "INGEST_METER_WITHOUT_READINGS"} {
		if !strings.Contains(out, want) {
			t.Errorf("log missing %s", want)
		}
	}
	if strings.Contains(out, "otro_archivo") || strings.Contains(out, "secreto") {
		t.Error("RF-D-10: foreign file was read or logged")
	}
}

func TestRunMissingAndBadHeader(t *testing.T) { // CB-D-01, CB-D-02, RF-D-02
	dir := t.TempDir()
	writeFile(t, dir, "readings.csv", "meter_id,timestamp,kwh,voltage_v,current_a,power_factor,status\n")
	w := &fakeWriter{}
	log, buf := logger()
	st := Run(context.Background(), w, dir, NewTracker(), log)
	if st.Status != StatusCompleted || st.Readings != 0 || st.Events != 0 {
		t.Fatalf("stats %+v", st)
	}
	if !strings.Contains(buf.String(), "INGEST_SCHEMA_MISMATCH") || !strings.Contains(buf.String(), "INGEST_FILE_MISSING") {
		t.Fatalf("log %s", buf.String())
	}
}

func TestRunTransactionFailure(t *testing.T) { // CB-D-10
	dir := t.TempDir()
	writeFile(t, dir, "readings.csv", header+"M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n")
	writeFile(t, dir, "events.csv", evHeader)
	w := &fakeWriter{failOn: "readings"}
	log, buf := logger()
	st := Run(context.Background(), w, dir, NewTracker(), log)
	if st.Status != StatusFailed || !strings.Contains(buf.String(), "INGEST_TX_FAILED") {
		t.Fatalf("stats %+v log %s", st, buf.String())
	}
}

func TestRunConcurrentCallIsRejected(t *testing.T) { // CB-D-09
	tr := NewTracker()
	tr.run.Lock()
	defer tr.run.Unlock()
	log, buf := logger()
	Run(context.Background(), &fakeWriter{}, t.TempDir(), tr, log)
	if !strings.Contains(buf.String(), "INGEST_ALREADY_RUNNING") {
		t.Fatal("expected INGEST_ALREADY_RUNNING")
	}
}

func TestParseEventsBadRows(t *testing.T) {
	res, err := ParseEvents(strings.NewReader(evHeader + "m-1,2026-09-11 00:00,X,d\nM-101,11/09/2026,X,d\nM-101,2026-09-11 00:00,X\nM-101,2026-09-11 00:00,X,a\nM-101,2026-09-11 00:00,X,b\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rejected) != 3 || res.Duplicates != 1 || len(res.Rows) != 1 || res.Rows[0].Description != "b" {
		t.Fatalf("%+v", res)
	}
	if _, err := ParseEvents(strings.NewReader("")); err == nil {
		t.Fatal("empty file should be a schema mismatch")
	}
}
