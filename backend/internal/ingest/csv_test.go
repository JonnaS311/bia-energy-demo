package ingest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const header = "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status\n"

// RF-D-02: BOM + CRLF se tratan igual que LF.
func TestParseReadingsBOMAndCRLF(t *testing.T) {
	body := "\xEF\xBB\xBF" + strings.ReplaceAll(header+"M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\nM-101,2026-09-01 01:00:00,20.11,221.15,100.49,0.935,OK\n", "\n", "\r\n")
	res, err := ParseReadings(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 2 || len(res.Rejected) != 0 {
		t.Fatalf("rows=%d rejected=%d", len(res.Rows), len(res.Rejected))
	}
	if got := res.Rows[0].Timestamp; !got.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("timestamp %v", got)
	}
}

// RF-D-02: columna extra se ignora; encabezado distinto aborta.
func TestParseReadingsHeader(t *testing.T) {
	res, err := ParseReadings(strings.NewReader("meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status,site\nM-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK,A\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ExtraColumns) != 1 || res.ExtraColumns[0] != "site" || len(res.Rows) != 1 {
		t.Fatalf("extra=%v rows=%d", res.ExtraColumns, len(res.Rows))
	}
	_, err = ParseReadings(strings.NewReader("meter_id,timestamp,kwh,voltage_v,current_a,power_factor,status\n"))
	var sm *ErrSchemaMismatch
	if !errors.As(err, &sm) {
		t.Fatalf("expected schema mismatch, got %v", err)
	}
}

// RF-D-04: un rechazo por código, con número de línea.
func TestParseReadingsRejections(t *testing.T) {
	rows := []struct {
		line   string
		reason string
	}{
		{"M-101,2026-09-01 00:00:00,23.5", ReasonColumnCount},
		{"m101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK", ReasonInvalidMeterID},
		{"M-1000,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK", ReasonInvalidMeterID},
		{"M-101,2026/09/01 00:00,23.5,221.9,101.28,0.954,OK", ReasonInvalidTS},
		{"M-101,2026-09-01 02:00:00,,221.9,101.28,0.954,OK", ReasonNotANumber},
		{"M-101,2026-09-01 03:00:00,abc,221.9,101.28,0.954,OK", ReasonNotANumber},
		{"M-101,2026-09-01 04:00:00,-1.5,221.9,101.28,0.954,OK", ReasonNegativeKWh},
		{"M-101,2026-09-01 05:00:00,1.5,0,101.28,0.954,OK", ReasonVoltage},
		{"M-101,2026-09-01 06:00:00,1.5,220,-3,0.954,OK", ReasonNegativeCurrent},
		{"M-101,2026-09-01 07:00:00,1.5,220,3,1.2,OK", ReasonPF},
	}
	var b strings.Builder
	b.WriteString(header)
	for _, r := range rows {
		b.WriteString(r.line + "\n")
	}
	res, err := ParseReadings(strings.NewReader(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 0 || len(res.Rejected) != len(rows) {
		t.Fatalf("rows=%d rejected=%d", len(res.Rows), len(res.Rejected))
	}
	for i, r := range rows {
		if res.Rejected[i].Reason != r.reason || res.Rejected[i].Line != i+2 {
			t.Errorf("row %d: got %+v want reason %s line %d", i, res.Rejected[i], r.reason, i+2)
		}
	}
}

// RF-D-03 / CB-D-12: espacios alrededor se recortan.
func TestParseReadingsTrim(t *testing.T) {
	res, err := ParseReadings(strings.NewReader(header + " M-101 , 2026-09-01 00:00:00 , 23.5 ,221.9,101.28,0.954,OK\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 1 || res.Rows[0].MeterID != "M-101" {
		t.Fatalf("%+v %+v", res.Rows, res.Rejected)
	}
}

// RF-D-05: duplicados, la última fila gana.
func TestParseReadingsDuplicates(t *testing.T) {
	res, err := ParseReadings(strings.NewReader(header +
		"M-101,2026-09-01 08:00:00,35.07,221.9,101.28,0.954,OK\n" +
		"M-101,2026-09-01 09:00:00,35.00,221.9,101.28,0.954,OK\n" +
		"M-101,2026-09-01 08:00:00,99.9,221.9,101.28,0.954,OK\n"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Duplicates != 1 || len(res.Rows) != 2 || res.Rows[0].ConsumptionKWh != 99.9 {
		t.Fatalf("dups=%d rows=%+v", res.Duplicates, res.Rows)
	}
	if res.DuplicatesByMeterDay["M-101"]["2026-09-01"] != 1 {
		t.Fatalf("by day %+v", res.DuplicatesByMeterDay)
	}
}

// RF-D-03: eventos con y sin segundos; tipo vacío se rechaza.
func TestParseEvents(t *testing.T) {
	res, err := ParseEvents(strings.NewReader("meter_id,event_timestamp,event_type,description\n" +
		"M-104,2026-09-11 00:00,OPERATIONAL_CHANGE,New production line activated\n" +
		"M-109,2026-09-12 14:00:00,UNKNOWN,No operational event reported\n" +
		"M-104,2026-09-11 00:00,,texto\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 2 || len(res.Rejected) != 1 || res.Rejected[0].Reason != ReasonEmptyEventType {
		t.Fatalf("rows=%d rej=%+v", len(res.Rows), res.Rejected)
	}
	if !res.Rows[0].Timestamp.Equal(time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("ts %v", res.Rows[0].Timestamp)
	}
}

// Dataset real: 4.032 lecturas, 4 eventos, 0 rechazos, 0 duplicados.
func TestParseRealDataset(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "data")
	f, err := os.Open(filepath.Join(dir, "readings.csv"))
	if err != nil {
		t.Fatalf("dataset not found: %v", err)
	}
	defer f.Close()
	res, err := ParseReadings(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 4032 || len(res.Rejected) != 0 || res.Duplicates != 0 {
		t.Fatalf("rows=%d rejected=%d dups=%d", len(res.Rows), len(res.Rejected), res.Duplicates)
	}
	g, err := os.Open(filepath.Join(dir, "events.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	ev, err := ParseEvents(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev.Rows) != 4 || len(ev.Rejected) != 0 {
		t.Fatalf("events=%d rejected=%d", len(ev.Rows), len(ev.Rejected))
	}
}
