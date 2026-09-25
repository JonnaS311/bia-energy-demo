// Punto de entrada: solo cablea dependencias (spec backend RF-B-01, RF-B-02).
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/jsotelo/energy-platform/backend/internal/config"
	apihttp "github.com/jsotelo/energy-platform/backend/internal/http"
	"github.com/jsotelo/energy-platform/backend/internal/ingest"
	"github.com/jsotelo/energy-platform/backend/internal/store"
)

func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}

func main() {
	seedOnly := flag.Bool("seed-only", false, "ejecuta migraciones e ingesta y termina (make seed)")
	flag.Parse()
	os.Exit(run(*seedOnly))
}

func run(seedOnly bool) int {
	cfg, err := config.Load()
	log := newLogger(cfg.LogLevel)
	if err != nil {
		var ie *config.InvalidError
		if errors.As(err, &ie) {
			log.Error("CONFIG_INVALID", "field", ie.Field)
		} else {
			log.Error("CONFIG_INVALID", "error", err.Error())
		}
		return 1
	}
	if cfg.CORSDefaulted {
		log.Warn("CORS_DEFAULT_ORIGIN", "origin", cfg.CORSOrigins[0])
	}
	if cfg.BaselineInvalid {
		log.Warn("CONFIG_INVALID", "field", "BASELINE_DAYS", "using", 7)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// (2) BD con reintentos.
	st, err := store.Connect(ctx, cfg.DatabaseURL, store.ConnectOptions{}, log)
	if err != nil {
		log.Error("DB_UNAVAILABLE", "error", err.Error())
		return 1
	}
	defer st.Close()
	st.SetBaselineDays(cfg.BaselineDays)

	// (3) Migraciones.
	if err := store.Migrate(cfg.DatabaseURL); err != nil {
		log.Error("MIGRATION_FAILED", "error", err.Error())
		return 1
	}
	// (4) Análisis huérfanos.
	if n, err := st.FailOrphanAnalyses(ctx); err != nil {
		log.Error("DB_ERROR", "error", err.Error())
	} else if n > 0 {
		log.Warn("orphan_analyses_failed", "count", n)
	}
	// (5) Ingesta.
	tracker := ingest.NewTracker()
	ingest.Run(ctx, st, cfg.DataDir, tracker, log)
	// (6) Usuario demo.
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.DemoUserPassword), 10)
	if err == nil {
		err = st.SeedUser(ctx, cfg.DemoUserEmail, "Analista Demo", string(hash))
	}
	if err != nil {
		log.Error("DB_ERROR", "error", err.Error())
		return 1
	}
	if seedOnly {
		log.Info("seed_completed")
		return 0
	}

	// (7) HTTP con graceful shutdown.
	srv := apihttp.NewServer(cfg, st, tracker, log, apihttp.Options{})
	httpSrv := &http.Server{
		Addr: ":" + cfg.Port, Handler: srv.Router(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
	case err := <-errCh:
		log.Error("listen_failed", "error", err.Error())
		return 1
	}
	log.Info("shutting_down")
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(sctx); err != nil {
		log.Error("shutdown_error", "error", err.Error())
	}
	return 0
}
