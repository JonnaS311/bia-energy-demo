package engine

import (
	"math"
	"sort"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// classify aplica la precedencia fija de RF-E-10.
func classify(ma *meterAnalysis) domain.AnomalyType {
	if ma.dq {
		return domain.DataQuality
	}
	if ma.related != nil {
		switch ma.related.Type {
		case "SCHEDULED_OUTAGE":
			if ma.signedDev < 0 && len(ma.spikeIdx) <= 2 {
				return domain.FalsePositive
			}
		case "OPERATIONAL_CHANGE":
			return domain.ExplainableAnomaly
		}
	}
	return domain.RealAnomaly
}

// severity aplica la tabla de RF-E-11.
func severity(t domain.AnomalyType, ma *meterAnalysis, th Thresholds) domain.Severity {
	switch t {
	case domain.DataQuality:
		return domain.SeverityHigh
	case domain.FalsePositive:
		return domain.SeverityLow
	case domain.ExplainableAnomaly:
		return domain.SeverityMedium
	default:
		if len(ma.spikeIdx) == 0 { // solo patrón horario
			return domain.SeverityMedium
		}
		if ma.maxDev >= th.DevHigh {
			return domain.SeverityHigh
		}
		return domain.SeverityMedium
	}
}

// confidence aplica la fórmula aditiva de RF-E-12 en centésimas enteras (sin error de coma flotante),
// acotada a [0,50; 0,98].
func confidence(t domain.AnomalyType, ma *meterAnalysis, th Thresholds) float64 {
	c := 50
	if t == domain.DataQuality {
		// Volumen: lecturas inconsistentes (vOut + rOut) en todos los días; violación física: días DQ con rOut ≥ 3.
		inconsistent, rDays := 0, 0
		for _, d := range ma.days {
			inconsistent += d.VOut + d.ROut
		}
		for _, i := range ma.dqIdx {
			if ma.days[i].ROut >= th.MinInconsistent {
				rDays++
			}
		}
		if inconsistent >= 2*th.MinInconsistent {
			c += 20
		}
		if len(ma.dqIdx) >= 2 {
			c += 15
		}
		if rDays >= 1 {
			c += 15
		}
		if ma.related != nil && ma.related.Type == "DATA_QUALITY" {
			c += 10
		}
	} else {
		switch {
		case ma.maxDev >= th.DevHigh:
			c += 20
		case ma.maxDev >= th.DevSpike:
			c += 5
		}
		if ma.persistent {
			c += 15
		}
		if ma.electrical {
			c += 15
		}
		coherent := (t == domain.ExplainableAnomaly && ma.signedDev > 0) ||
			(t == domain.FalsePositive && ma.signedDev < 0) ||
			(t == domain.RealAnomaly && ma.related == nil && ma.electrical)
		if coherent {
			c += 10
		}
		if !ma.persistent && !ma.electrical {
			c -= 10
		}
	}
	if c < 50 {
		c = 50
	}
	if c > 98 {
		c = 98
	}
	return float64(c) / 100
}

var sevRank = map[domain.Severity]int{domain.SeverityHigh: 3, domain.SeverityMedium: 2, domain.SeverityLow: 1}
var typeRank = map[domain.AnomalyType]int{domain.RealAnomaly: 4, domain.DataQuality: 3, domain.ExplainableAnomaly: 2, domain.FalsePositive: 1}

// prioritize ordena y asigna priority_rank (RF-E-13).
func prioritize(list []Anomaly) {
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if sevRank[a.Severity] != sevRank[b.Severity] {
			return sevRank[a.Severity] > sevRank[b.Severity]
		}
		if typeRank[a.Type] != typeRank[b.Type] {
			return typeRank[a.Type] > typeRank[b.Type]
		}
		if math.Abs(a.Confidence-b.Confidence) > 1e-9 {
			return a.Confidence > b.Confidence
		}
		return a.MeterID < b.MeterID
	})
	for i := range list {
		list[i].PriorityRank = i + 1
	}
}
