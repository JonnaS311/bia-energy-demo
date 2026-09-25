package llm

import (
	"encoding/json"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
)

// SystemPrompt es el prompt de sistema exacto (spec del motor §8.2). Contiene la palabra "JSON",
// requisito del JSON mode de DeepSeek.
const SystemPrompt = `Eres un analista senior de gestión energética. Recibes la clasificación ya decidida de una anomalía en un medidor eléctrico industrial, junto con la evidencia numérica que la sustenta.

Tu tarea es redactar, en español y para un operador de planta, una explicación breve y una acción recomendada.

Reglas obligatorias:
1. No cambies ni cuestiones la clasificación (type, severity, confidence): ya está decidida por un motor determinístico.
2. Usa únicamente los números que aparecen en "comparison" y "evidence". No inventes cifras, fechas ni causas que no estén en los datos.
3. Si "related_event" es null, di explícitamente que no hay evento operativo que explique el cambio.
4. Si "related_event" existe, menciónalo con su tipo y descripción y explica cómo se relaciona con el cambio.
5. "recommended_action" debe ser una sola frase imperativa coherente con "action_default"; puedes precisarla con datos de la evidencia, pero no cambiar su sentido (por ejemplo, no recomiendes escalar un FALSE_POSITIVE).
6. Responde ÚNICAMENTE con un objeto JSON válido, sin texto antes ni después, sin bloques de código, con esta forma exacta:
{"reason": "<1 a 3 frases>", "recommended_action": "<1 frase>", "explanation_points": ["<punto 1>", "<punto 2>", "... entre 2 y 6 puntos"]}
7. Cada punto de "explanation_points" cita al menos un número de la evidencia con su unidad.
8. Formato numérico: coma decimal y punto de miles (ej. 1.048,8 kWh; 110,5 %).`

type payloadEvidence struct {
	Metric   string   `json:"metric"`
	Observed *float64 `json:"observed"`
	Baseline *float64 `json:"baseline"`
	DeltaPct *float64 `json:"delta_pct"`
	Unit     string   `json:"unit"`
	Window   string   `json:"window"`
	Detail   string   `json:"detail"`
}

type payloadEvent struct {
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

type payload struct {
	MeterID        string `json:"meter_id"`
	Classification struct {
		Type       domain.AnomalyType `json:"type"`
		Severity   domain.Severity    `json:"severity"`
		Confidence float64            `json:"confidence"`
	} `json:"classification"`
	Window struct {
		Start          string `json:"start"`
		End            string `json:"end"`
		BaselineWindow string `json:"baseline_window"`
	} `json:"window"`
	Comparison    domain.Comparison `json:"comparison"`
	Evidence      []payloadEvidence `json:"evidence"`
	RelatedEvent  *payloadEvent     `json:"related_event"`
	ActionDefault string            `json:"action_default"`
}

// BuildPayload arma el mensaje de usuario (spec del motor §8.1): nunca incluye lecturas crudas.
func BuildPayload(in engine.ExplainInput) ([]byte, error) {
	var p payload
	p.MeterID = in.MeterID
	p.Classification.Type, p.Classification.Severity, p.Classification.Confidence = in.Type, in.Severity, in.Confidence
	p.Window.Start, p.Window.End, p.Window.BaselineWindow = in.WindowStart, in.WindowEnd, in.BaselineWindow
	p.Comparison = in.Comparison
	for _, e := range in.Evidence {
		p.Evidence = append(p.Evidence, payloadEvidence{Metric: e.Metric, Observed: e.Observed, Baseline: e.Baseline, DeltaPct: e.DeltaPct, Unit: e.Unit, Window: e.Window, Detail: e.Detail})
	}
	if in.RelatedEvent != nil {
		p.RelatedEvent = &payloadEvent{Type: in.RelatedEvent.Type, Timestamp: in.RelatedEvent.Timestamp.UTC(), Description: in.RelatedEvent.Description}
	}
	p.ActionDefault = in.ActionDefault
	return json.Marshal(p)
}
