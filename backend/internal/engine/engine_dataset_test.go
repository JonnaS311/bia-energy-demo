package engine_test

// Test de aceptación del motor sobre los CSV reales (spec de pruebas RF-T-02).
// Fuente del oráculo: sección 9 del PDF de la prueba técnica y spec-energy-motor-anomalias-ia-v1.md RF-E-10..13.
// Estos valores NO provienen del archivo reservado al evaluador (no presente en el repo).

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
	"github.com/jsotelo/energy-platform/backend/internal/ingest"
	"github.com/jsotelo/energy-platform/backend/internal/llm"
)

type oracleCase struct {
	MeterID     string
	Type        domain.AnomalyType // "" = sin anomalía
	Severity    domain.Severity
	Confidence  float64
	Rank        int
	WindowStart string
	WindowEnd   string
	MinEvidence int
}

var datasetOracle = []oracleCase{
	{"M-109", domain.RealAnomaly, domain.SeverityHigh, 0.98, 1, "2026-09-12", "2026-09-14", 5},
	{"M-112", domain.DataQuality, domain.SeverityHigh, 0.98, 2, "2026-09-13", "2026-09-14", 5},
	{"M-104", domain.ExplainableAnomaly, domain.SeverityMedium, 0.95, 3, "2026-09-11", "2026-09-14", 5},
	{"M-106", domain.FalsePositive, domain.SeverityLow, 0.80, 4, "2026-09-08", "2026-09-08", 5},
	{"M-101", "", "", 0, 0, "", "", 0}, {"M-102", "", "", 0, 0, "", "", 0},
	{"M-103", "", "", 0, 0, "", "", 0}, {"M-105", "", "", 0, 0, "", "", 0},
	{"M-107", "", "", 0, 0, "", "", 0}, {"M-108", "", "", 0, 0, "", "", 0},
	{"M-110", "", "", 0, 0, "", "", 0}, {"M-111", "", "", 0, 0, "", "", 0},
}

// LoadDataset lee data/readings.csv y data/events.csv del repositorio.
func LoadDataset(t testing.TB) engine.Input {
	t.Helper()
	dir, _ := filepath.Abs(filepath.Join("..", "..", "..", "data"))
	f, err := os.Open(filepath.Join(dir, "readings.csv"))
	if err != nil {
		t.Fatalf("dataset not found: %s", dir)
	}
	defer f.Close()
	rr, err := ingest.ParseReadings(f)
	if err != nil {
		t.Fatal(err)
	}
	g, err := os.Open(filepath.Join(dir, "events.csv"))
	if err != nil {
		t.Fatalf("dataset not found: %s", dir)
	}
	defer g.Close()
	er, err := ingest.ParseEvents(g)
	if err != nil {
		t.Fatal(err)
	}
	for i := range er.Rows {
		er.Rows[i].ID = int64(i + 1)
	}
	return engine.Input{Readings: rr.Rows, Events: er.Rows}
}

func TestEngineDataset(t *testing.T) {
	in := LoadDataset(t)
	var steps []string
	start := time.Now()
	res, err := engine.New(llm.TemplateExplainer{}).Run(context.Background(), in, func(key string, status domain.StepStatus, detail string) error {
		steps = append(steps, key+":"+string(status)+":"+detail)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("RNF-E-01: engine took %s (limit 2s)", d)
	}
	t.Logf("engine duration: %s (limit 2s)", time.Since(start))

	if len(steps) != 14 {
		t.Fatalf("onStep calls = %d, want 14: %v", len(steps), steps)
	}
	if want := "RECOMMENDATION:COMPLETED:4 anomalías · 2 requieren atención prioritaria"; steps[13] != want {
		t.Errorf("final step = %q, want %q", steps[13], want)
	}
	if want := "READINGS:COMPLETED:4.032 lecturas de 12 medidores"; steps[1] != want {
		t.Errorf("readings step = %q, want %q", steps[1], want)
	}

	byMeter := map[string]engine.Anomaly{}
	for _, a := range res.Anomalies {
		byMeter[a.MeterID] = a
	}
	for _, c := range datasetOracle {
		a, ok := byMeter[c.MeterID]
		if c.Type == "" {
			if ok {
				t.Errorf("%s: unexpected anomaly type = %s", c.MeterID, a.Type)
			}
			continue
		}
		if !ok {
			t.Errorf("%s: no anomaly, want %s", c.MeterID, c.Type)
			continue
		}
		if a.Type != c.Type {
			t.Errorf("%s: type = %s, want %s", c.MeterID, a.Type, c.Type)
		}
		if a.Severity != c.Severity {
			t.Errorf("%s: severity = %s, want %s", c.MeterID, a.Severity, c.Severity)
		}
		if a.Confidence != c.Confidence {
			t.Errorf("%s: confidence = %.2f, want %.2f", c.MeterID, a.Confidence, c.Confidence)
		}
		if a.PriorityRank != c.Rank {
			t.Errorf("%s: rank = %d, want %d", c.MeterID, a.PriorityRank, c.Rank)
		}
		if ws, we := domain.ISODate(a.WindowStart), domain.ISODate(a.WindowEnd); ws != c.WindowStart || we != c.WindowEnd {
			t.Errorf("%s: window = %s..%s, want %s..%s", c.MeterID, ws, we, c.WindowStart, c.WindowEnd)
		}
		if len(a.Evidence) < c.MinEvidence {
			t.Errorf("%s: evidence = %d items, want ≥ %d", c.MeterID, len(a.Evidence), c.MinEvidence)
		}
		for i, e := range a.Evidence {
			if e.Position != i+1 || e.Detail == "" {
				t.Errorf("%s: evidence[%d] position=%d detail=%q", c.MeterID, i, e.Position, e.Detail)
			}
		}
		if a.RecommendedAction != engine.ActionDefault[c.Type] {
			t.Errorf("%s: action = %q", c.MeterID, a.RecommendedAction)
		}
		if a.Reason == "" || len(a.ExplanationPoints) < 2 || a.ExplanationSource != "template" {
			t.Errorf("%s: explanation reason=%q points=%d source=%s", c.MeterID, a.Reason, len(a.ExplanationPoints), a.ExplanationSource)
		}
	}

	// Estado derivado de los medidores (tabla 8.3).
	status := map[string]domain.MeterStatus{}
	for _, m := range res.Meters {
		status[m.MeterID] = m.Status
	}
	if status["M-109"] != domain.MeterCritical || status["M-104"] != domain.MeterAlert || status["M-112"] != domain.MeterAlert || status["M-106"] != domain.MeterOK || status["M-101"] != domain.MeterOK {
		t.Errorf("meter status = %v", status)
	}
}

// RF-E-14: evidencia de M-109 y M-112 contra los ejemplos del spec (tolerancias ±0,1 / ±0,2).
func TestEngineEvidenceExamples(t *testing.T) {
	res, err := engine.New(llm.TemplateExplainer{}).Run(context.Background(), LoadDataset(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	get := func(meter string) engine.Anomaly {
		for _, a := range res.Anomalies {
			if a.MeterID == meter {
				return a
			}
		}
		t.Fatalf("no anomaly for %s", meter)
		return engine.Anomaly{}
	}
	near := func(p *float64, want, tol float64) bool { return p != nil && *p >= want-tol && *p <= want+tol }

	m109 := get("M-109")
	e := m109.Evidence
	if e[0].Metric != "DAILY_CONSUMPTION" || !near(e[0].Observed, 2207.6, 0.1) || !near(e[0].Baseline, 1048.8, 0.1) || !near(e[0].DeltaPct, 110.5, 0.2) || e[0].Window != "2026-09-14" {
		t.Errorf("M-109 evidence[0] = %+v", e[0])
	}
	if e[1].Metric != "PERSISTENCE_DAYS" || !near(e[1].Observed, 3, 0) || e[1].Window != "2026-09-12..2026-09-14" {
		t.Errorf("M-109 evidence[1] = %+v", e[1])
	}
	if e[2].Metric != "CURRENT_A" || !near(e[2].Observed, 420.5, 0.1) || !near(e[2].Baseline, 200.5, 0.1) {
		t.Errorf("M-109 evidence[2] = %+v", e[2])
	}
	if e[3].Metric != "POWER_FACTOR" || !near(e[3].Observed, 0.744, 0.001) || !near(e[3].Baseline, 0.939, 0.001) {
		t.Errorf("M-109 evidence[3] = %+v", e[3])
	}
	lastE := e[len(e)-1]
	if lastE.Metric != "RELATED_EVENT" || lastE.Window != "2026-09-11..2026-09-13" || m109.RelatedEvent != nil {
		t.Errorf("M-109 related = %+v %+v", lastE, m109.RelatedEvent)
	}
	if !near(&m109.Comparison.ConsumptionKWh.Observed, 2207.6, 0.1) || !near(&m109.Comparison.CurrentA.Observed, 420.5, 0.1) {
		t.Errorf("M-109 comparison = %+v", m109.Comparison)
	}

	m112 := get("M-112")
	e = m112.Evidence
	if len(e) != 5 {
		t.Fatalf("M-112 evidence = %d items: %+v", len(e), e)
	}
	if e[0].Metric != "VOLTAGE_OUT_OF_RANGE" || !near(e[0].Observed, 16, 0) || e[0].Window != "2026-09-13..2026-09-14" {
		t.Errorf("M-112 evidence[0] = %+v", e[0])
	}
	if e[1].Metric != "PHYSICAL_RATIO" || !near(e[1].Observed, 4.22, 0.01) || !near(e[1].DeltaPct, 322.0, 0.2) {
		t.Errorf("M-112 evidence[1] = %+v", e[1])
	}
	if e[2].Metric != "POWER_FACTOR" || !near(e[2].Observed, 0.58, 0.001) || !near(e[2].Baseline, 0.951, 0.001) {
		t.Errorf("M-112 evidence[2] = %+v", e[2])
	}
	if e[3].Metric != "DAILY_CONSUMPTION" || !near(e[3].Observed, 662.7, 0.1) || !near(e[3].Baseline, 662.2, 0.1) {
		t.Errorf("M-112 evidence[3] = %+v", e[3])
	}
	if e[4].Metric != "RELATED_EVENT" || m112.RelatedEvent == nil || m112.RelatedEvent.Type != "DATA_QUALITY" {
		t.Errorf("M-112 related = %+v", e[4])
	}

	m106 := get("M-106")
	if m106.RelatedEvent == nil || m106.RelatedEvent.Type != "SCHEDULED_OUTAGE" || !near(m106.Evidence[0].Observed, 852.4, 0.1) || !near(m106.Evidence[2].Observed, 161.6, 0.1) {
		t.Errorf("M-106 = %+v / %+v", m106.RelatedEvent, m106.Evidence)
	}
}

// RF-E-18: dos ejecuciones producen la misma clasificación y evidencia.
func TestEngineReproducible(t *testing.T) {
	in := LoadDataset(t)
	a, _ := engine.New(llm.TemplateExplainer{}).Run(context.Background(), in, nil)
	b, _ := engine.New(llm.TemplateExplainer{}).Run(context.Background(), in, nil)
	if len(a.Anomalies) != len(b.Anomalies) {
		t.Fatal("different anomaly count")
	}
	for i := range a.Anomalies {
		x, y := a.Anomalies[i], b.Anomalies[i]
		if x.MeterID != y.MeterID || x.Type != y.Type || x.Confidence != y.Confidence || x.PriorityRank != y.PriorityRank || len(x.Evidence) != len(y.Evidence) {
			t.Errorf("run differs at %d: %+v vs %+v", i, x, y)
		}
	}
}

// RF-T-02: un umbral de ratio más estricto rompe el oráculo (el test lo detecta).
func TestEngineDatasetDetectsThresholdRegression(t *testing.T) {
	in := LoadDataset(t)
	in.Thresholds = engine.Thresholds{RatioMax: 1.4}
	res, err := engine.New(llm.TemplateExplainer{}).Run(context.Background(), in, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range res.Anomalies {
		if a.MeterID == "M-109" && a.Type == domain.RealAnomaly {
			t.Fatal("with RatioMax=1.4, M-109 should no longer be REAL_ANOMALY")
		}
	}
}
