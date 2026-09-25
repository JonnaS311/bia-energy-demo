package store

import (
	"context"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// Cálculos derivados (spec de data §8.4, RF-D-08). La API y el motor consumen estas funciones;
// todas delegan en la implementación pura de internal/domain sobre la foto en memoria del dataset,
// que se invalida al terminar cada ingesta.

// Dataset devuelve la foto en memoria del dataset (cacheada por proceso).
func (s *Store) Dataset(ctx context.Context) (*domain.Dataset, error) {
	s.mu.RLock()
	ds := s.dataset
	s.mu.RUnlock()
	if ds != nil {
		return ds, nil
	}
	readings, err := s.LoadReadings(ctx)
	if err != nil {
		return nil, err
	}
	events, err := s.LoadEvents(ctx)
	if err != nil {
		return nil, err
	}
	meters, err := s.ListMeters(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(meters))
	for _, m := range meters {
		ids = append(ids, m.MeterID)
	}
	ds = domain.BuildDataset(readings, events, ids, s.baselineDays)
	s.mu.Lock()
	s.dataset = ds
	s.mu.Unlock()
	return ds, nil
}

// Invalidate descarta la caché del dataset.
func (s *Store) Invalidate() {
	s.mu.Lock()
	s.dataset = nil
	s.mu.Unlock()
}

func (s *Store) meter(ctx context.Context, meterID string) (*domain.MeterStats, *domain.Dataset, error) {
	ds, err := s.Dataset(ctx)
	if err != nil {
		return nil, nil, err
	}
	return ds.Meters[meterID], ds, nil
}

// Period devuelve el periodo del dataset (8.4.1).
func (s *Store) Period(ctx context.Context) (time.Time, time.Time, error) {
	ds, err := s.Dataset(ctx)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return ds.PeriodStart, ds.PeriodEnd, nil
}

// DailyBaseline devuelve el baseline diario (8.4.5).
func (s *Store) DailyBaseline(ctx context.Context, meterID string) (value float64, days int, insufficient bool, err error) {
	m, _, err := s.meter(ctx, meterID)
	if err != nil || m == nil {
		return 0, 0, true, err
	}
	return m.Baseline, m.BaselineDays, m.Insufficient, nil
}

// CurrentConsumption devuelve el consumo del último día completo (8.4.7).
func (s *Store) CurrentConsumption(ctx context.Context, meterID string) (*float64, time.Time, error) {
	m, _, err := s.meter(ctx, meterID)
	if err != nil || m == nil || m.Current() == nil {
		return nil, time.Time{}, err
	}
	c := m.Current()
	v := c.ConsumptionKWh
	return &v, c.Date, nil
}

// Variation devuelve la variación en porcentaje (8.4.8).
func (s *Store) Variation(ctx context.Context, meterID string) (*float64, error) {
	m, _, err := s.meter(ctx, meterID)
	if err != nil || m == nil {
		return nil, err
	}
	return m.VariationPct(), nil
}

// HourlyProfile devuelve el perfil horario baseline (8.4.6).
func (s *Store) HourlyProfile(ctx context.Context, meterID string) ([24]domain.HourStat, error) {
	m, _, err := s.meter(ctx, meterID)
	if err != nil || m == nil {
		return [24]domain.HourStat{}, err
	}
	return m.Profile, nil
}

// PeriodTotal devuelve el consumo total del periodo (8.4.12), calculado en SQL.
func (s *Store) PeriodTotal(ctx context.Context) (float64, error) {
	var v float64
	err := s.pool.QueryRow(ctx, `SELECT coalesce(sum(consumption_kwh), 0)::float8 FROM readings`).Scan(&v)
	return v, err
}

// GlobalDailyTotals devuelve el consumo diario sumado de todos los medidores (8.4.12), en SQL.
func (s *Store) GlobalDailyTotals(ctx context.Context) ([]domain.DailyTotal, error) {
	rows, err := s.pool.Query(ctx, `SELECT (timestamp AT TIME ZONE 'UTC')::date AS d, sum(consumption_kwh)::float8
		FROM readings GROUP BY 1 ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DailyTotal
	for rows.Next() {
		var d domain.DailyTotal
		if err := rows.Scan(&d.Date, &d.ConsumptionKWh); err != nil {
			return nil, err
		}
		d.Date = time.Date(d.Date.Year(), d.Date.Month(), d.Date.Day(), 0, 0, 0, 0, time.UTC)
		out = append(out, d)
	}
	return out, rows.Err()
}
