// Package store es la única capa que ejecuta SQL (spec backend §3).
package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // driver pgx5://
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/migrations"
)

// Store envuelve el pool de PostgreSQL y la caché en memoria del dataset.
type Store struct {
	pool *pgxpool.Pool

	mu           sync.RWMutex
	dataset      *domain.Dataset
	baselineDays int
}

// ConnectOptions controla los reintentos de conexión (CB-M02: cada 2 s hasta 30 intentos).
type ConnectOptions struct {
	Attempts int
	Interval time.Duration
}

// Connect abre el pool con reintentos.
func Connect(ctx context.Context, url string, opts ConnectOptions, log *slog.Logger) (*Store, error) {
	if opts.Attempts <= 0 {
		opts.Attempts = 30
	}
	if opts.Interval <= 0 {
		opts.Interval = 2 * time.Second
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.HealthCheckPeriod = 15 * time.Second
	var last error
	for attempt := 1; attempt <= opts.Attempts; attempt++ {
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = pool.Ping(pctx)
			cancel()
			if err == nil {
				return &Store{pool: pool, baselineDays: domain.DefaultBaselineDays}, nil
			}
			pool.Close()
		}
		last = err
		log.Warn("db retry", "attempt", attempt, "error", err.Error())
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(opts.Interval):
		}
	}
	return nil, fmt.Errorf("DB_UNAVAILABLE: %w", last)
}

// Close cierra el pool.
func (s *Store) Close() { s.pool.Close() }

// Ping ejecuta SELECT 1.
func (s *Store) Ping(ctx context.Context) error {
	var one int
	return s.pool.QueryRow(ctx, "SELECT 1").Scan(&one)
}

// SetBaselineDays fija BASELINE_DAYS para los cálculos derivados.
func (s *Store) SetBaselineDays(n int) {
	if n >= 2 {
		s.baselineDays = n
	}
}

// Migrate aplica las migraciones embebidas (RF-B-01, CB-B-15).
func Migrate(databaseURL string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	url := databaseURL
	for _, p := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(url, p) {
			url = "pgx5://" + strings.TrimPrefix(url, p)
			break
		}
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// Pool expone el pool (solo para tests).
func (s *Store) Pool() *pgxpool.Pool { return s.pool }
