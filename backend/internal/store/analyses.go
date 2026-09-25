package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
)

// ErrAnalysisInProgress indica que ya hay un análisis QUEUED/RUNNING.
type ErrAnalysisInProgress struct{ ID uuid.UUID }

func (e *ErrAnalysisInProgress) Error() string { return "analysis in progress: " + e.ID.String() }

// Analysis es una fila de analyses.
type Analysis struct {
	ID           uuid.UUID
	Status       domain.AnalysisStatus
	Steps        []domain.Step
	Summary      *domain.Summary
	ErrorMessage *string
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time
}

// CreateAnalysis inserta un análisis QUEUED dentro de una transacción con SELECT … FOR UPDATE (RF-B-11).
func (s *Store) CreateAnalysis(ctx context.Context) (uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Bloqueo de tabla corto: serializa creaciones concurrentes aunque no exista ninguna fila activa que bloquear.
	if _, err := tx.Exec(ctx, `LOCK TABLE analyses IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return uuid.Nil, err
	}
	var active uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM analyses WHERE status IN ('QUEUED','RUNNING') ORDER BY created_at DESC LIMIT 1 FOR UPDATE`).Scan(&active)
	if err == nil {
		return uuid.Nil, &ErrAnalysisInProgress{ID: active}
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	id := uuid.New()
	steps, _ := json.Marshal(domain.NewSteps())
	if _, err := tx.Exec(ctx, `INSERT INTO analyses (id, status, steps) VALUES ($1, 'QUEUED', $2)`, id, steps); err != nil {
		return uuid.Nil, err
	}
	return id, tx.Commit(ctx)
}

// ActiveAnalysisID devuelve el análisis QUEUED/RUNNING más reciente, si existe.
func (s *Store) ActiveAnalysisID(ctx context.Context) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM analyses WHERE status IN ('QUEUED','RUNNING') ORDER BY created_at DESC LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	return id, err == nil, err
}

// MarkRunning pasa el análisis a RUNNING.
func (s *Store) MarkRunning(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE analyses SET status = 'RUNNING', started_at = now() WHERE id = $1`, id)
	return err
}

// UpdateSteps persiste el arreglo de etapas (una escritura por transición, RF-B-12).
func (s *Store) UpdateSteps(ctx context.Context, id uuid.UUID, steps []domain.Step) error {
	b, _ := json.Marshal(steps)
	_, err := s.pool.Exec(ctx, `UPDATE analyses SET steps = $2 WHERE id = $1`, id, b)
	return err
}

// FailAnalysis marca el análisis FAILED sin tocar anomalías ni medidores.
func (s *Store) FailAnalysis(ctx context.Context, id uuid.UUID, steps []domain.Step, msg string) error {
	b, _ := json.Marshal(steps)
	_, err := s.pool.Exec(ctx, `UPDATE analyses SET status = 'FAILED', steps = $2, error_message = $3, finished_at = now() WHERE id = $1`, id, b, msg)
	return err
}

// FailOrphanAnalyses marca FAILED los análisis QUEUED/RUNNING al arrancar (CB-B-09).
func (s *Store) FailOrphanAnalyses(ctx context.Context) (int, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE analyses SET status = 'FAILED', error_message = 'process restarted', finished_at = now()
		WHERE status IN ('QUEUED','RUNNING')`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// CompleteAnalysis publica los resultados en una sola transacción (RF-B-12 paso 3).
func (s *Store) CompleteAnalysis(ctx context.Context, id uuid.UUID, steps []domain.Step, res engine.Result, summary domain.Summary) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE anomalies SET superseded = true WHERE superseded = false`); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, a := range res.Anomalies {
		aid := uuid.New()
		points, _ := json.Marshal(a.ExplanationPoints)
		var related *int64
		if a.RelatedEvent != nil && a.RelatedEvent.ID > 0 {
			related = &a.RelatedEvent.ID
		}
		if _, err := tx.Exec(ctx, `INSERT INTO anomalies (id, analysis_id, meter_id, detected_at, window_start, window_end, type, severity,
			confidence, priority_rank, reason, recommended_action, explanation_points, explanation_source, related_event_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			aid, id, a.MeterID, now, a.WindowStart, a.WindowEnd, string(a.Type), string(a.Severity), a.Confidence, a.PriorityRank,
			a.Reason, a.RecommendedAction, points, a.ExplanationSource, related); err != nil {
			return fmt.Errorf("insert anomaly %s: %w", a.MeterID, err)
		}
		b := &pgx.Batch{}
		for _, e := range a.Evidence {
			b.Queue(`INSERT INTO anomaly_evidence (anomaly_id, position, metric, observed, baseline, delta_pct, unit, "window", detail)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, aid, e.Position, e.Metric, e.Observed, e.Baseline, e.DeltaPct, e.Unit, e.Window, e.Detail)
		}
		if err := tx.SendBatch(ctx, b).Close(); err != nil {
			return fmt.Errorf("insert evidence %s: %w", a.MeterID, err)
		}
	}
	for _, m := range res.Meters {
		if _, err := tx.Exec(ctx, `UPDATE meters SET status = $2 WHERE meter_id = $1`, m.MeterID, string(m.Status)); err != nil {
			return err
		}
	}
	sb, _ := json.Marshal(summary)
	stb, _ := json.Marshal(steps)
	if _, err := tx.Exec(ctx, `UPDATE analyses SET status = 'COMPLETED', steps = $2, summary = $3, finished_at = now(), error_message = NULL WHERE id = $1`, id, stb, sb); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func scanAnalysis(row pgx.Row) (*Analysis, error) {
	var a Analysis
	var steps, summary []byte
	if err := row.Scan(&a.ID, &a.Status, &steps, &summary, &a.ErrorMessage, &a.StartedAt, &a.FinishedAt, &a.CreatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(steps, &a.Steps)
	if len(summary) > 0 {
		var sm domain.Summary
		if json.Unmarshal(summary, &sm) == nil {
			a.Summary = &sm
		}
	}
	return &a, nil
}

const analysisCols = `id, status, steps, summary, error_message, started_at, finished_at, created_at`

// GetAnalysis devuelve un análisis o nil.
func (s *Store) GetAnalysis(ctx context.Context, id uuid.UUID) (*Analysis, error) {
	a, err := scanAnalysis(s.pool.QueryRow(ctx, `SELECT `+analysisCols+` FROM analyses WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// LastAnalysis devuelve el análisis más reciente por created_at (cualquier estado), o nil.
func (s *Store) LastAnalysis(ctx context.Context) (*Analysis, error) {
	a, err := scanAnalysis(s.pool.QueryRow(ctx, `SELECT `+analysisCols+` FROM analyses ORDER BY created_at DESC LIMIT 1`))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// LastCompletedAnalysisID devuelve el id del último análisis COMPLETED.
func (s *Store) LastCompletedAnalysisID(ctx context.Context) (*uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM analyses WHERE status = 'COMPLETED' ORDER BY finished_at DESC LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
