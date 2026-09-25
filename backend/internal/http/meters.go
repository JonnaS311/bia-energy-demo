package http

import (
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/store"
)

type anomalyRefDTO struct {
	ID         string             `json:"id"`
	Type       domain.AnomalyType `json:"type"`
	Severity   domain.Severity    `json:"severity"`
	Confidence float64            `json:"confidence"`
}

type meterItemDTO struct {
	MeterID               string             `json:"meter_id"`
	Name                  string             `json:"name"`
	Location              string             `json:"location"`
	Status                domain.MeterStatus `json:"status"`
	CurrentConsumptionKWh *float64           `json:"current_consumption_kwh"`
	BaselineDailyKWh      *float64           `json:"baseline_daily_kwh"`
	VariationPct          *float64           `json:"variation_pct"`
	PeriodConsumptionKWh  float64            `json:"period_consumption_kwh"`
	Anomaly               *anomalyRefDTO     `json:"anomaly"`
}

func meterItem(m store.Meter, ms *domain.MeterStats, a *store.Anomaly) meterItemDTO {
	it := meterItemDTO{MeterID: m.MeterID, Name: m.Name, Location: m.Location, Status: m.Status}
	if ms != nil {
		if c := ms.Current(); c != nil {
			it.CurrentConsumptionKWh = domain.RoundPtr(&c.ConsumptionKWh, 1)
		}
		if ms.HasBaseline {
			b := ms.Baseline
			it.BaselineDailyKWh = domain.RoundPtr(&b, 1)
		}
		it.VariationPct = domain.RoundPtr(ms.VariationPct(), 1)
		it.PeriodConsumptionKWh = domain.Round(ms.PeriodTotal, 1)
	}
	if a != nil {
		it.Anomaly = &anomalyRefDTO{ID: a.ID.String(), Type: a.Type, Severity: a.Severity, Confidence: a.Confidence}
	}
	return it
}

var statusRank = map[domain.MeterStatus]int{domain.MeterCritical: 3, domain.MeterAlert: 2, domain.MeterOK: 1}
var severityRank = map[domain.Severity]int{domain.SeverityHigh: 3, domain.SeverityMedium: 2, domain.SeverityLow: 1}

// sortMeters aplica los comparadores de RF-B-06.
func sortMeters(items []meterItemDTO, sortBy, order string) {
	desc := order == "desc"
	num := func(p *float64) float64 {
		if p == nil {
			return 0
		}
		return *p
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		switch sortBy {
		case "consumption":
			x, y := num(a.CurrentConsumptionKWh), num(b.CurrentConsumptionKWh)
			if x != y {
				return (x > y) == desc
			}
		case "variation":
			if (a.VariationPct == nil) != (b.VariationPct == nil) {
				return b.VariationPct == nil // null al final en ambos órdenes
			}
			x, y := num(a.VariationPct), num(b.VariationPct)
			if x != y {
				return (x > y) == desc
			}
		default: // severity
			if statusRank[a.Status] != statusRank[b.Status] {
				return (statusRank[a.Status] > statusRank[b.Status]) == desc
			}
			sa, sb, ca, cb := 0, 0, 0.0, 0.0
			if a.Anomaly != nil {
				sa, ca = severityRank[a.Anomaly.Severity], a.Anomaly.Confidence
			}
			if b.Anomaly != nil {
				sb, cb = severityRank[b.Anomaly.Severity], b.Anomaly.Confidence
			}
			if sa != sb {
				return (sa > sb) == desc
			}
			if ca != cb {
				return (ca > cb) == desc
			}
		}
		return a.MeterID < b.MeterID
	})
}

// listMeters implementa GET /meters (RF-B-06).
func (s *Server) listMeters(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	statuses := q["status"]
	for _, st := range statuses {
		if !domain.Contains(domain.MeterStatuses, st) {
			invalidQuery(w, "status", domain.MeterStatuses, "")
			return
		}
	}
	search := q.Get("q")
	if len([]rune(search)) > 20 {
		invalidQuery(w, "q", nil, "max length 20")
		return
	}
	sortBy := q.Get("sort")
	if sortBy == "" {
		sortBy = "severity"
	}
	if !domain.Contains([]string{"consumption", "variation", "severity"}, sortBy) {
		invalidQuery(w, "sort", []string{"consumption", "variation", "severity"}, "")
		return
	}
	order := q.Get("order")
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		invalidQuery(w, "order", []string{"asc", "desc"}, "")
		return
	}

	ctx := r.Context()
	meters, err := s.store.ListMeters(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	ds, err := s.store.Dataset(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	current, err := s.store.CurrentAnomaliesByMeter(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	items := []meterItemDTO{}
	for _, m := range meters {
		if len(statuses) > 0 && !domain.Contains(statuses, string(m.Status)) {
			continue
		}
		if search != "" && !containsFold(m.MeterID, search) {
			continue
		}
		var ap *store.Anomaly
		if a, ok := current[m.MeterID]; ok {
			ap = &a
		}
		items = append(items, meterItem(m, ds.Meters[m.MeterID], ap))
	}
	sortMeters(items, sortBy, order)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func containsFold(s, sub string) bool {
	return len(sub) == 0 || indexFold(s, sub) >= 0
}

func indexFold(s, sub string) int {
	ls, lsub := []rune(toLower(s)), []rune(toLower(sub))
	for i := 0; i+len(lsub) <= len(ls); i++ {
		if string(ls[i:i+len(lsub)]) == string(lsub) {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := []rune(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

type eventDTO struct {
	ID          int64     `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
}

func eventOut(e domain.Event) eventDTO {
	return eventDTO{ID: e.ID, Timestamp: e.Timestamp.UTC(), Type: e.Type, Description: e.Description}
}

type electricalDTO struct {
	CurrentAvg  *float64 `json:"current_avg"`
	BaselineAvg float64  `json:"baseline_avg"`
}

// getMeter implementa GET /meters/{meterId} (RF-B-07).
func (s *Server) getMeter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "meterId")
	ctx := r.Context()
	m, err := s.store.GetMeter(ctx, id)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	if m == nil {
		writeError(w, http.StatusNotFound, CodeMeterNotFound, "No existe el medidor "+id, nil)
		return
	}
	ds, err := s.store.Dataset(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	current, err := s.store.CurrentAnomaliesByMeter(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	var ap *store.Anomaly
	if a, ok := current[id]; ok {
		ap = &a
	}
	ms := ds.Meters[id]
	item := meterItem(*m, ms, ap)

	events := []eventDTO{}
	for _, e := range ds.EventsFor(id) {
		events = append(events, eventOut(e))
	}
	type dailyDTO struct {
		Date           string   `json:"date"`
		ConsumptionKWh float64  `json:"consumption_kwh"`
		DeviationPct   *float64 `json:"deviation_pct"`
	}
	daily := []dailyDTO{}
	count := 0
	elec := map[string]electricalDTO{}
	if ms != nil {
		count = len(ms.Readings)
		for _, d := range ms.Daily {
			var dev *float64
			if v := ms.Deviation(d); v != nil {
				p := *v * 100
				dev = domain.RoundPtr(&p, 1)
			}
			daily = append(daily, dailyDTO{Date: domain.ISODate(d.Date), ConsumptionKWh: domain.Round(d.ConsumptionKWh, 1), DeviationPct: dev})
		}
		c := ms.Current()
		cur := func(v float64, dec int) *float64 {
			if c == nil {
				return nil
			}
			return domain.RoundPtr(&v, dec)
		}
		var cv, ci, cp float64
		if c != nil {
			cv, ci, cp = c.VoltageAvg, c.CurrentAvg, c.PowerFactorAvg
		}
		elec["voltage_v"] = electricalDTO{CurrentAvg: cur(cv, 1), BaselineAvg: domain.Round(ms.BaseVoltage, 1)}
		elec["current_a"] = electricalDTO{CurrentAvg: cur(ci, 1), BaselineAvg: domain.Round(ms.BaseCurrent, 1)}
		elec["power_factor"] = electricalDTO{CurrentAvg: cur(cp, 3), BaselineAvg: domain.Round(ms.BasePF, 3)}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"meter_id": item.MeterID, "name": item.Name, "location": item.Location, "status": item.Status,
		"current_consumption_kwh": item.CurrentConsumptionKWh, "baseline_daily_kwh": item.BaselineDailyKWh,
		"variation_pct": item.VariationPct, "period_consumption_kwh": item.PeriodConsumptionKWh, "anomaly": item.Anomaly,
		"period":     map[string]any{"start": domain.ISODate(ds.PeriodStart), "end": domain.ISODate(ds.PeriodEnd), "readings_count": count},
		"electrical": elec,
		"events":     events,
		"daily":      daily,
	})
}

func parseBound(v string, endOfDay bool) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC(), true
	}
	if t, err := time.ParseInLocation("2006-01-02", v, time.UTC); err == nil {
		if endOfDay {
			return t.Add(24*time.Hour - time.Second), true
		}
		return t, true
	}
	return time.Time{}, false
}

// getReadings implementa GET /meters/{meterId}/readings (RF-B-08).
func (s *Server) getReadings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "meterId")
	q := r.URL.Query()
	gran := q.Get("granularity")
	if gran == "" {
		gran = "hour"
	}
	if gran != "hour" && gran != "day" {
		invalidQuery(w, "granularity", []string{"hour", "day"}, "")
		return
	}
	ctx := r.Context()
	m, err := s.store.GetMeter(ctx, id)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	if m == nil {
		writeError(w, http.StatusNotFound, CodeMeterNotFound, "No existe el medidor "+id, nil)
		return
	}
	ds, err := s.store.Dataset(ctx)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	from := ds.PeriodStart
	to := ds.PeriodEnd.Add(24*time.Hour - time.Second)
	if v := q.Get("from"); v != "" {
		t, ok := parseBound(v, false)
		if !ok {
			invalidQuery(w, "from", nil, "expected RFC 3339 or YYYY-MM-DD")
			return
		}
		from = t
	}
	if v := q.Get("to"); v != "" {
		t, ok := parseBound(v, true)
		if !ok {
			invalidQuery(w, "to", nil, "expected RFC 3339 or YYYY-MM-DD")
			return
		}
		to = t
	}
	if from.After(to) {
		invalidQuery(w, "from", nil, "from must be <= to")
		return
	}
	ms := ds.Meters[id]

	if gran == "day" {
		type dayDTO struct {
			Date           string   `json:"date"`
			ConsumptionKWh float64  `json:"consumption_kwh"`
			DeviationPct   *float64 `json:"deviation_pct"`
			VoltageAvg     float64  `json:"voltage_avg"`
			CurrentAvg     float64  `json:"current_avg"`
			PowerFactorAvg float64  `json:"power_factor_avg"`
		}
		items := []dayDTO{}
		if ms != nil {
			for _, d := range ms.Daily {
				if d.Date.Before(domain.DateOf(from)) || d.Date.After(to) {
					continue
				}
				var dev *float64
				if v := ms.Deviation(d); v != nil {
					p := *v * 100
					dev = domain.RoundPtr(&p, 1)
				}
				items = append(items, dayDTO{Date: domain.ISODate(d.Date), ConsumptionKWh: domain.Round(d.ConsumptionKWh, 1), DeviationPct: dev,
					VoltageAvg: domain.Round(d.VoltageAvg, 1), CurrentAvg: domain.Round(d.CurrentAvg, 1), PowerFactorAvg: domain.Round(d.PowerFactorAvg, 3)})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"meter_id": id, "granularity": "day", "items": items})
		return
	}

	type hourDTO struct {
		Timestamp      time.Time `json:"timestamp"`
		ConsumptionKWh float64   `json:"consumption_kwh"`
		VoltageV       float64   `json:"voltage_v"`
		CurrentA       float64   `json:"current_a"`
		PowerFactor    float64   `json:"power_factor"`
		Status         string    `json:"status"`
		IsOutlier      bool      `json:"is_outlier"`
	}
	type profileDTO struct {
		Hour    int     `json:"hour"`
		MeanKWh float64 `json:"mean_kwh"`
		StdKWh  float64 `json:"std_kwh"`
	}
	items := []hourDTO{}
	profile := []profileDTO{}
	if ms != nil {
		for _, rd := range ms.Readings {
			if rd.Timestamp.Before(from) || rd.Timestamp.After(to) {
				continue
			}
			items = append(items, hourDTO{Timestamp: rd.Timestamp.UTC(), ConsumptionKWh: rd.ConsumptionKWh, VoltageV: rd.VoltageV,
				CurrentA: rd.CurrentA, PowerFactor: rd.PowerFactor, Status: rd.Status, IsOutlier: ms.IsOutlier(rd)})
		}
		if !ms.Insufficient {
			for _, p := range ms.Profile {
				profile = append(profile, profileDTO{Hour: p.Hour, MeanKWh: domain.Round(p.Mean, 1), StdKWh: domain.Round(p.Std, 1)})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"meter_id": id, "granularity": "hour", "baseline_profile": profile, "items": items})
}
