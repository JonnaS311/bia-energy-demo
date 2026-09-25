package engine_test

import (
	"testing"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// RF-E-06: los medidores sanos no alcanzan 6 outliers en ningún día, por eso el patrón horario no los
// marca. Valores reales del dataset: máximo 4 por día (M-107) y 14 en total (M-107).
func TestHealthyMetersStayBelowPatternThreshold(t *testing.T) {
	in := LoadDataset(t)
	ds := domain.BuildDataset(in.Readings, in.Events, nil, 7)
	for _, id := range []string{"M-101", "M-102", "M-103", "M-105", "M-107", "M-108", "M-110", "M-111"} {
		m := ds.Meters[id]
		perDay := map[string]int{}
		for _, r := range m.Readings {
			if m.IsOutlier(r) {
				perDay[domain.ISODate(r.Timestamp)]++
			}
		}
		for day, n := range perDay {
			if n >= 6 {
				t.Errorf("%s has %d outliers on %s (pattern threshold is 6)", id, n, day)
			}
		}
	}
}
