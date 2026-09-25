package http

import (
	"context"
	"net/http"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// dashboard implementa GET /dashboard/summary (RF-B-14, tabla 8.4 del maestro).
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	meters, err := s.store.ListMeters(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	total, err := s.store.PeriodTotal(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	daily, err := s.store.GlobalDailyTotals(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	ds, err := s.store.Dataset(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	last, err := s.store.LastAnalysis(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	byStatus := map[string]int{"OK": 0, "ALERT": 0, "CRITICAL": 0}
	for _, m := range meters {
		byStatus[string(m.Status)]++
	}
	anomalies, err := s.currentAnomalies(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	high := 0
	sum := 0.0
	top := []map[string]any{}
	for i, a := range anomalies {
		if a.Severity == domain.SeverityHigh {
			high++
		}
		sum += a.Confidence
		if i < 3 {
			top = append(top, map[string]any{"id": a.ID.String(), "meter_id": a.MeterID, "type": a.Type, "severity": a.Severity, "confidence": a.Confidence})
		}
	}
	var avg any
	if len(anomalies) > 0 {
		avg = domain.Round(sum/float64(len(anomalies)), 2)
	}
	dailyOut := []map[string]any{}
	for _, d := range daily {
		dailyOut = append(dailyOut, map[string]any{"date": domain.ISODate(d.Date), "consumption_kwh": domain.Round(d.ConsumptionKWh, 1)})
	}
	var lastOut any
	if last != nil {
		var fin *time.Time
		if last.FinishedAt != nil {
			f := last.FinishedAt.UTC()
			fin = &f
		}
		lastOut = map[string]any{"id": last.ID.String(), "status": last.Status, "finished_at": fin}
	}
	period := map[string]any{"start": nil, "end": nil}
	if ds.ReadingCount > 0 {
		period = map[string]any{"start": domain.ISODate(ds.PeriodStart), "end": domain.ISODate(ds.PeriodEnd)}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"meters_total":          len(meters),
		"total_consumption_kwh": domain.Round(total, 1),
		"period":                period,
		"anomalies_total":       len(anomalies),
		"high_priority_total":   high,
		"ai_confidence_avg":     avg,
		"meters_by_status":      byStatus,
		"last_analysis":         lastOut,
		"daily_consumption":     dailyOut,
		"top_anomalies":         top,
	})
}

type anomalyLite struct {
	ID         interface{ String() string }
	MeterID    string
	Type       domain.AnomalyType
	Severity   domain.Severity
	Confidence float64
}

func (s *Server) currentAnomalies(ctx context.Context) ([]anomalyLite, error) {
	last, err := s.store.LastCompletedAnalysisID(ctx)
	if err != nil || last == nil {
		return nil, err
	}
	m, err := s.store.CurrentAnomaliesByMeter(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]anomalyLite, 0, len(m))
	ordered := make([]anomalyLite, len(m)+1)
	for _, a := range m {
		if a.PriorityRank >= 1 && a.PriorityRank <= len(m) {
			ordered[a.PriorityRank] = anomalyLite{ID: a.ID, MeterID: a.MeterID, Type: a.Type, Severity: a.Severity, Confidence: a.Confidence}
		}
	}
	for _, a := range ordered[1:] {
		if a.MeterID != "" {
			list = append(list, a)
		}
	}
	return list, nil
}

// health implementa GET /health (RF-B-16). Sin auth.
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	st := s.tracker.Get()
	if err := s.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "db": "error", "ingest": st})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "db": "ok", "ingest": st})
}
