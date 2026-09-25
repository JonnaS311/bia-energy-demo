package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

const batchSize = 500

// UpsertMeters crea los medidores sin sobrescribir su status (RF-D-06).
func (s *Store) UpsertMeters(ctx context.Context, meterIDs []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	b := &pgx.Batch{}
	for _, id := range meterIDs {
		b.Queue(`INSERT INTO meters (meter_id, name, location, status) VALUES ($1, $2, 'Planta principal', 'OK') ON CONFLICT (meter_id) DO NOTHING`, id, "Medidor "+id)
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpsertReadings escribe las lecturas en una transacción, en lotes de 500 (RF-D-05, RNF-D-02).
func (s *Store) UpsertReadings(ctx context.Context, rows []domain.Reading) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for start := 0; start < len(rows); start += batchSize {
		end := min(start+batchSize, len(rows))
		b := &pgx.Batch{}
		for _, r := range rows[start:end] {
			b.Queue(`INSERT INTO readings (meter_id, timestamp, consumption_kwh, voltage_v, current_a, power_factor, status)
				VALUES ($1,$2,$3,$4,$5,$6,$7)
				ON CONFLICT (meter_id, timestamp) DO UPDATE SET consumption_kwh = EXCLUDED.consumption_kwh,
				  voltage_v = EXCLUDED.voltage_v, current_a = EXCLUDED.current_a, power_factor = EXCLUDED.power_factor, status = EXCLUDED.status`,
				r.MeterID, r.Timestamp, r.ConsumptionKWh, r.VoltageV, r.CurrentA, r.PowerFactor, r.Status)
		}
		if err := tx.SendBatch(ctx, b).Close(); err != nil {
			return fmt.Errorf("batch %d: %w", start/batchSize, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

// UpsertEvents escribe los eventos en una transacción (RF-D-05).
func (s *Store) UpsertEvents(ctx context.Context, rows []domain.Event) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	b := &pgx.Batch{}
	for _, e := range rows {
		b.Queue(`INSERT INTO events (meter_id, timestamp, type, description) VALUES ($1,$2,$3,$4)
			ON CONFLICT (meter_id, timestamp, type) DO UPDATE SET description = EXCLUDED.description`,
			e.MeterID, e.Timestamp, e.Type, e.Description)
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.Invalidate()
	return nil
}

// Counts devuelve los conteos de readings, events y meters.
func (s *Store) Counts(ctx context.Context) (readings, events, meters int, err error) {
	err = s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM readings), (SELECT count(*) FROM events), (SELECT count(*) FROM meters)`).Scan(&readings, &events, &meters)
	return
}

// LoadReadings lee todas las lecturas ordenadas.
func (s *Store) LoadReadings(ctx context.Context) ([]domain.Reading, error) {
	rows, err := s.pool.Query(ctx, `SELECT meter_id, timestamp, consumption_kwh::float8, voltage_v::float8, current_a::float8, power_factor::float8, status
		FROM readings ORDER BY meter_id, timestamp`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Reading
	for rows.Next() {
		var r domain.Reading
		if err := rows.Scan(&r.MeterID, &r.Timestamp, &r.ConsumptionKWh, &r.VoltageV, &r.CurrentA, &r.PowerFactor, &r.Status); err != nil {
			return nil, err
		}
		r.Timestamp = r.Timestamp.UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadEvents lee todos los eventos ordenados por timestamp.
func (s *Store) LoadEvents(ctx context.Context) ([]domain.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, meter_id, timestamp, type, description FROM events ORDER BY timestamp, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Event
	for rows.Next() {
		var e domain.Event
		if err := rows.Scan(&e.ID, &e.MeterID, &e.Timestamp, &e.Type, &e.Description); err != nil {
			return nil, err
		}
		e.Timestamp = e.Timestamp.UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}

// Meter es la fila de meters.
type Meter struct {
	MeterID  string
	Name     string
	Location string
	Status   domain.MeterStatus
}

// ListMeters devuelve todos los medidores ordenados por meter_id.
func (s *Store) ListMeters(ctx context.Context) ([]Meter, error) {
	rows, err := s.pool.Query(ctx, `SELECT meter_id, name, location, status FROM meters ORDER BY meter_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Meter
	for rows.Next() {
		var m Meter
		if err := rows.Scan(&m.MeterID, &m.Name, &m.Location, &m.Status); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetMeter devuelve un medidor por meter_id exacto, o nil.
func (s *Store) GetMeter(ctx context.Context, meterID string) (*Meter, error) {
	var m Meter
	err := s.pool.QueryRow(ctx, `SELECT meter_id, name, location, status FROM meters WHERE meter_id = $1`, meterID).Scan(&m.MeterID, &m.Name, &m.Location, &m.Status)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
