package domain

import (
	"math"
	"sort"
	"time"
)

// Cálculos derivados puros (spec de data §8.4). Única implementación: la usan la API (vía store)
// y el motor de anomalías, de modo que baseline, variación y perfil horario nunca divergen.

// DefaultBaselineDays es BASELINE_DAYS por defecto.
const DefaultBaselineDays = 7

// DailyTotal es el agregado diario de un medidor (8.4.3).
type DailyTotal struct {
	Date           time.Time
	ConsumptionKWh float64
	VoltageAvg     float64
	CurrentAvg     float64
	PowerFactorAvg float64
	ReadingsCount  int
	IsComplete     bool
}

// HourStat es una hora del perfil baseline (8.4.6).
type HourStat struct {
	Hour int
	Mean float64
	Std  float64
	N    int
}

// MeterStats reúne todos los derivados de un medidor.
type MeterStats struct {
	MeterID       string
	Readings      []Reading // ordenadas por timestamp
	Daily         []DailyTotal
	CompleteDays  int
	Baseline      float64
	BaselineDays  int
	Insufficient  bool // < 2 días completos en todo el periodo
	HasBaseline   bool // !Insufficient y al menos 1 día completo en la ventana
	WindowStart   time.Time
	WindowEnd     time.Time
	Profile       [24]HourStat
	BaseVoltage   float64
	BaseCurrent   float64
	BasePF        float64
	CurrentIdx    int // índice en Daily del último día completo; -1 si no hay
	PeriodTotal   float64
}

// DateOf trunca un instante a su fecha UTC.
func DateOf(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// BaselineWindow devuelve la ventana de baseline global (8.4.4).
func BaselineWindow(periodStart time.Time, days int) (time.Time, time.Time) {
	if days < 1 {
		days = DefaultBaselineDays
	}
	start := DateOf(periodStart)
	return start, start.AddDate(0, 0, days-1)
}

func inWindow(d, start, end time.Time) bool {
	return !d.Before(start) && !d.After(end)
}

func sampleStd(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	s := 0.0
	for _, v := range values {
		s += (v - mean) * (v - mean)
	}
	return math.Sqrt(s / float64(len(values)-1))
}

// ComputeMeterStats calcula todos los derivados de un medidor a partir de sus lecturas.
func ComputeMeterStats(meterID string, readings []Reading, windowStart, windowEnd time.Time) *MeterStats {
	rs := make([]Reading, len(readings))
	copy(rs, readings)
	sort.Slice(rs, func(i, j int) bool { return rs[i].Timestamp.Before(rs[j].Timestamp) })

	m := &MeterStats{MeterID: meterID, Readings: rs, WindowStart: windowStart, WindowEnd: windowEnd, CurrentIdx: -1}

	// 8.4.3 totales diarios + 8.4.2 días completos (24 lecturas con 24 horas distintas).
	type acc struct {
		sum, v, i, pf float64
		n            int
		hours        map[int]bool
	}
	byDay := map[time.Time]*acc{}
	var order []time.Time
	for _, r := range rs {
		d := DateOf(r.Timestamp)
		a, ok := byDay[d]
		if !ok {
			a = &acc{hours: map[int]bool{}}
			byDay[d] = a
			order = append(order, d)
		}
		a.sum += r.ConsumptionKWh
		a.v += r.VoltageV
		a.i += r.CurrentA
		a.pf += r.PowerFactor
		a.n++
		a.hours[r.Timestamp.UTC().Hour()] = true
		m.PeriodTotal += r.ConsumptionKWh
	}
	sort.Slice(order, func(i, j int) bool { return order[i].Before(order[j]) })
	for _, d := range order {
		a := byDay[d]
		n := float64(a.n)
		complete := a.n == 24 && len(a.hours) == 24
		m.Daily = append(m.Daily, DailyTotal{Date: d, ConsumptionKWh: a.sum, VoltageAvg: a.v / n, CurrentAvg: a.i / n, PowerFactorAvg: a.pf / n, ReadingsCount: a.n, IsComplete: complete})
		if complete {
			m.CompleteDays++
			m.CurrentIdx = len(m.Daily) - 1
		}
	}

	// 8.4.5 baseline diario.
	sum, days := 0.0, 0
	for _, d := range m.Daily {
		if d.IsComplete && inWindow(d.Date, windowStart, windowEnd) {
			sum += d.ConsumptionKWh
			days++
		}
	}
	m.Insufficient = m.CompleteDays < 2
	if days > 0 {
		m.Baseline = sum / float64(days)
		m.BaselineDays = days
	}
	m.HasBaseline = !m.Insufficient && days > 0

	// 8.4.6 perfil horario y 8.4.10 promedios eléctricos de baseline.
	perHour := make([][]float64, 24)
	var vs, is, pfs float64
	nb := 0
	for _, r := range rs {
		if !inWindow(DateOf(r.Timestamp), windowStart, windowEnd) {
			continue
		}
		h := r.Timestamp.UTC().Hour()
		perHour[h] = append(perHour[h], r.ConsumptionKWh)
		vs += r.VoltageV
		is += r.CurrentA
		pfs += r.PowerFactor
		nb++
	}
	for h := 0; h < 24; h++ {
		vals := perHour[h]
		st := HourStat{Hour: h, N: len(vals)}
		if len(vals) > 0 {
			t := 0.0
			for _, v := range vals {
				t += v
			}
			st.Mean = t / float64(len(vals))
			st.Std = sampleStd(vals, st.Mean)
		}
		m.Profile[h] = st
	}
	if nb > 0 {
		m.BaseVoltage = vs / float64(nb)
		m.BaseCurrent = is / float64(nb)
		m.BasePF = pfs / float64(nb)
	}
	return m
}

// Current devuelve el último día completo (8.4.7) o nil.
func (m *MeterStats) Current() *DailyTotal {
	if m.CurrentIdx < 0 {
		return nil
	}
	return &m.Daily[m.CurrentIdx]
}

// Deviation devuelve la desviación diaria en fracción (8.4.9) o nil.
func (m *MeterStats) Deviation(d DailyTotal) *float64 {
	if !d.IsComplete || !m.HasBaseline || m.Baseline == 0 {
		return nil
	}
	v := (d.ConsumptionKWh - m.Baseline) / m.Baseline
	return &v
}

// VariationPct devuelve la variación en porcentaje (8.4.8) o nil.
func (m *MeterStats) VariationPct() *float64 {
	c := m.Current()
	if c == nil {
		return nil
	}
	dev := m.Deviation(*c)
	if dev == nil {
		return nil
	}
	v := *dev * 100
	return &v
}

// ZScore devuelve el z-score absoluto de una lectura respecto al perfil horario; 0 si σ = 0.
func (m *MeterStats) ZScore(r Reading) float64 {
	p := m.Profile[r.Timestamp.UTC().Hour()]
	if p.Std <= 0 {
		return 0
	}
	return math.Abs(r.ConsumptionKWh-p.Mean) / p.Std
}

// IsOutlier aplica 8.4.11 con umbral z ≥ 3.
func (m *MeterStats) IsOutlier(r Reading) bool {
	return m.ZScore(r) >= 3
}

// DailyAt devuelve el agregado de una fecha o nil.
func (m *MeterStats) DailyAt(date time.Time) *DailyTotal {
	for i := range m.Daily {
		if m.Daily[i].Date.Equal(date) {
			return &m.Daily[i]
		}
	}
	return nil
}

// Dataset es la foto en memoria del dataset tras la ingesta.
type Dataset struct {
	PeriodStart  time.Time
	PeriodEnd    time.Time
	WindowStart  time.Time
	WindowEnd    time.Time
	BaselineDays int
	MeterIDs     []string
	Meters       map[string]*MeterStats
	Events       []Event
	Total        float64
	GlobalDaily  []DailyTotal // solo Date y ConsumptionKWh
	ReadingCount int
}

// BuildDataset agrupa lecturas por medidor y calcula todos los derivados.
func BuildDataset(readings []Reading, events []Event, meterIDs []string, baselineDays int) *Dataset {
	ds := &Dataset{Meters: map[string]*MeterStats{}, Events: events, BaselineDays: baselineDays, ReadingCount: len(readings)}
	if baselineDays < 1 {
		ds.BaselineDays = DefaultBaselineDays
	}
	byMeter := map[string][]Reading{}
	global := map[time.Time]float64{}
	for i, r := range readings {
		byMeter[r.MeterID] = append(byMeter[r.MeterID], r)
		ds.Total += r.ConsumptionKWh
		global[DateOf(r.Timestamp)] += r.ConsumptionKWh
		if i == 0 || r.Timestamp.Before(ds.PeriodStart) {
			ds.PeriodStart = r.Timestamp
		}
		if i == 0 || r.Timestamp.After(ds.PeriodEnd) {
			ds.PeriodEnd = r.Timestamp
		}
	}
	ds.PeriodStart, ds.PeriodEnd = DateOf(ds.PeriodStart), DateOf(ds.PeriodEnd)
	ds.WindowStart, ds.WindowEnd = BaselineWindow(ds.PeriodStart, ds.BaselineDays)

	seen := map[string]bool{}
	for _, id := range meterIDs {
		seen[id] = true
	}
	for id := range byMeter {
		if !seen[id] {
			meterIDs = append(meterIDs, id)
			seen[id] = true
		}
	}
	sort.Strings(meterIDs)
	ds.MeterIDs = meterIDs
	for _, id := range meterIDs {
		ds.Meters[id] = ComputeMeterStats(id, byMeter[id], ds.WindowStart, ds.WindowEnd)
	}
	dates := make([]time.Time, 0, len(global))
	for d := range global {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	for _, d := range dates {
		ds.GlobalDaily = append(ds.GlobalDaily, DailyTotal{Date: d, ConsumptionKWh: global[d]})
	}
	return ds
}

// EventsFor devuelve los eventos de un medidor ordenados por timestamp.
func (ds *Dataset) EventsFor(meterID string) []Event {
	var out []Event
	for _, e := range ds.Events {
		if e.MeterID == meterID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}

// ComparisonEntry es observado vs baseline de una variable (maestro 8.2.6).
type ComparisonEntry struct {
	Observed float64  `json:"observed"`
	Baseline float64  `json:"baseline"`
	DeltaPct *float64 `json:"delta_pct"`
}

// Comparison agrupa las 4 variables comparadas.
type Comparison struct {
	ConsumptionKWh ComparisonEntry `json:"consumption_kwh"`
	VoltageV       ComparisonEntry `json:"voltage_v"`
	CurrentA       ComparisonEntry `json:"current_a"`
	PowerFactor    ComparisonEntry `json:"power_factor"`
}

// DeltaPct devuelve (obs − base)/base × 100 redondeado a 1 decimal, o nil si base = 0.
func DeltaPct(obs, base float64) *float64 {
	if base == 0 {
		return nil
	}
	v := Round((obs-base)/base*100, 1)
	return &v
}

// CompareAt construye la comparación del día `date` contra el baseline (maestro 8.2.6).
func (m *MeterStats) CompareAt(date time.Time) Comparison {
	d := m.DailyAt(date)
	if d == nil {
		d = m.Current()
	}
	var c Comparison
	if d == nil {
		return c
	}
	mk := func(obs, base float64, dec int) ComparisonEntry {
		return ComparisonEntry{Observed: Round(obs, dec), Baseline: Round(base, dec), DeltaPct: DeltaPct(obs, base)}
	}
	c.ConsumptionKWh = mk(d.ConsumptionKWh, m.Baseline, 1)
	c.VoltageV = mk(d.VoltageAvg, m.BaseVoltage, 1)
	c.CurrentA = mk(d.CurrentAvg, m.BaseCurrent, 1)
	c.PowerFactor = mk(d.PowerFactorAvg, m.BasePF, 3)
	return c
}

// Round redondea half-away-from-zero a `dec` decimales.
func Round(v float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Round(v*p) / p
}

// RoundPtr redondea un puntero o devuelve nil.
func RoundPtr(v *float64, dec int) *float64 {
	if v == nil {
		return nil
	}
	r := Round(*v, dec)
	return &r
}
