package engine_test

// Casos borde con datos sintéticos generados en memoria (spec de pruebas RF-T-03): mismos
// escenarios que los fixtures CSV del catálogo, construidos de forma determinística.

import (
	"context"
	"testing"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
	"github.com/jsotelo/energy-platform/backend/internal/llm"
)

var day0 = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// profileKWh es un perfil diario determinístico (sin ruido): 30 kWh de noche, 40 de día.
func profileKWh(h int) float64 {
	if h >= 8 && h < 18 {
		return 40
	}
	return 30
}

type dayMod func(h int, r *domain.Reading)

// synth genera `days` días completos de un medidor sano con un pequeño ruido determinístico,
// aplicando mods por día (índice de día → modificador).
func synth(meter string, days int, mods map[int]dayMod) []domain.Reading {
	var out []domain.Reading
	for d := 0; d < days; d++ {
		for h := 0; h < 24; h++ {
			kwh := profileKWh(h) * (1 + 0.01*float64((d*7+h*3)%5-2))
			r := domain.Reading{MeterID: meter, Timestamp: day0.AddDate(0, 0, d).Add(time.Duration(h) * time.Hour),
				ConsumptionKWh: kwh, VoltageV: 220 + float64((h%3)-1), PowerFactor: 0.94}
			r.CurrentA = kwh * 1000 / (r.VoltageV * r.PowerFactor)
			if m, ok := mods[d]; ok {
				m(h, &r)
			}
			out = append(out, r)
		}
	}
	return out
}

func scale(f float64) dayMod {
	return func(_ int, r *domain.Reading) {
		r.ConsumptionKWh *= f
		r.CurrentA *= f
	}
}

func run(t *testing.T, readings []domain.Reading, events []domain.Event) map[string]engine.Anomaly {
	t.Helper()
	res, err := engine.New(llm.TemplateExplainer{}).Run(context.Background(), engine.Input{Readings: readings, Events: events}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]engine.Anomaly{}
	for _, a := range res.Anomalies {
		out[a.MeterID] = a
	}
	return out
}

// RF-E-10: caída del 40 % sin eventos → REAL_ANOMALY (no FALSE_POSITIVE).
func TestDropWithoutEventIsReal(t *testing.T) {
	got := run(t, synth("M-201", 14, map[int]dayMod{10: scale(0.6)}), nil)
	a, ok := got["M-201"]
	if !ok || a.Type != domain.RealAnomaly || a.Severity != domain.SeverityMedium {
		t.Fatalf("got %+v", a)
	}
}

// CB-E-10: subida del 40 % con SCHEDULED_OUTAGE → REAL_ANOMALY (una parada no explica una subida).
func TestRiseWithOutageIsReal(t *testing.T) {
	ev := []domain.Event{{ID: 1, MeterID: "M-202", Timestamp: day0.AddDate(0, 0, 10), Type: "SCHEDULED_OUTAGE"}}
	got := run(t, synth("M-202", 14, map[int]dayMod{10: scale(1.4)}), ev)
	if a := got["M-202"]; a.Type != domain.RealAnomaly {
		t.Fatalf("got %+v", a)
	}
}

// CB-E-11: caída del 30 % durante 4 días con SCHEDULED_OUTAGE el día 1 → REAL_ANOMALY.
func TestLongDropWithOutageIsReal(t *testing.T) {
	ev := []domain.Event{{ID: 1, MeterID: "M-203", Timestamp: day0.AddDate(0, 0, 9), Type: "SCHEDULED_OUTAGE"}}
	mods := map[int]dayMod{9: scale(0.7), 10: scale(0.7), 11: scale(0.7), 12: scale(0.7)}
	if a := run(t, synth("M-203", 14, mods), ev)["M-203"]; a.Type != domain.RealAnomaly {
		t.Fatalf("got %+v", a)
	}
}

// Caída corta con parada programada → FALSE_POSITIVE LOW.
func TestShortDropWithOutageIsFalsePositive(t *testing.T) {
	ev := []domain.Event{{ID: 1, MeterID: "M-204", Timestamp: day0.AddDate(0, 0, 9), Type: "SCHEDULED_OUTAGE"}}
	a := run(t, synth("M-204", 14, map[int]dayMod{9: scale(0.6)}), ev)["M-204"]
	if a.Type != domain.FalsePositive || a.Severity != domain.SeverityLow {
		t.Fatalf("got %+v", a)
	}
}

// CB-E-12: días DQ + OPERATIONAL_CHANGE → DATA_QUALITY (regla 1 gana).
func TestDataQualityWinsOverOperationalChange(t *testing.T) {
	mods := map[int]dayMod{11: func(h int, r *domain.Reading) {
		r.ConsumptionKWh *= 1.5
		r.CurrentA *= 1.5
		if h%4 == 0 {
			r.VoltageV = 245
		}
	}}
	ev := []domain.Event{{ID: 1, MeterID: "M-205", Timestamp: day0.AddDate(0, 0, 11), Type: "OPERATIONAL_CHANGE"}}
	a := run(t, synth("M-205", 14, mods), ev)["M-205"]
	if a.Type != domain.DataQuality || a.Severity != domain.SeverityHigh {
		t.Fatalf("got %+v", a)
	}
}

// RF-E-08 (d): un día con 20 lecturas es día DQ.
func TestGapDayIsDataQuality(t *testing.T) {
	rs := synth("M-206", 14, nil)
	var filtered []domain.Reading
	for _, r := range rs {
		if domain.DateOf(r.Timestamp).Equal(day0.AddDate(0, 0, 10)) && r.Timestamp.Hour() >= 20 {
			continue
		}
		filtered = append(filtered, r)
	}
	if a := run(t, filtered, nil)["M-206"]; a.Type != domain.DataQuality {
		t.Fatalf("got %+v", a)
	}
}

// RF-E-03 / CB-E-01: medidor con 30 lecturas → datos insuficientes, sin anomalía.
func TestShortMeterInsufficient(t *testing.T) {
	rs := synth("M-207", 2, nil)[:30]
	res, err := engine.New(llm.TemplateExplainer{}).Run(context.Background(), engine.Input{Readings: rs}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Anomalies) != 0 || !res.Meters[0].InsufficientData {
		t.Fatalf("got %+v", res)
	}
}

// RF-E-06: día sin spike con 8 lecturas al 150 % del perfil → patrón horario, REAL_ANOMALY MEDIUM.
func TestHourlyPatternCandidate(t *testing.T) {
	mods := map[int]dayMod{11: func(h int, r *domain.Reading) {
		if h < 8 {
			r.ConsumptionKWh *= 1.5
			r.CurrentA *= 1.5
		} else if h < 16 {
			r.ConsumptionKWh *= 0.75
			r.CurrentA *= 0.75
		}
	}}
	a, ok := run(t, synth("M-208", 14, mods), nil)["M-208"]
	if !ok || a.Type != domain.RealAnomaly || a.Severity != domain.SeverityMedium {
		t.Fatalf("got %+v ok=%v", a, ok)
	}
}

// CB-E-06: dos eventos a 0 h y 12 h del inicio → gana el de 0 h.
func TestNearestEventWins(t *testing.T) {
	ev := []domain.Event{
		{ID: 1, MeterID: "M-209", Timestamp: day0.AddDate(0, 0, 10).Add(12 * time.Hour), Type: "SCHEDULED_OUTAGE"},
		{ID: 2, MeterID: "M-209", Timestamp: day0.AddDate(0, 0, 10), Type: "OPERATIONAL_CHANGE"},
	}
	a := run(t, synth("M-209", 14, map[int]dayMod{10: scale(1.4), 11: scale(1.4)}), ev)["M-209"]
	if a.RelatedEvent == nil || a.RelatedEvent.ID != 2 || a.Type != domain.ExplainableAnomaly {
		t.Fatalf("got %+v", a)
	}
}

// CB-E-07/08: evento a > 24 h o UNKNOWN no cuentan.
func TestFarOrUnknownEventIgnored(t *testing.T) {
	ev := []domain.Event{
		{ID: 1, MeterID: "M-210", Timestamp: day0.AddDate(0, 0, 7), Type: "OPERATIONAL_CHANGE"},
		{ID: 2, MeterID: "M-210", Timestamp: day0.AddDate(0, 0, 10), Type: "UNKNOWN"},
	}
	a := run(t, synth("M-210", 14, map[int]dayMod{10: scale(1.4), 11: scale(1.4)}), ev)["M-210"]
	if a.RelatedEvent != nil || a.Type != domain.RealAnomaly {
		t.Fatalf("got %+v", a)
	}
}

// RF-E-12: REAL de un día, sin corroboración y max_dev 0,30 → confianza acotada a 0,50.
func TestConfidenceFloor(t *testing.T) {
	mods := map[int]dayMod{10: func(_ int, r *domain.Reading) { r.ConsumptionKWh *= 1.3 }} // corriente sin cambio
	a := run(t, synth("M-211", 14, mods), nil)["M-211"]
	if a.Confidence != 0.50 {
		t.Fatalf("confidence = %.2f", a.Confidence)
	}
}

// Una etapa que falla por contexto cancelado detiene el pipeline.
func TestPipelineContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := engine.New(llm.TemplateExplainer{}).Run(ctx, engine.Input{Readings: synth("M-212", 3, nil)}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
