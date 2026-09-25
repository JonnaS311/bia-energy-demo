package engine

import (
	"fmt"
	"math"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

func fp(v float64) *float64 { return &v }

func r1(v float64) *float64 { return fp(domain.Round(v, 1)) }
func r3(v float64) *float64 { return fp(domain.Round(v, 3)) }

func window(ma *meterAnalysis) string {
	s, e := domain.ISODate(ma.start), domain.ISODate(ma.end)
	if s == e {
		return s
	}
	return s + ".." + e
}

func pctText(v float64) string { return domain.FormatNumber(math.Abs(domain.Round(v, 1)), 1) }

// buildEvidence genera la evidencia por tipo (RF-E-14) y los hechos para las plantillas.
func buildEvidence(t domain.AnomalyType, ma *meterAnalysis, th Thresholds) ([]EvidenceItem, Facts) {
	st := ma.stats
	endDay := ma.days[indexOf(ma, ma.end)]
	var ev []EvidenceItem
	facts := Facts{
		Baseline: st.Baseline, WindowStart: domain.ISODate(ma.start), WindowEnd: domain.ISODate(ma.end),
		PFBaseline: st.BasePF, PFObserved: endDay.PFAvg,
	}
	if st.BaseCurrent > 0 {
		facts.CurrentDeltaPct = (endDay.IAvg - st.BaseCurrent) / st.BaseCurrent * 100
	}
	if ma.related != nil {
		facts.EventDescription = ma.related.Description
		facts.EventDate = domain.ISODateTimeShort(ma.related.Timestamp)
	}

	currentItem := func() EvidenceItem {
		d := domain.DeltaPct(endDay.IAvg, st.BaseCurrent)
		detail := "Corriente media diaria sin cambio relevante"
		if d != nil {
			switch {
			case *d >= 100:
				detail = fmt.Sprintf("Corriente media diaria %s veces el baseline", domain.FormatNumber(endDay.IAvg/st.BaseCurrent, 1))
			case *d >= 0:
				detail = fmt.Sprintf("Corriente media diaria %s %% por encima del baseline", pctText(*d))
			default:
				detail = fmt.Sprintf("Corriente media diaria %s %% por debajo del baseline", pctText(*d))
			}
		}
		return EvidenceItem{Metric: "CURRENT_A", Observed: r1(endDay.IAvg), Baseline: r1(st.BaseCurrent), DeltaPct: d, Unit: "A", Window: domain.ISODate(ma.end), Detail: detail}
	}
	relatedItem := func() EvidenceItem {
		if ma.related != nil {
			return EvidenceItem{Metric: "RELATED_EVENT", Window: domain.ISODate(ma.related.Timestamp),
				Detail: fmt.Sprintf("Evento %s %s: %s", ma.related.Type, domain.ISODateTimeShort(ma.related.Timestamp), ma.related.Description)}
		}
		from, to := ma.start.AddDate(0, 0, -1), ma.start.AddDate(0, 0, 1)
		detail := "Sin evento operativo que explique el cambio"
		if ma.unknownNearby {
			detail += " (evento UNKNOWN ignorado)"
		}
		return EvidenceItem{Metric: "RELATED_EVENT", Window: domain.ISODate(from) + ".." + domain.ISODate(to), Detail: detail}
	}

	if t == domain.DataQuality {
		vOut, rOut, pfLow := 0, 0, 0
		vMin, vMax, ratioMax, pfMin := math.Inf(1), math.Inf(-1), 0.0, math.Inf(1)
		for _, i := range ma.dqIdx {
			d := ma.days[i]
			vOut += d.VOut
			rOut += d.ROut
			pfLow += d.PFLow
			vMin, vMax = math.Min(vMin, d.VMin), math.Max(vMax, d.VMax)
			ratioMax = math.Max(ratioMax, d.RatioMax)
			pfMin = math.Min(pfMin, d.PFMin)
		}
		facts.VOutCount, facts.RatioMax, facts.PFMin = vOut, ratioMax, pfMin
		facts.Observed = endDay.Total
		if endDay.Dev != nil {
			facts.DeltaPct = *endDay.Dev * 100
		}
		w := window(ma)
		ev = append(ev,
			EvidenceItem{Metric: "VOLTAGE_OUT_OF_RANGE", Observed: fp(float64(vOut)), Unit: "readings", Window: w,
				Detail: fmt.Sprintf("%d lecturas con voltaje fuera de %s-%s V (mín. %s V, máx. %s V)", vOut,
					domain.FormatNumber(th.VoltageMin, 0), domain.FormatNumber(th.VoltageMax, 0), domain.FormatNumber(vMin, 1), domain.FormatNumber(vMax, 1))},
			EvidenceItem{Metric: "PHYSICAL_RATIO", Observed: fp(domain.Round(ratioMax, 2)), Baseline: fp(1.0), DeltaPct: domain.DeltaPct(domain.Round(ratioMax, 2), 1.0), Unit: "ratio", Window: w,
				Detail: fmt.Sprintf("%d lecturas con ratio kWh/(V·I·PF) fuera de %s-%s; máximo %s", rOut,
					domain.FormatNumber(th.RatioMin, 1), domain.FormatNumber(th.RatioMax, 1), domain.FormatNumber(ratioMax, 2))},
			EvidenceItem{Metric: "POWER_FACTOR", Observed: r3(pfMin), Baseline: r3(st.BasePF), DeltaPct: domain.DeltaPct(pfMin, st.BasePF), Unit: "", Window: w,
				Detail: fmt.Sprintf("Factor de potencia mínimo %s frente a %s de baseline; %d lecturas por debajo de %s",
					domain.FormatNumber(pfMin, 2), domain.FormatNumber(st.BasePF, 3), pfLow, domain.FormatNumber(th.PFLow, 2))},
		)
		dcDetail := "Consumo diario del último día de la ventana"
		if endDay.Dev != nil {
			dcDetail = fmt.Sprintf("Consumo diario estable (%s %%) pese a las lecturas eléctricas inconsistentes", domain.FormatSigned(*endDay.Dev*100, 1))
		}
		ev = append(ev, EvidenceItem{Metric: "DAILY_CONSUMPTION", Observed: r1(endDay.Total), Baseline: r1(st.Baseline), DeltaPct: domain.DeltaPct(endDay.Total, st.Baseline), Unit: "kWh", Window: domain.ISODate(ma.end), Detail: dcDetail})
		if st.BaseCurrent > 0 && math.Abs(endDay.IAvg-st.BaseCurrent)/st.BaseCurrent >= th.CurrentDelta {
			ev = append(ev, currentItem())
		}
		ev = append(ev, relatedItem())
	} else {
		peak := ma.days[ma.maxDevIdx]
		facts.Observed = peak.Total
		facts.DeltaPct = ma.signedDev * 100
		facts.Days = ma.longestRun
		if facts.Days == 0 {
			facts.Days = 1
		}
		dir := "por encima del"
		if ma.signedDev < 0 {
			dir = "por debajo del"
		}
		ev = append(ev, EvidenceItem{Metric: "DAILY_CONSUMPTION", Observed: r1(peak.Total), Baseline: r1(st.Baseline), DeltaPct: domain.DeltaPct(peak.Total, st.Baseline),
			Unit: "kWh", Window: domain.ISODate(peak.Date), Detail: fmt.Sprintf("Total diario %s %% %s baseline", pctText(ma.signedDev*100), dir)})
		pDetail := fmt.Sprintf("%d días consecutivos con desviación ≥ %s %%", facts.Days, domain.FormatNumber(th.DevSpike*100, 0))
		if facts.Days == 1 {
			pDetail = fmt.Sprintf("Un solo día con desviación ≥ %s %%", domain.FormatNumber(th.DevSpike*100, 0))
			if len(ma.spikeIdx) == 0 {
				pDetail = "Ningún día supera el umbral de desviación; el cambio es de patrón horario"
			}
		}
		ev = append(ev, EvidenceItem{Metric: "PERSISTENCE_DAYS", Observed: fp(float64(facts.Days)), Unit: "days", Window: window(ma), Detail: pDetail})
		ev = append(ev, currentItem())
		pfDrop := st.BasePF - endDay.PFAvg
		pfDetail := fmt.Sprintf("Factor de potencia sin cambio relevante (%s)", domain.FormatSigned(-pfDrop, 3))
		if pfDrop >= th.PFDrop {
			pfDetail = fmt.Sprintf("Factor de potencia cae %s respecto al baseline", domain.FormatNumber(pfDrop, 2))
		}
		ev = append(ev, EvidenceItem{Metric: "POWER_FACTOR", Observed: r3(endDay.PFAvg), Baseline: r3(st.BasePF), DeltaPct: domain.DeltaPct(endDay.PFAvg, st.BasePF), Unit: "", Window: domain.ISODate(ma.end), Detail: pfDetail})
		outliers := 0
		for i := range ma.days {
			d := ma.days[i]
			if !d.Date.Before(ma.start) && !d.Date.After(ma.end) {
				outliers += d.Outliers
			}
		}
		if outliers > 0 {
			ev = append(ev, EvidenceItem{Metric: "HOURLY_OUTLIERS", Observed: fp(float64(outliers)), Unit: "readings", Window: window(ma),
				Detail: fmt.Sprintf("%d lecturas fuera del perfil horario (z ≥ %s)", outliers, domain.FormatNumber(th.ZOutlier, 0))})
		}
		ev = append(ev, relatedItem())
	}
	for i := range ev {
		ev[i].Position = i + 1
	}
	return ev, facts
}

func indexOf(ma *meterAnalysis, date interface{ Unix() int64 }) int {
	for i := range ma.days {
		if ma.days[i].Date.Unix() == date.Unix() {
			return i
		}
	}
	return len(ma.days) - 1
}
