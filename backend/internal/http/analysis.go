package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
	"github.com/jsotelo/energy-platform/backend/internal/store"
)

// analyze implementa POST /ai/analyze (RF-B-11). Ignora el body y el Content-Type.
func (s *Server) analyze(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !s.analysisMu.TryLock() {
		id, _, _ := s.store.ActiveAnalysisID(ctx)
		writeError(w, http.StatusConflict, CodeAnalysisInProgress, "Ya hay un análisis en curso", map[string]any{"analysis_id": id.String()})
		return
	}
	id, err := s.store.CreateAnalysis(ctx)
	s.analysisMu.Unlock()
	var inProgress *store.ErrAnalysisInProgress
	if errors.As(err, &inProgress) {
		writeError(w, http.StatusConflict, CodeAnalysisInProgress, "Ya hay un análisis en curso", map[string]any{"analysis_id": inProgress.ID.String()})
		return
	}
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	s.jobs.Add(1)
	go s.runJob(id)
	writeJSON(w, http.StatusAccepted, map[string]any{"analysis_id": id.String(), "status": domain.AnalysisQueued, "created_at": time.Now().UTC().Format(time.RFC3339)})
}

// runJob ejecuta el pipeline fuera del ciclo de vida de la petición (RF-B-12).
func (s *Server) runJob(id uuid.UUID) {
	defer s.jobs.Done()
	ctx, cancel := context.WithTimeout(context.Background(), s.jobTimeout)
	defer cancel()
	log := s.log.With("analysis_id", id.String())
	steps := domain.NewSteps()
	current := -1

	fail := func(msg string) {
		if current >= 0 && steps[current].Status == domain.StepRunning {
			now := time.Now().UTC()
			steps[current].Status, steps[current].FinishedAt = domain.StepFailed, &now
			steps[current].Detail = &msg
		}
		for attempt := 1; attempt <= 3; attempt++ { // CB-B-12
			if err := s.store.FailAnalysis(context.Background(), id, steps, msg); err == nil {
				break
			} else if attempt < 3 {
				log.Error("DB_ERROR", "error", err.Error(), "attempt", attempt)
				time.Sleep(time.Second)
			}
		}
		log.Error("analysis_failed", "error", msg)
	}

	defer func() {
		if rec := recover(); rec != nil {
			fail(fmt.Sprintf("panic: %v", rec))
		}
	}()

	if err := s.store.MarkRunning(ctx, id); err != nil {
		fail(err.Error())
		return
	}
	ds, err := s.store.Dataset(ctx)
	if err != nil {
		fail(err.Error())
		return
	}
	in := engine.Input{Events: ds.Events, MeterIDs: ds.MeterIDs, DuplicatesByMeterDay: s.tracker.Get().DuplicatesByMeterDay,
		Thresholds: engine.Thresholds{BaselineDays: s.cfg.BaselineDays}}
	for _, mid := range ds.MeterIDs {
		in.Readings = append(in.Readings, ds.Meters[mid].Readings...)
	}

	onStep := func(key string, status domain.StepStatus, detail string) error {
		for i := range steps {
			if steps[i].Key != key {
				continue
			}
			now := time.Now().UTC()
			switch status {
			case domain.StepRunning:
				current = i
				steps[i].Status, steps[i].StartedAt = status, &now
			default:
				steps[i].Status, steps[i].FinishedAt = status, &now
				if detail != "" {
					d := detail
					steps[i].Detail = &d
				}
			}
			log.Info("analysis_step", "step", key, "status", status)
			return s.store.UpdateSteps(ctx, id, steps)
		}
		return nil
	}

	res, err := s.newEngine(log).Run(ctx, in, onStep)
	if err != nil {
		msg := err.Error()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			msg = fmt.Sprintf("analysis timeout after %s", s.jobTimeout)
		}
		fail(msg)
		return
	}

	summary := domain.Summary{AnomaliesDetected: len(res.Anomalies), MetersAnalyzed: len(ds.MeterIDs), ExplanationSource: "template"}
	llmCount := 0
	for _, a := range res.Anomalies {
		if a.Severity == domain.SeverityHigh {
			summary.HighPriority++
		}
		if a.ExplanationSource == "llm" {
			llmCount++
		}
	}
	if len(res.Anomalies) > 0 && llmCount == len(res.Anomalies) {
		summary.ExplanationSource = "llm"
	} else if llmCount > 0 {
		summary.ExplanationSource = "mixed"
	}
	if err := s.store.CompleteAnalysis(ctx, id, steps, res, summary); err != nil {
		fail(err.Error())
		return
	}
	log.Info("analysis_completed", "anomalies", summary.AnomaliesDetected, "high_priority", summary.HighPriority)
}

type analysisDTO struct {
	ID           string          `json:"id"`
	Status       string          `json:"status"`
	Steps        []domain.Step   `json:"steps"`
	Summary      *domain.Summary `json:"summary"`
	ErrorMessage *string         `json:"error_message"`
	StartedAt    *time.Time      `json:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at"`
}

// getAnalysis implementa GET /ai/analysis/{id} (RF-B-13).
func (s *Server) getAnalysis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, CodeAnalysisNotFound, "El análisis no existe", nil)
		return
	}
	a, err := s.store.GetAnalysis(r.Context(), id)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	if a == nil {
		writeError(w, http.StatusNotFound, CodeAnalysisNotFound, "El análisis no existe", nil)
		return
	}
	writeJSON(w, http.StatusOK, analysisDTO{ID: a.ID.String(), Status: string(a.Status), Steps: a.Steps, Summary: a.Summary,
		ErrorMessage: a.ErrorMessage, StartedAt: a.StartedAt, FinishedAt: a.FinishedAt})
}
