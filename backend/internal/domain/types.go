// Package domain contiene los tipos de dominio, enums, la derivación del estado de medidor
// (tabla 8.3 del maestro) y los cálculos derivados puros (spec de data §8.4).
package domain

import "time"

type MeterStatus string

const (
	MeterOK       MeterStatus = "OK"
	MeterAlert    MeterStatus = "ALERT"
	MeterCritical MeterStatus = "CRITICAL"
)

var MeterStatuses = []string{string(MeterOK), string(MeterAlert), string(MeterCritical)}

type AnomalyType string

const (
	RealAnomaly        AnomalyType = "REAL_ANOMALY"
	ExplainableAnomaly AnomalyType = "EXPLAINABLE_ANOMALY"
	FalsePositive      AnomalyType = "FALSE_POSITIVE"
	DataQuality        AnomalyType = "DATA_QUALITY"
)

var AnomalyTypes = []string{string(RealAnomaly), string(ExplainableAnomaly), string(FalsePositive), string(DataQuality)}

type Severity string

const (
	SeverityHigh   Severity = "HIGH"
	SeverityMedium Severity = "MEDIUM"
	SeverityLow    Severity = "LOW"
)

var Severities = []string{string(SeverityHigh), string(SeverityMedium), string(SeverityLow)}

type AnomalyStatus string

const (
	AnomalyOpen          AnomalyStatus = "OPEN"
	AnomalyInvestigating AnomalyStatus = "INVESTIGATING"
	AnomalyResolved      AnomalyStatus = "RESOLVED"
	AnomalyDismissed     AnomalyStatus = "DISMISSED"
)

var AnomalyStatuses = []string{string(AnomalyOpen), string(AnomalyInvestigating), string(AnomalyResolved), string(AnomalyDismissed)}

type AnalysisStatus string

const (
	AnalysisQueued    AnalysisStatus = "QUEUED"
	AnalysisRunning   AnalysisStatus = "RUNNING"
	AnalysisCompleted AnalysisStatus = "COMPLETED"
	AnalysisFailed    AnalysisStatus = "FAILED"
)

type StepStatus string

const (
	StepPending   StepStatus = "PENDING"
	StepRunning   StepStatus = "RUNNING"
	StepCompleted StepStatus = "COMPLETED"
	StepFailed    StepStatus = "FAILED"
)

// Step es una etapa del pipeline tal como se serializa en analyses.steps (maestro 8.2.7).
type Step struct {
	Key        string     `json:"key"`
	Label      string     `json:"label"`
	Status     StepStatus `json:"status"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Detail     *string    `json:"detail"`
}

// StepDefs son las 7 etapas del pipeline con claves y labels exactos del maestro 8.2.7.
var StepDefs = []struct{ Key, Label string }{
	{"READINGS", "Lecturas"},
	{"BASELINE", "Baseline"},
	{"DETECTION", "Detección"},
	{"CORRELATION", "Correlación"},
	{"EVENTS", "Eventos"},
	{"EXPLANATION", "Explicación"},
	{"RECOMMENDATION", "Recomendación"},
}

// NewSteps devuelve las 7 etapas en estado PENDING.
func NewSteps() []Step {
	steps := make([]Step, len(StepDefs))
	for i, d := range StepDefs {
		steps[i] = Step{Key: d.Key, Label: d.Label, Status: StepPending}
	}
	return steps
}

// Summary es el resumen de un análisis completado (maestro 8.2.7).
type Summary struct {
	AnomaliesDetected int    `json:"anomalies_detected"`
	HighPriority      int    `json:"high_priority"`
	MetersAnalyzed    int    `json:"meters_analyzed"`
	ExplanationSource string `json:"explanation_source"`
}

// Reading es una lectura horaria.
type Reading struct {
	MeterID        string
	Timestamp      time.Time
	ConsumptionKWh float64
	VoltageV       float64
	CurrentA       float64
	PowerFactor    float64
	Status         string
}

// Event es un evento operativo.
type Event struct {
	ID          int64     `json:"id"`
	MeterID     string    `json:"-"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
}

// DeriveMeterStatus aplica la tabla 8.3 del maestro. anomalyType vacío = sin anomalía vigente.
func DeriveMeterStatus(anomalyType AnomalyType, severity Severity) MeterStatus {
	switch anomalyType {
	case "", FalsePositive:
		return MeterOK
	case RealAnomaly:
		if severity == SeverityHigh {
			return MeterCritical
		}
		return MeterAlert
	default:
		return MeterAlert
	}
}

// Contains indica si v pertenece a allowed.
func Contains(allowed []string, v string) bool {
	for _, a := range allowed {
		if a == v {
			return true
		}
	}
	return false
}
