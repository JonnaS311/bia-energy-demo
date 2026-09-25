// Package ingest lee y valida readings.csv y events.csv (spec de data RF-D-01..RF-D-10).
package ingest

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
)

// Encabezados obligatorios exactos (RF-D-02).
var (
	ReadingsHeader = []string{"meter_id", "timestamp", "consumption_kwh", "voltage_v", "current_a", "power_factor", "status"}
	EventsHeader   = []string{"meter_id", "event_timestamp", "event_type", "description"}
)

// Códigos de rechazo (tabla 8.2).
const (
	ReasonColumnCount     = "COLUMN_COUNT_MISMATCH"
	ReasonInvalidMeterID  = "INVALID_METER_ID"
	ReasonInvalidTS       = "INVALID_TIMESTAMP"
	ReasonNotANumber      = "NOT_A_NUMBER"
	ReasonNegativeKWh     = "NEGATIVE_CONSUMPTION"
	ReasonVoltage         = "VOLTAGE_NOT_POSITIVE"
	ReasonNegativeCurrent = "NEGATIVE_CURRENT"
	ReasonPF              = "PF_OUT_OF_RANGE"
	ReasonEmptyEventType  = "EMPTY_EVENT_TYPE"
)

var meterIDRe = regexp.MustCompile(`^M-\d{3}$`)

// ErrSchemaMismatch indica un encabezado inválido: se aborta la ingesta de ese archivo.
type ErrSchemaMismatch struct {
	Expected, Found []string
}

func (e *ErrSchemaMismatch) Error() string {
	return fmt.Sprintf("INGEST_SCHEMA_MISMATCH: expected %v, found %v", e.Expected, e.Found)
}

// Rejection describe una fila rechazada.
type Rejection struct {
	Line    int
	Reason  string
	MeterID string
}

// ParseResult es el resultado de parsear un archivo.
type ParseResult[T any] struct {
	Rows         []T
	Rejected     []Rejection
	Duplicates   int
	ExtraColumns []string
	// DuplicatesByKey cuenta sobrescrituras por medidor y fecha (insumo del detector DQ (d)).
	DuplicatesByMeterDay map[string]map[string]int
}

// newReader descarta BOM, normaliza CRLF y reemplaza bytes UTF-8 inválidos (CB-D-17).
func newReader(r io.Reader) (*csv.Reader, error) {
	br := bufio.NewReader(r)
	if b, err := br.Peek(3); err == nil && bytes.Equal(b, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
	}
	data, err := io.ReadAll(br)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) {
		data = bytes.ToValidUTF8(data, []byte("�"))
	}
	cr := csv.NewReader(bytes.NewReader(data))
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	cr.ReuseRecord = false
	return cr, nil
}

func checkHeader(got, want []string) ([]string, error) {
	for i := range got {
		got[i] = strings.TrimSpace(got[i])
	}
	if len(got) < len(want) {
		return nil, &ErrSchemaMismatch{Expected: want, Found: got}
	}
	for i, w := range want {
		if got[i] != w {
			return nil, &ErrSchemaMismatch{Expected: want, Found: got}
		}
	}
	return got[len(want):], nil
}

func trimAll(rec []string) {
	for i := range rec {
		rec[i] = strings.TrimSpace(rec[i])
	}
}

func parseReadingTS(s string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
	return t, err == nil
}

func parseEventTS(s string) (time.Time, bool) {
	if t, err := time.ParseInLocation("2006-01-02 15:04", s, time.UTC); err == nil {
		return t, true
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
	return t, err == nil
}

func parseNum(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

// ParseReadings parsea readings.csv. Duplicados (meter_id, timestamp): la última fila gana.
func ParseReadings(r io.Reader) (*ParseResult[domain.Reading], error) {
	cr, err := newReader(r)
	if err != nil {
		return nil, err
	}
	header, err := cr.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, &ErrSchemaMismatch{Expected: ReadingsHeader, Found: nil}
		}
		return nil, err
	}
	extra, err := checkHeader(header, ReadingsHeader)
	if err != nil {
		return nil, err
	}
	res := &ParseResult[domain.Reading]{ExtraColumns: extra, DuplicatesByMeterDay: map[string]map[string]int{}}
	index := map[string]int{}
	line := 1
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		line++
		if err != nil {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonColumnCount})
			continue
		}
		if len(rec) == 1 && rec[0] == "" {
			continue
		}
		trimAll(rec)
		if len(rec) != len(header) {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonColumnCount, MeterID: first(rec)})
			continue
		}
		meterID := rec[0]
		if !meterIDRe.MatchString(meterID) {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonInvalidMeterID, MeterID: meterID})
			continue
		}
		ts, ok := parseReadingTS(rec[1])
		if !ok {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonInvalidTS, MeterID: meterID})
			continue
		}
		var nums [4]float64
		bad := false
		for i := 0; i < 4; i++ {
			v, ok := parseNum(rec[2+i])
			if !ok {
				bad = true
				break
			}
			nums[i] = v
		}
		if bad {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonNotANumber, MeterID: meterID})
			continue
		}
		reason := ""
		switch {
		case nums[0] < 0:
			reason = ReasonNegativeKWh
		case nums[1] <= 0:
			reason = ReasonVoltage
		case nums[2] < 0:
			reason = ReasonNegativeCurrent
		case nums[3] < 0 || nums[3] > 1:
			reason = ReasonPF
		}
		if reason != "" {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: reason, MeterID: meterID})
			continue
		}
		row := domain.Reading{MeterID: meterID, Timestamp: ts, ConsumptionKWh: nums[0], VoltageV: nums[1], CurrentA: nums[2], PowerFactor: nums[3], Status: rec[6]}
		key := meterID + "|" + ts.Format(time.RFC3339)
		if i, ok := index[key]; ok {
			res.Rows[i] = row
			res.Duplicates++
			day := domain.ISODate(ts)
			if res.DuplicatesByMeterDay[meterID] == nil {
				res.DuplicatesByMeterDay[meterID] = map[string]int{}
			}
			res.DuplicatesByMeterDay[meterID][day]++
			continue
		}
		index[key] = len(res.Rows)
		res.Rows = append(res.Rows, row)
	}
	return res, nil
}

// ParseEvents parsea events.csv. Duplicados (meter_id, timestamp, type): la última fila gana.
func ParseEvents(r io.Reader) (*ParseResult[domain.Event], error) {
	cr, err := newReader(r)
	if err != nil {
		return nil, err
	}
	header, err := cr.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, &ErrSchemaMismatch{Expected: EventsHeader, Found: nil}
		}
		return nil, err
	}
	extra, err := checkHeader(header, EventsHeader)
	if err != nil {
		return nil, err
	}
	res := &ParseResult[domain.Event]{ExtraColumns: extra}
	index := map[string]int{}
	line := 1
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		line++
		if err != nil {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonColumnCount})
			continue
		}
		if len(rec) == 1 && rec[0] == "" {
			continue
		}
		trimAll(rec)
		if len(rec) != len(header) {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonColumnCount, MeterID: first(rec)})
			continue
		}
		meterID := rec[0]
		if !meterIDRe.MatchString(meterID) {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonInvalidMeterID, MeterID: meterID})
			continue
		}
		ts, ok := parseEventTS(rec[1])
		if !ok {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonInvalidTS, MeterID: meterID})
			continue
		}
		if rec[2] == "" {
			res.Rejected = append(res.Rejected, Rejection{Line: line, Reason: ReasonEmptyEventType, MeterID: meterID})
			continue
		}
		ev := domain.Event{MeterID: meterID, Timestamp: ts, Type: rec[2], Description: rec[3]}
		key := meterID + "|" + ts.Format(time.RFC3339) + "|" + rec[2]
		if i, ok := index[key]; ok {
			res.Rows[i] = ev
			res.Duplicates++
			continue
		}
		index[key] = len(res.Rows)
		res.Rows = append(res.Rows, ev)
	}
	return res, nil
}

func first(rec []string) string {
	if len(rec) == 0 {
		return ""
	}
	return rec[0]
}
