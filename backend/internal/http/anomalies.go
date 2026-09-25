package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
	"github.com/jsotelo/energy-platform/backend/internal/store"
)

type anomalyItemDTO struct {
	ID                string               `json:"id"`
	AnalysisID        string               `json:"analysis_id"`
	Superseded        bool                 `json:"superseded"`
	MeterID           string               `json:"meter_id"`
	Type              domain.AnomalyType   `json:"type"`
	Severity          domain.Severity      `json:"severity"`
	Confidence        float64              `json:"confidence"`
	PriorityRank      int                  `json:"priority_rank"`
	Reason            string               `json:"reason"`
	RecommendedAction string               `json:"recommended_action"`
	Status            domain.AnomalyStatus `json:"status"`
	DetectedAt        time.Time            `json:"detected_at"`
	WindowStart       string               `json:"window_start"`
	WindowEnd         string               `json:"window_end"`
}

type anomalyDetailDTO struct {
	anomalyItemDTO
	ExplanationSource string                `json:"explanation_source"`
	ExplanationPoints []string              `json:"explanation_points"`
	RelatedEvent      *eventDTO             `json:"related_event"`
	Comparison        domain.Comparison     `json:"comparison"`
	Evidence          []engine.EvidenceItem `json:"evidence"`
}

func anomalyItem(a store.Anomaly) anomalyItemDTO {
	return anomalyItemDTO{ID: a.ID.String(), AnalysisID: a.AnalysisID.String(), Superseded: a.Superseded, MeterID: a.MeterID,
		Type: a.Type, Severity: a.Severity, Confidence: a.Confidence, PriorityRank: a.PriorityRank, Reason: a.Reason,
		RecommendedAction: a.RecommendedAction, Status: a.Status, DetectedAt: a.DetectedAt.UTC(),
		WindowStart: domain.ISODate(a.WindowStart), WindowEnd: domain.ISODate(a.WindowEnd)}
}

func validateMulti(w http.ResponseWriter, r *http.Request, param string, allowed []string) ([]string, bool) {
	vals := r.URL.Query()[param]
	for _, v := range vals {
		if !domain.Contains(allowed, v) {
			invalidQuery(w, param, allowed, "")
			return nil, false
		}
	}
	return vals, true
}

// listAnomalies implementa GET /anomalies (RF-B-09).
func (s *Server) listAnomalies(w http.ResponseWriter, r *http.Request) {
	var f store.AnomalyFilter
	var ok bool
	if f.Types, ok = validateMulti(w, r, "type", domain.AnomalyTypes); !ok {
		return
	}
	if f.Severities, ok = validateMulti(w, r, "severity", domain.Severities); !ok {
		return
	}
	if f.Statuses, ok = validateMulti(w, r, "status", domain.AnomalyStatuses); !ok {
		return
	}
	switch v := r.URL.Query().Get("include_superseded"); v {
	case "", "false":
	case "true":
		f.IncludeSuperseded = true
	default:
		invalidQuery(w, "include_superseded", []string{"true", "false"}, "")
		return
	}
	ctx := r.Context()
	last, err := s.store.LastCompletedAnalysisID(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	items := []anomalyItemDTO{}
	var analysisID any
	if last != nil {
		analysisID = last.String()
		list, err := s.store.ListAnomalies(ctx, f)
		if err != nil {
			s.dbError(w, r, err)
			return
		}
		for _, a := range list {
			items = append(items, anomalyItem(a))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"analysis_id": analysisID, "items": items, "total": len(items)})
}

func (s *Server) loadAnomalyDetail(r *http.Request, id uuid.UUID) (*anomalyDetailDTO, error) {
	ctx := r.Context()
	a, err := s.store.GetAnomaly(ctx, id)
	if err != nil || a == nil {
		return nil, err
	}
	ds, err := s.store.Dataset(ctx)
	if err != nil {
		return nil, err
	}
	d := &anomalyDetailDTO{anomalyItemDTO: anomalyItem(*a), ExplanationSource: a.ExplanationSource,
		ExplanationPoints: a.ExplanationPoints, Evidence: a.Evidence}
	if a.RelatedEvent != nil {
		e := eventOut(*a.RelatedEvent)
		d.RelatedEvent = &e
	}
	if ms := ds.Meters[a.MeterID]; ms != nil {
		d.Comparison = ms.CompareAt(a.WindowEnd)
	}
	return d, nil
}

// getAnomaly implementa GET /anomalies/{id} (RF-B-10).
func (s *Server) getAnomaly(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, CodeAnomalyNotFound, "La anomalía no existe", nil)
		return
	}
	d, err := s.loadAnomalyDetail(r, id)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	if d == nil {
		writeError(w, http.StatusNotFound, CodeAnomalyNotFound, "La anomalía no existe", nil)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// patchAnomalyStatus implementa PATCH /anomalies/{id}/status (RF-B-15).
func (s *Server) patchAnomalyStatus(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	var body struct {
		Status *string `json:"status"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Status == nil || !domain.Contains(domain.AnomalyStatuses, *body.Status) {
		invalidBody(w, "status", domain.AnomalyStatuses, "")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, CodeAnomalyNotFound, "La anomalía no existe", nil)
		return
	}
	found, err := s.store.UpdateAnomalyStatus(r.Context(), id, domain.AnomalyStatus(*body.Status))
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, CodeAnomalyNotFound, "La anomalía no existe", nil)
		return
	}
	d, err := s.loadAnomalyDetail(r, id)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}
