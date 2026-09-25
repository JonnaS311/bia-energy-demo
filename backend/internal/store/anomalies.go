package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
)

// Anomaly es una fila de anomalies con su evidencia y evento relacionado.
type Anomaly struct {
	ID                uuid.UUID
	AnalysisID        uuid.UUID
	Superseded        bool
	MeterID           string
	Type              domain.AnomalyType
	Severity          domain.Severity
	Confidence        float64
	PriorityRank      int
	Reason            string
	RecommendedAction string
	Status            domain.AnomalyStatus
	DetectedAt        time.Time
	WindowStart       time.Time
	WindowEnd         time.Time
	ExplanationSource string
	ExplanationPoints []string
	RelatedEvent      *domain.Event
	Evidence          []engine.EvidenceItem
}

// AnomalyFilter son los filtros de GET /anomalies (OR dentro de cada lista, AND entre listas).
type AnomalyFilter struct {
	Types             []string
	Severities        []string
	Statuses          []string
	IncludeSuperseded bool
}

const anomalyCols = `a.id, a.analysis_id, a.superseded, a.meter_id, a.type, a.severity, a.confidence::float8, a.priority_rank,
	a.reason, a.recommended_action, a.status, a.detected_at, a.window_start, a.window_end, a.explanation_source, a.explanation_points,
	e.id, e.timestamp, e.type, e.description`

func scanAnomaly(row pgx.Row) (*Anomaly, error) {
	var a Anomaly
	var points []byte
	var evID *int64
	var evTS *time.Time
	var evType, evDesc *string
	if err := row.Scan(&a.ID, &a.AnalysisID, &a.Superseded, &a.MeterID, &a.Type, &a.Severity, &a.Confidence, &a.PriorityRank,
		&a.Reason, &a.RecommendedAction, &a.Status, &a.DetectedAt, &a.WindowStart, &a.WindowEnd, &a.ExplanationSource, &points,
		&evID, &evTS, &evType, &evDesc); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(points, &a.ExplanationPoints)
	if a.ExplanationPoints == nil {
		a.ExplanationPoints = []string{}
	}
	a.WindowStart = domain.DateOf(a.WindowStart)
	a.WindowEnd = domain.DateOf(a.WindowEnd)
	if evID != nil {
		a.RelatedEvent = &domain.Event{ID: *evID, MeterID: a.MeterID, Timestamp: evTS.UTC(), Type: *evType, Description: *evDesc}
	}
	return &a, nil
}

// ListAnomalies aplica filtros y orden de RF-B-09.
func (s *Store) ListAnomalies(ctx context.Context, f AnomalyFilter) ([]Anomaly, error) {
	var where []string
	var args []any
	add := func(col string, vals []string) {
		if len(vals) == 0 {
			return
		}
		args = append(args, vals)
		where = append(where, fmt.Sprintf("a.%s = ANY($%d)", col, len(args)))
	}
	if !f.IncludeSuperseded {
		where = append(where, "a.superseded = false")
	}
	add("type", f.Types)
	add("severity", f.Severities)
	add("status", f.Statuses)
	q := `SELECT ` + anomalyCols + ` FROM anomalies a LEFT JOIN events e ON e.id = a.related_event_id`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	if f.IncludeSuperseded {
		q += " ORDER BY a.detected_at DESC, a.priority_rank ASC"
	} else {
		q += " ORDER BY a.priority_rank ASC"
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Anomaly
	for rows.Next() {
		a, err := scanAnomaly(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// GetAnomaly devuelve una anomalía con evidencia, o nil (incluye supersedidas).
func (s *Store) GetAnomaly(ctx context.Context, id uuid.UUID) (*Anomaly, error) {
	a, err := scanAnomaly(s.pool.QueryRow(ctx, `SELECT `+anomalyCols+` FROM anomalies a LEFT JOIN events e ON e.id = a.related_event_id WHERE a.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT position, metric, observed::float8, baseline::float8, delta_pct::float8, unit, "window", detail
		FROM anomaly_evidence WHERE anomaly_id = $1 ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	a.Evidence = []engine.EvidenceItem{}
	for rows.Next() {
		var e engine.EvidenceItem
		if err := rows.Scan(&e.Position, &e.Metric, &e.Observed, &e.Baseline, &e.DeltaPct, &e.Unit, &e.Window, &e.Detail); err != nil {
			return nil, err
		}
		a.Evidence = append(a.Evidence, e)
	}
	return a, rows.Err()
}

// UpdateAnomalyStatus cambia el estado (RF-B-15). Devuelve false si no existe.
func (s *Store) UpdateAnomalyStatus(ctx context.Context, id uuid.UUID, status domain.AnomalyStatus) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE anomalies SET status = $2 WHERE id = $1`, id, string(status))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// CurrentAnomaliesByMeter devuelve la anomalía vigente de cada medidor.
func (s *Store) CurrentAnomaliesByMeter(ctx context.Context) (map[string]Anomaly, error) {
	list, err := s.ListAnomalies(ctx, AnomalyFilter{})
	if err != nil {
		return nil, err
	}
	out := map[string]Anomaly{}
	for _, a := range list {
		out[a.MeterID] = a
	}
	return out, nil
}
