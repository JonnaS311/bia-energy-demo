package ingest

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// Estados de ingesta (RF-D-07).
const (
	StatusPending   = "PENDING"
	StatusRunning   = "RUNNING"
	StatusCompleted = "COMPLETED"
	StatusFailed    = "FAILED"
)

// Stats es ingest_stats (spec de data §8.3).
type Stats struct {
	Status     string     `json:"status"`
	Readings   int        `json:"readings"`
	Events     int        `json:"events"`
	Meters     int        `json:"meters"`
	Rejected   int        `json:"rejected"`
	Duplicates int        `json:"duplicates"`
	DurationMs int64      `json:"duration_ms"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	// DuplicatesByMeterDay alimenta el detector de calidad de datos (d); no se serializa.
	DuplicatesByMeterDay map[string]map[string]int `json:"-"`
}

// Tracker guarda el estado de la última ingesta de forma concurrente-segura.
type Tracker struct {
	mu    sync.RWMutex
	stats Stats
	run   sync.Mutex
}

// NewTracker crea un tracker en estado PENDING.
func NewTracker() *Tracker { return &Tracker{stats: Stats{Status: StatusPending}} }

// Get devuelve una copia del estado actual.
func (t *Tracker) Get() Stats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stats
}

// Set reemplaza el estado.
func (t *Tracker) Set(s Stats) {
	t.mu.Lock()
	t.stats = s
	t.mu.Unlock()
}

// Busy indica si la ingesta aún no terminó (los endpoints de datos responden 503).
func (t *Tracker) Busy() bool {
	s := t.Get().Status
	return s == StatusPending || s == StatusRunning
}

// Writer es la persistencia que necesita la ingesta (implementada por store.Store).
type Writer interface {
	UpsertMeters(ctx context.Context, meterIDs []string) error
	UpsertReadings(ctx context.Context, rows []domain.Reading) error
	UpsertEvents(ctx context.Context, rows []domain.Event) error
	Counts(ctx context.Context) (readings, events, meters int, err error)
}

// Run ejecuta la ingesta de DATA_DIR/readings.csv y DATA_DIR/events.csv (RF-D-01..RF-D-07).
// Solo abre esos dos archivos (RF-D-10). Nunca aborta el arranque: los errores quedan en el log y en Stats.
func Run(ctx context.Context, w Writer, dataDir string, tr *Tracker, log *slog.Logger) Stats {
	if !tr.run.TryLock() {
		log.Warn("INGEST_ALREADY_RUNNING")
		return tr.Get()
	}
	defer tr.run.Unlock()

	start := time.Now().UTC()
	st := Stats{Status: StatusRunning, StartedAt: &start, DuplicatesByMeterDay: map[string]map[string]int{}}
	tr.Set(st)

	readingsPath, _ := filepath.Abs(filepath.Join(dataDir, "readings.csv"))
	eventsPath, _ := filepath.Abs(filepath.Join(dataDir, "events.csv"))
	log.Info("ingest_started", "readings_path", readingsPath, "events_path", eventsPath)

	failed := false

	// Lecturas.
	var readings *ParseResult[domain.Reading]
	if f, err := os.Open(readingsPath); err != nil {
		log.Error("INGEST_FILE_MISSING", "path", readingsPath)
	} else {
		readings, err = ParseReadings(f)
		_ = f.Close()
		if err != nil {
			var sm *ErrSchemaMismatch
			if errors.As(err, &sm) {
				log.Error("INGEST_SCHEMA_MISMATCH", "file", "readings.csv", "expected", sm.Expected, "found", sm.Found)
			} else {
				log.Error("INGEST_READ_FAILED", "file", "readings.csv", "error", err.Error())
			}
			readings = nil
		}
	}

	// Eventos.
	var events *ParseResult[domain.Event]
	if f, err := os.Open(eventsPath); err != nil {
		log.Error("INGEST_FILE_MISSING", "path", eventsPath)
	} else {
		events, err = ParseEvents(f)
		_ = f.Close()
		if err != nil {
			var sm *ErrSchemaMismatch
			if errors.As(err, &sm) {
				log.Error("INGEST_SCHEMA_MISMATCH", "file", "events.csv", "expected", sm.Expected, "found", sm.Found)
			} else {
				log.Error("INGEST_READ_FAILED", "file", "events.csv", "error", err.Error())
			}
			events = nil
		}
	}

	// Semilla de medidores antes de escribir lecturas (RF-D-06).
	meterSet := map[string]bool{}
	var meterIDs []string
	if readings != nil {
		for _, r := range readings.Rows {
			if !meterSet[r.MeterID] {
				meterSet[r.MeterID] = true
				meterIDs = append(meterIDs, r.MeterID)
			}
		}
	}
	if events != nil {
		for _, e := range events.Rows {
			if !meterSet[e.MeterID] {
				meterSet[e.MeterID] = true
				meterIDs = append(meterIDs, e.MeterID)
				log.Warn("INGEST_METER_WITHOUT_READINGS", "meter_id", e.MeterID)
			}
		}
	}
	if len(meterIDs) > 0 {
		if err := w.UpsertMeters(ctx, meterIDs); err != nil {
			log.Error("INGEST_TX_FAILED", "file", "meters", "error", err.Error())
			failed = true
		}
	}

	if readings != nil && !failed {
		logRejections(log, "readings.csv", readings.Rejected, readings.ExtraColumns)
		st.Rejected += len(readings.Rejected)
		st.Duplicates += readings.Duplicates
		st.DuplicatesByMeterDay = readings.DuplicatesByMeterDay
		if err := w.UpsertReadings(ctx, readings.Rows); err != nil {
			log.Error("INGEST_TX_FAILED", "file", "readings.csv", "error", err.Error())
			failed = true
		}
	}
	if events != nil && !failed {
		logRejections(log, "events.csv", events.Rejected, events.ExtraColumns)
		st.Rejected += len(events.Rejected)
		st.Duplicates += events.Duplicates
		if err := w.UpsertEvents(ctx, events.Rows); err != nil {
			log.Error("INGEST_TX_FAILED", "file", "events.csv", "error", err.Error())
			failed = true
		}
	}

	if r, e, m, err := w.Counts(ctx); err == nil {
		st.Readings, st.Events, st.Meters = r, e, m
	} else {
		failed = true
		log.Error("DB_ERROR", "error", err.Error())
	}

	end := time.Now().UTC()
	st.FinishedAt = &end
	st.DurationMs = end.Sub(start).Milliseconds()
	st.Status = StatusCompleted
	if failed {
		st.Status = StatusFailed
	}
	tr.Set(st)
	log.Info("ingest_completed", "status", st.Status, "readings", st.Readings, "events", st.Events, "meters", st.Meters,
		"rejected", st.Rejected, "duplicates", st.Duplicates, "duration_ms", st.DurationMs)
	return st
}

func logRejections(log *slog.Logger, file string, rej []Rejection, extra []string) {
	if len(extra) > 0 {
		log.Warn("INGEST_EXTRA_COLUMNS", "file", file, "columns", extra)
	}
	for _, r := range rej {
		log.Warn("INGEST_ROW_REJECTED", "file", file, "line", r.Line, "reason", r.Reason, "meter_id", r.MeterID)
	}
}
