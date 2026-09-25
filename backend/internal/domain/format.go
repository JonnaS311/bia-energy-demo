package domain

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// FormatNumber formatea en es-CO: miles ".", decimal ",", signo menos U+2212.
func FormatNumber(v float64, dec int) string {
	neg := v < 0
	s := strconv.FormatFloat(math.Abs(v), 'f', dec, 64)
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	out := b.String()
	if frac != "" {
		out += "," + frac
	}
	if neg && strings.Trim(out, "0.,") != "" {
		out = "−" + out
	}
	return out
}

// FormatSigned formatea con signo explícito ("+110,5" / "−0,1").
func FormatSigned(v float64, dec int) string {
	r := Round(v, dec)
	if r < 0 {
		return FormatNumber(r, dec)
	}
	return "+" + FormatNumber(r, dec)
}

// ISODate formatea una fecha como YYYY-MM-DD.
func ISODate(t time.Time) string { return t.UTC().Format("2006-01-02") }

// ISODateTimeShort formatea como YYYY-MM-DD HH:MM (UTC).
func ISODateTimeShort(t time.Time) string { return t.UTC().Format("2006-01-02 15:04") }
