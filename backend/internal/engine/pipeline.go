package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// Pipeline ejecuta las 7 etapas del análisis (RF-E-01).
type Pipeline struct {
	Explainer Explainer
}

// New crea un pipeline con el Explainer dado.
func New(explainer Explainer) *Pipeline { return &Pipeline{Explainer: explainer} }

// Run ejecuta el pipeline. onStep puede ser nil.
func (p *Pipeline) Run(ctx context.Context, in Input, onStep StepCallback) (Result, error) {
	if p.Explainer == nil {
		return Result{}, errors.New("engine: explainer is nil")
	}
	th := in.Thresholds.WithDefaults()
	notify := func(key string, status domain.StepStatus, detail string) error {
		if onStep == nil {
			return nil
		}
		return onStep(key, status, detail)
	}
	var (
		ds       *domain.Dataset
		analyses []*meterAnalysis
		result   Result
	)

	stages := []struct {
		key string
		run func() (string, error)
	}{
		{"READINGS", func() (string, error) {
			ids := in.MeterIDs
			ds = domain.BuildDataset(in.Readings, in.Events, ids, th.BaselineDays)
			n := 0
			for _, m := range ds.Meters {
				if len(m.Readings) > 0 {
					n++
				}
			}
			return fmt.Sprintf("%s lecturas de %d medidores", domain.FormatNumber(float64(len(in.Readings)), 0), n), nil
		}},
		{"BASELINE", func() (string, error) {
			insufficient := 0
			for _, id := range ds.MeterIDs {
				if ds.Meters[id].Insufficient {
					insufficient++
				}
			}
			d := fmt.Sprintf("Baseline calculado para %d medidores", len(ds.MeterIDs)-insufficient)
			if insufficient > 0 {
				d += fmt.Sprintf("; %d con datos insuficientes", insufficient)
			}
			return d, nil
		}},
		{"DETECTION", func() (string, error) {
			n := 0
			for _, id := range ds.MeterIDs {
				ma := analyzeMeter(ds.Meters[id], th, in.DuplicatesByMeterDay[id])
				analyses = append(analyses, ma)
				if ma.candidate {
					n++
				}
			}
			return fmt.Sprintf("%d medidores con desviaciones o inconsistencias", n), nil
		}},
		{"CORRELATION", func() (string, error) {
			n := 0
			for _, ma := range analyses {
				if ma.candidate {
					n++
				}
			}
			return fmt.Sprintf("Corroboración eléctrica evaluada en %d medidores", n), nil
		}},
		{"EVENTS", func() (string, error) {
			for _, ma := range analyses {
				if !ma.candidate {
					continue
				}
				id := ma.stats.MeterID
				ma.related = relatedEvent(in.Events, id, ma.start, th.EventWindowHours)
				for _, e := range in.Events {
					if e.MeterID == id && e.Type == "UNKNOWN" && math.Abs(e.Timestamp.Sub(ma.start).Hours()) <= th.EventWindowHours {
						ma.unknownNearby = true
					}
				}
				t := classify(ma)
				a := Anomaly{MeterID: id, Type: t, Severity: severity(t, ma, th), Confidence: confidence(t, ma, th),
					WindowStart: ma.start, WindowEnd: ma.end, RelatedEvent: ma.related, ActionDefault: ActionDefault[t],
					Comparison: ma.stats.CompareAt(ma.end)}
				a.Evidence, _ = buildEvidence(t, ma, th)
				ma.anomaly = &a
				result.Anomalies = append(result.Anomalies, a)
			}
			prioritize(result.Anomalies)
			return fmt.Sprintf("%d anomalías clasificadas", len(result.Anomalies)), nil
		}},
		{"EXPLANATION", func() (string, error) {
			byMeter := map[string]*meterAnalysis{}
			for _, ma := range analyses {
				byMeter[ma.stats.MeterID] = ma
			}
			nLLM, nTpl := 0, 0
			for i := range result.Anomalies {
				if err := ctx.Err(); err != nil {
					return "", err
				}
				a := &result.Anomalies[i]
				ma := byMeter[a.MeterID]
				_, facts := buildEvidence(a.Type, ma, th)
				out := p.Explainer.Explain(ctx, ExplainInput{
					MeterID: a.MeterID, Type: a.Type, Severity: a.Severity, Confidence: a.Confidence,
					WindowStart: domain.ISODate(a.WindowStart), WindowEnd: domain.ISODate(a.WindowEnd),
					BaselineWindow: domain.ISODate(ds.WindowStart) + ".." + domain.ISODate(ds.WindowEnd),
					Comparison: a.Comparison, Evidence: a.Evidence, RelatedEvent: a.RelatedEvent,
					ActionDefault: a.ActionDefault, Facts: facts,
				})
				a.Reason, a.ExplanationPoints, a.ExplanationSource = out.Reason, out.Points, out.Source
				a.RecommendedAction = out.RecommendedAction
				if out.Source == "llm" {
					nLLM++
				} else {
					nTpl++
				}
			}
			return fmt.Sprintf("%d explicaciones por LLM, %d por plantilla", nLLM, nTpl), nil
		}},
		{"RECOMMENDATION", func() (string, error) {
			high := 0
			for i := range result.Anomalies {
				a := &result.Anomalies[i]
				if a.RecommendedAction == "" {
					a.RecommendedAction = a.ActionDefault
				}
				if a.Severity == domain.SeverityHigh {
					high++
				}
			}
			status := map[string]domain.MeterStatus{}
			for _, a := range result.Anomalies {
				status[a.MeterID] = domain.DeriveMeterStatus(a.Type, a.Severity)
			}
			for _, id := range ds.MeterIDs {
				s, ok := status[id]
				if !ok {
					s = domain.MeterOK
				}
				result.Meters = append(result.Meters, MeterSummary{MeterID: id, InsufficientData: ds.Meters[id].Insufficient, Status: s})
			}
			sort.Slice(result.Meters, func(i, j int) bool { return result.Meters[i].MeterID < result.Meters[j].MeterID })
			return fmt.Sprintf("%d anomalías · %d requieren atención prioritaria", len(result.Anomalies), high), nil
		}},
	}

	for _, s := range stages {
		if err := ctx.Err(); err != nil {
			_ = notify(s.key, domain.StepFailed, err.Error())
			return Result{}, err
		}
		if err := notify(s.key, domain.StepRunning, ""); err != nil {
			return Result{}, err
		}
		detail, err := s.run()
		if err != nil {
			_ = notify(s.key, domain.StepFailed, err.Error())
			return Result{}, fmt.Errorf("step %s: %w", s.key, err)
		}
		if err := notify(s.key, domain.StepCompleted, detail); err != nil {
			return Result{}, err
		}
	}
	return result, nil
}
