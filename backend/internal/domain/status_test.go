package domain

import "testing"

// RF-M04: tabla 8.3 del maestro con las 6 combinaciones.
func TestDeriveMeterStatus(t *testing.T) {
	cases := []struct {
		typ  AnomalyType
		sev  Severity
		want MeterStatus
	}{
		{"", "", MeterOK},
		{FalsePositive, SeverityLow, MeterOK},
		{ExplainableAnomaly, SeverityMedium, MeterAlert},
		{DataQuality, SeverityHigh, MeterAlert},
		{RealAnomaly, SeverityMedium, MeterAlert},
		{RealAnomaly, SeverityHigh, MeterCritical},
	}
	for _, c := range cases {
		if got := DeriveMeterStatus(c.typ, c.sev); got != c.want {
			t.Errorf("DeriveMeterStatus(%q,%q) = %q, want %q", c.typ, c.sev, got, c.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	cases := map[string]string{
		FormatNumber(1048.8, 1):  "1.048,8",
		FormatNumber(155250.8, 1): "155.250,8",
		FormatNumber(-0.1, 1):    "−0,1",
		FormatNumber(0.744, 3):   "0,744",
		FormatSigned(110.5, 1):   "+110,5",
		FormatSigned(-36.8, 1):   "−36,8",
		FormatNumber(16, 0):      "16",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
}
