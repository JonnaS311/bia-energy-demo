// Package llm implementa el Explainer del motor: DeepSeek (API compatible con OpenAI vía go-openai)
// con fallback determinístico por plantillas (spec del motor RF-E-15..RF-E-17, §8.1..§8.5).
package llm

import (
	"context"
	"strings"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
)

// Plantillas de fallback (texto exacto, spec del motor §8.4).
var reasonTemplates = map[domain.AnomalyType]string{
	domain.RealAnomaly:        "El consumo diario de {meter_id} está {delta_pct} % respecto a su baseline de {baseline} kWh durante {days} día(s) consecutivo(s), sin ningún evento operativo que lo explique. La corriente media varió {current_delta_pct} % y el factor de potencia pasó de {pf_baseline} a {pf_observed}.",
	domain.ExplainableAnomaly: "El consumo diario de {meter_id} está {delta_pct} % respecto a su baseline de {baseline} kWh durante {days} día(s), y el cambio coincide con el evento operativo del {event_date}: {event_description}.",
	domain.FalsePositive:      "El consumo diario de {meter_id} bajó {delta_pct} % respecto a su baseline de {baseline} kWh el {window_start}, y la caída está explicada por la parada programada del {event_date}: {event_description}. No se requiere escalamiento.",
	domain.DataQuality:        "Las lecturas eléctricas de {meter_id} entre {window_start} y {window_end} son inconsistentes entre sí: {v_out_count} lecturas con voltaje fuera de 209-231 V, ratio kWh/(V·I·PF) de hasta {ratio_max} y factor de potencia mínimo de {pf_min}, mientras el consumo diario se mantiene estable ({delta_pct} %). Esto apunta a un problema del sensor o de la transmisión de datos, no del consumo.",
}

var pointTemplates = map[domain.AnomalyType][]string{
	domain.RealAnomaly: {
		"Consumo diario: {observed} kWh el {window_end} frente a {baseline} kWh de baseline ({delta_pct} %).",
		"Duración: {days} día(s) consecutivo(s) con desviación ≥ 20 % ({window_start} a {window_end}).",
		"Corriente media diaria: {current_delta_pct} % respecto al baseline.",
		"Factor de potencia: {pf_observed} frente a {pf_baseline} de baseline.",
		"Sin evento operativo registrado que explique el cambio.",
	},
	domain.ExplainableAnomaly: {
		"Consumo diario: {observed} kWh el {window_end} frente a {baseline} kWh de baseline ({delta_pct} %).",
		"Duración: {days} día(s) consecutivo(s) desde {window_start}.",
		"Corriente media diaria: {current_delta_pct} %, coherente con el cambio de consumo.",
		"Evento relacionado el {event_date}: {event_description}.",
	},
	domain.FalsePositive: {
		"Consumo diario: {observed} kWh el {window_start} frente a {baseline} kWh de baseline ({delta_pct} %).",
		"Duración: {days} día(s); el consumo vuelve al baseline después.",
		"Corriente media diaria: {current_delta_pct} %, consistente con equipos apagados.",
		"Evento relacionado el {event_date}: {event_description}.",
	},
	domain.DataQuality: {
		"Voltaje: {v_out_count} lecturas fuera del rango 209-231 V.",
		"Relación física: ratio kWh/(V·I·PF) máximo de {ratio_max} (rango coherente 0,5-2,0).",
		"Factor de potencia mínimo: {pf_min} frente a {pf_baseline} de baseline.",
		"Consumo diario estable: {observed} kWh frente a {baseline} kWh ({delta_pct} %).",
		"Evento relacionado el {event_date}: {event_description}.",
	},
}

func fill(tpl string, in engine.ExplainInput) string {
	f := in.Facts
	delta := domain.FormatSigned(f.DeltaPct, 1)
	if in.Type == domain.FalsePositive {
		// "bajó {delta_pct} %" se lee sin signo.
		delta = domain.FormatNumber(-domain.Round(f.DeltaPct, 1), 1)
	}
	r := strings.NewReplacer(
		"{meter_id}", in.MeterID,
		"{delta_pct}", delta,
		"{baseline}", domain.FormatNumber(f.Baseline, 1),
		"{observed}", domain.FormatNumber(f.Observed, 1),
		"{days}", domain.FormatNumber(float64(f.Days), 0),
		"{event_description}", f.EventDescription,
		"{event_date}", f.EventDate,
		"{current_delta_pct}", domain.FormatSigned(f.CurrentDeltaPct, 1),
		"{pf_observed}", domain.FormatNumber(f.PFObserved, 3),
		"{pf_baseline}", domain.FormatNumber(f.PFBaseline, 3),
		"{v_out_count}", domain.FormatNumber(float64(f.VOutCount), 0),
		"{ratio_max}", domain.FormatNumber(f.RatioMax, 2),
		"{pf_min}", domain.FormatNumber(f.PFMin, 2),
		"{window_start}", f.WindowStart,
		"{window_end}", f.WindowEnd,
	)
	return r.Replace(tpl)
}

// Template genera la explicación por plantilla.
func Template(in engine.ExplainInput) engine.ExplainOutput {
	points := make([]string, 0, 5)
	for i, p := range pointTemplates[in.Type] {
		last := i == len(pointTemplates[in.Type])-1
		if last && in.Type == domain.DataQuality && in.RelatedEvent == nil {
			points = append(points, "Sin evento operativo registrado.")
			continue
		}
		points = append(points, fill(p, in))
	}
	return engine.ExplainOutput{
		Reason:            fill(reasonTemplates[in.Type], in),
		RecommendedAction: in.ActionDefault,
		Points:            points,
		Source:            "template",
	}
}

// TemplateExplainer es un Explainer que siempre usa plantillas (sin red).
type TemplateExplainer struct{}

// Explain implementa engine.Explainer.
func (TemplateExplainer) Explain(_ context.Context, in engine.ExplainInput) engine.ExplainOutput {
	return Template(in)
}
