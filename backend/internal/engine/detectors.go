package engine

import (
	"math"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// dayInfo acumula por día todas las señales de los detectores RF-E-04..RF-E-08.
type dayInfo struct {
	Date       time.Time
	Total      float64
	Complete   bool
	Count      int
	Dev        *float64 // fracción; nil si no hay baseline o el día no es completo
	Spike      bool
	IAvg       float64
	PFAvg      float64
	VAvg       float64
	Electrical bool
	VOut       int
	ROut       int
	PFLow      int
	Outliers   int
	DQ         bool
	Pattern    bool
	VMin, VMax float64
	RatioMax   float64
	PFMin      float64
}

// meterAnalysis es el estado del motor para un medidor.
type meterAnalysis struct {
	stats      *domain.MeterStats
	days       []dayInfo
	candidate  bool
	dq         bool
	pattern    bool
	spikeIdx   []int
	dqIdx      []int
	persistent bool
	longestRun int
	maxDev     float64 // |dev| máximo entre días spike (o del día de patrón)
	signedDev  float64
	maxDevIdx  int
	start, end time.Time
	electrical bool
	related    *domain.Event
	// unknownNearby: hay un evento UNKNOWN a ≤ 24 h del inicio (se menciona en la evidencia, no clasifica).
	unknownNearby bool
	anomaly       *Anomaly
}

// ratio físico kWh/(V·I·PF/1000); +Inf si el denominador es 0 (CB-E-04).
func physicalRatio(r domain.Reading) float64 {
	den := r.VoltageV * r.CurrentA * r.PowerFactor / 1000
	if den == 0 {
		return math.Inf(1)
	}
	return r.ConsumptionKWh / den
}

// analyzeMeter ejecuta los cálculos por día (RF-E-02) y los detectores (RF-E-04..RF-E-08).
func analyzeMeter(st *domain.MeterStats, th Thresholds, dups map[string]int) *meterAnalysis {
	ma := &meterAnalysis{stats: st, maxDevIdx: -1}
	if st.Insufficient || len(st.Readings) == 0 {
		return ma
	}
	byDay := map[time.Time]int{}
	for _, d := range st.Daily {
		di := dayInfo{Date: d.Date, Total: d.ConsumptionKWh, Complete: d.IsComplete, Count: d.ReadingsCount,
			IAvg: d.CurrentAvg, PFAvg: d.PowerFactorAvg, VAvg: d.VoltageAvg, VMin: math.Inf(1), VMax: math.Inf(-1), PFMin: math.Inf(1)}
		if st.HasBaseline && st.Baseline != 0 {
			di.Dev = st.Deviation(d)
		}
		byDay[d.Date] = len(ma.days)
		ma.days = append(ma.days, di)
	}
	for _, r := range st.Readings {
		di := &ma.days[byDay[domain.DateOf(r.Timestamp)]]
		if r.VoltageV < th.VoltageMin || r.VoltageV > th.VoltageMax {
			di.VOut++
		}
		ratio := physicalRatio(r)
		if ratio < th.RatioMin || ratio > th.RatioMax {
			di.ROut++
		}
		if !math.IsInf(ratio, 0) && ratio > di.RatioMax {
			di.RatioMax = ratio
		}
		if r.PowerFactor < th.PFLow {
			di.PFLow++
		}
		if st.ZScore(r) >= th.ZOutlier {
			di.Outliers++
		}
		di.VMin = math.Min(di.VMin, r.VoltageV)
		di.VMax = math.Max(di.VMax, r.VoltageV)
		di.PFMin = math.Min(di.PFMin, r.PowerFactor)
	}

	for i := range ma.days {
		di := &ma.days[i]
		// RF-E-04 spike.
		if di.Dev != nil && math.Abs(*di.Dev) >= th.DevSpike {
			di.Spike = true
			ma.spikeIdx = append(ma.spikeIdx, i)
		}
		// RF-E-08 calidad de datos.
		stable := di.Dev != nil && math.Abs(*di.Dev) < th.DQStableDev
		if di.VOut >= th.MinInconsistent || di.ROut >= th.MinInconsistent || (di.PFLow >= th.MinInconsistent && stable) ||
			di.Count < 24 || dups[domain.ISODate(di.Date)] > 0 {
			di.DQ = true
			ma.dqIdx = append(ma.dqIdx, i)
		}
		// RF-E-07 corroboración eléctrica (solo días spike).
		if di.Spike && st.BaseCurrent > 0 {
			dI := (di.IAvg - st.BaseCurrent) / st.BaseCurrent
			sameSign := (dI > 0) == (*di.Dev > 0)
			if (math.Abs(dI) >= th.CurrentDelta && sameSign) || st.BasePF-di.PFAvg >= th.PFDrop {
				di.Electrical = true
				ma.electrical = true
			}
		}
	}

	// RF-E-05 persistencia: racha más larga de días spike con fechas adyacentes.
	run := 0
	for i := range ma.days {
		if ma.days[i].Spike && (run == 0 || ma.days[i].Date.Sub(ma.days[i-1].Date) == 24*time.Hour && ma.days[i-1].Spike) {
			run++
		} else if ma.days[i].Spike {
			run = 1
		} else {
			run = 0
		}
		if run > ma.longestRun {
			ma.longestRun = run
		}
	}
	ma.persistent = ma.longestRun >= 2

	// max_dev / signed_dev entre días spike.
	for _, i := range ma.spikeIdx {
		d := math.Abs(*ma.days[i].Dev)
		if ma.maxDevIdx < 0 || d > ma.maxDev {
			ma.maxDev, ma.signedDev, ma.maxDevIdx = d, *ma.days[i].Dev, i
		}
	}

	// RF-E-06 patrón horario: día no spike con ≥ OutliersPerDay outliers (solo si no hay spikes).
	if len(ma.spikeIdx) == 0 {
		for i := range ma.days {
			di := &ma.days[i]
			if !di.Spike && di.Outliers >= th.OutliersPerDay {
				di.Pattern = true
				if !ma.pattern {
					ma.pattern = true
					ma.maxDevIdx = i
					if di.Dev != nil {
						ma.maxDev, ma.signedDev = math.Abs(*di.Dev), *di.Dev
					}
				}
			}
		}
	}

	ma.dq = len(ma.dqIdx) > 0
	ma.candidate = ma.dq || len(ma.spikeIdx) > 0 || ma.pattern
	switch {
	case ma.dq:
		ma.start, ma.end = ma.days[ma.dqIdx[0]].Date, ma.days[ma.dqIdx[len(ma.dqIdx)-1]].Date
	case len(ma.spikeIdx) > 0:
		ma.start, ma.end = ma.days[ma.spikeIdx[0]].Date, ma.days[ma.spikeIdx[len(ma.spikeIdx)-1]].Date
	case ma.pattern:
		ma.start, ma.end = ma.days[ma.maxDevIdx].Date, ma.days[ma.maxDevIdx].Date
	}
	return ma
}

// eventPrecedence para desempate (RF-E-09).
func eventPrecedence(t string) int {
	switch t {
	case "SCHEDULED_OUTAGE":
		return 3
	case "OPERATIONAL_CHANGE":
		return 2
	case "DATA_QUALITY":
		return 1
	default:
		return 0
	}
}

// relatedEvent busca el evento del medidor a ≤ 24 h del inicio de la anomalía (RF-E-09).
func relatedEvent(events []domain.Event, meterID string, start time.Time, windowHours float64) *domain.Event {
	var best *domain.Event
	bestDist := math.Inf(1)
	for i := range events {
		e := events[i]
		if e.MeterID != meterID || e.Type == "UNKNOWN" {
			continue
		}
		dist := math.Abs(e.Timestamp.Sub(start).Hours())
		if dist > windowHours {
			continue
		}
		if dist < bestDist || (dist == bestDist && eventPrecedence(e.Type) > eventPrecedence(best.Type)) {
			ev := e
			best, bestDist = &ev, dist
		}
	}
	return best
}
