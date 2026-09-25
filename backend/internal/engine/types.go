// Package engine implementa el motor de anomalías determinístico (spec del motor RF-E-01..RF-E-19).
// No hace I/O: recibe estructuras en memoria y devuelve estructuras en memoria. La explicación en
// lenguaje natural se delega en un Explainer inyectado (implementado en internal/llm).
package engine

import (
	"context"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// Thresholds agrupa los umbrales del motor (RF-E-19). El valor cero de cada campo usa el default.
type Thresholds struct {
	DevSpike         float64
	DevHigh          float64
	ZOutlier         float64
	OutliersPerDay   int
	CurrentDelta     float64
	PFDrop           float64
	VoltageMin       float64
	VoltageMax       float64
	RatioMin         float64
	RatioMax         float64
	PFLow            float64
	DQStableDev      float64
	MinInconsistent  int
	EventWindowHours float64
	BaselineDays     int
}

// DefaultThresholds son los umbrales calibrados sobre el dataset entregado (spec del motor §9).
var DefaultThresholds = Thresholds{
	DevSpike: 0.20, DevHigh: 0.50, ZOutlier: 3.0, OutliersPerDay: 6, CurrentDelta: 0.20, PFDrop: 0.10,
	VoltageMin: 209, VoltageMax: 231, RatioMin: 0.5, RatioMax: 2.0, PFLow: 0.85, DQStableDev: 0.10,
	MinInconsistent: 3, EventWindowHours: 24, BaselineDays: domain.DefaultBaselineDays,
}

// WithDefaults completa los campos en cero con los valores por defecto.
func (t Thresholds) WithDefaults() Thresholds {
	d := DefaultThresholds
	pick := func(v, def float64) float64 {
		if v == 0 {
			return def
		}
		return v
	}
	pickI := func(v, def int) int {
		if v == 0 {
			return def
		}
		return v
	}
	return Thresholds{
		DevSpike: pick(t.DevSpike, d.DevSpike), DevHigh: pick(t.DevHigh, d.DevHigh), ZOutlier: pick(t.ZOutlier, d.ZOutlier),
		OutliersPerDay: pickI(t.OutliersPerDay, d.OutliersPerDay), CurrentDelta: pick(t.CurrentDelta, d.CurrentDelta),
		PFDrop: pick(t.PFDrop, d.PFDrop), VoltageMin: pick(t.VoltageMin, d.VoltageMin), VoltageMax: pick(t.VoltageMax, d.VoltageMax),
		RatioMin: pick(t.RatioMin, d.RatioMin), RatioMax: pick(t.RatioMax, d.RatioMax), PFLow: pick(t.PFLow, d.PFLow),
		DQStableDev: pick(t.DQStableDev, d.DQStableDev), MinInconsistent: pickI(t.MinInconsistent, d.MinInconsistent),
		EventWindowHours: pick(t.EventWindowHours, d.EventWindowHours), BaselineDays: pickI(t.BaselineDays, d.BaselineDays),
	}
}

// Input es la entrada del pipeline (spec del motor §8.0).
type Input struct {
	Readings []domain.Reading
	Events   []domain.Event
	MeterIDs []string
	// DuplicatesByMeterDay: duplicados reportados por la ingesta, por medidor y fecha YYYY-MM-DD.
	DuplicatesByMeterDay map[string]map[string]int
	Thresholds           Thresholds
}

// EvidenceItem es un ítem de evidencia estructurada (RF-E-14).
type EvidenceItem struct {
	Position int      `json:"position"`
	Metric   string   `json:"metric"`
	Observed *float64 `json:"observed"`
	Baseline *float64 `json:"baseline"`
	DeltaPct *float64 `json:"delta_pct"`
	Unit     string   `json:"unit"`
	Window   string   `json:"window"`
	Detail   string   `json:"detail"`
}

// Anomaly es una anomalía clasificada, priorizada y explicada.
type Anomaly struct {
	MeterID           string
	Type              domain.AnomalyType
	Severity          domain.Severity
	Confidence        float64
	PriorityRank      int
	WindowStart       time.Time
	WindowEnd         time.Time
	RelatedEvent      *domain.Event
	Reason            string
	RecommendedAction string
	ExplanationPoints []string
	ExplanationSource string // "llm" | "template"
	ActionDefault     string
	Evidence          []EvidenceItem
	Comparison        domain.Comparison
}

// MeterSummary resume el resultado por medidor.
type MeterSummary struct {
	MeterID          string
	InsufficientData bool
	Status           domain.MeterStatus
}

// Result es la salida del pipeline.
type Result struct {
	Anomalies []Anomaly // ordenadas por PriorityRank
	Meters    []MeterSummary
}

// StepCallback se invoca al iniciar (RUNNING) y terminar (COMPLETED/FAILED) cada etapa.
// Si devuelve error, el pipeline se detiene con ese error.
type StepCallback func(key string, status domain.StepStatus, detail string) error

// Facts son los valores que usan las plantillas de fallback (spec del motor §8.4).
type Facts struct {
	DeltaPct         float64
	Baseline         float64
	Observed         float64
	Days             int
	EventDescription string
	EventDate        string
	CurrentDeltaPct  float64
	PFObserved       float64
	PFBaseline       float64
	VOutCount        int
	RatioMax         float64
	PFMin            float64
	WindowStart      string
	WindowEnd        string
}

// ExplainInput es lo que recibe el Explainer: clasificación ya decidida + evidencia.
type ExplainInput struct {
	MeterID        string
	Type           domain.AnomalyType
	Severity       domain.Severity
	Confidence     float64
	WindowStart    string
	WindowEnd      string
	BaselineWindow string
	Comparison     domain.Comparison
	Evidence       []EvidenceItem
	RelatedEvent   *domain.Event
	ActionDefault  string
	Facts          Facts
}

// ExplainOutput es el texto generado. Source ∈ {"llm","template"}.
type ExplainOutput struct {
	Reason            string
	RecommendedAction string
	Points            []string
	Source            string
}

// Explainer redacta la explicación. Nunca devuelve error: ante cualquier fallo usa plantilla.
type Explainer interface {
	Explain(ctx context.Context, in ExplainInput) ExplainOutput
}

// ActionDefault es la acción recomendada por defecto por tipo (RF-E-15).
var ActionDefault = map[domain.AnomalyType]string{
	domain.RealAnomaly:        "Investigar el medidor y la instalación.",
	domain.DataQuality:        "Validar el sensor y la calidad de las lecturas del medidor.",
	domain.ExplainableAnomaly: "Validar con operaciones que el cambio corresponde al evento registrado.",
	domain.FalsePositive:      "No escalar; el cambio está explicado por la parada programada.",
}
