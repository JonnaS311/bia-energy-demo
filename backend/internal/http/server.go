package http

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/jsotelo/energy-platform/backend/internal/config"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
	"github.com/jsotelo/energy-platform/backend/internal/ingest"
	"github.com/jsotelo/energy-platform/backend/internal/llm"
	"github.com/jsotelo/energy-platform/backend/internal/store"
)

// Analyzer es el motor que ejecuta el job (sustituible por fakes en tests).
type Analyzer interface {
	Run(ctx context.Context, in engine.Input, onStep engine.StepCallback) (engine.Result, error)
}

// AnalyzerFactory crea un Analyzer por análisis (el Explainer LLM es por análisis, CB-E-16).
type AnalyzerFactory func(log *slog.Logger) Analyzer

// Server agrupa las dependencias de los handlers.
type Server struct {
	cfg        config.Config
	store      *store.Store
	tracker    *ingest.Tracker
	log        *slog.Logger
	newEngine  AnalyzerFactory
	jobTimeout time.Duration
	reqTimeout time.Duration

	analysisMu sync.Mutex
	jobs       sync.WaitGroup
	limiter    *loginLimiter
	dummyHash  []byte
}

// Options permite ajustar el servidor en tests.
type Options struct {
	Analyzer       AnalyzerFactory
	JobTimeout     time.Duration
	RequestTimeout time.Duration
}

// NewServer construye el servidor.
func NewServer(cfg config.Config, st *store.Store, tr *ingest.Tracker, log *slog.Logger, opts Options) *Server {
	s := &Server{cfg: cfg, store: st, tracker: tr, log: log, newEngine: opts.Analyzer,
		jobTimeout: opts.JobTimeout, reqTimeout: opts.RequestTimeout, limiter: newLoginLimiter()}
	if s.newEngine == nil {
		llmCfg := llm.Config{APIKey: cfg.DeepSeekAPIKey, Model: cfg.DeepSeekModel, BaseURL: cfg.DeepSeekBaseURL, Timeout: cfg.LLMTimeout}
		s.newEngine = func(l *slog.Logger) Analyzer { return engine.New(llm.NewExplainer(llmCfg, l)) }
	}
	if s.jobTimeout <= 0 {
		s.jobTimeout = 120 * time.Second
	}
	if s.reqTimeout <= 0 {
		s.reqTimeout = 30 * time.Second
	}
	// Hash fijo para igualar el tiempo de respuesta cuando el usuario no existe (RF-B-04).
	s.dummyHash, _ = bcrypt.GenerateFromPassword([]byte("no-user-placeholder"), 10)
	return s
}

// WaitJobs espera a que terminen los jobs en curso (tests y shutdown).
func (s *Server) WaitJobs() { s.jobs.Wait() }

// Router construye el router con la cadena de middleware de RF-B-03.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(s.requestID, s.accessLog, s.recoverer, s.cors, s.bodyLimit, s.timeout)
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, CodeNotFound, "Ruta no encontrada", nil)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "Método no permitido", nil)
	})

	r.Get("/health", s.health)
	r.Post("/auth/login", s.login)

	r.Group(func(r chi.Router) {
		r.Use(s.auth, s.ingestGate)
		r.Get("/meters", s.listMeters)
		r.Get("/meters/{meterId}", s.getMeter)
		r.Get("/meters/{meterId}/readings", s.getReadings)
		r.Get("/anomalies", s.listAnomalies)
		r.Get("/anomalies/{id}", s.getAnomaly)
		r.Patch("/anomalies/{id}/status", s.patchAnomalyStatus)
		r.Post("/ai/analyze", s.analyze)
		r.Get("/ai/analysis/{id}", s.getAnalysis)
		r.Get("/dashboard/summary", s.dashboard)
	})
	return r
}
