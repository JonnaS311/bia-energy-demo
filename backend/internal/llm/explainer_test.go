package llm

// Fake de DeepSeek por HTTP (spec de pruebas RF-T-04): ningún test toca api.deepseek.com.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
)

const validAnswer = `{"reason":"El consumo diario de M-109 subió 110,5 % sobre su baseline de 1.048,8 kWh durante 3 días consecutivos sin evento operativo que lo explique.","recommended_action":"Investigar el medidor y la instalación.","explanation_points":["Consumo diario: 2.207,6 kWh frente a 1.048,8 kWh de baseline (+110,5 %).","Corriente media diaria: 420,5 A frente a 200,5 A (+109,7 %)."]}`

type fakeServer struct {
	srv      *httptest.Server
	requests atomic.Int32
	mu       sync.Mutex
	bodies   [][]byte
}

func completion(content, finish string) []byte {
	b, _ := json.Marshal(map[string]any{
		"id": "fake-1", "object": "chat.completion", "model": "deepseek-flash",
		"choices": []any{map[string]any{"index": 0, "finish_reason": finish, "message": map[string]any{"role": "assistant", "content": content}}},
		"usage":   map[string]any{"prompt_tokens": 812, "completion_tokens": 96, "total_tokens": 908},
	})
	return b
}

func newFake(t *testing.T, handler func(n int32, w http.ResponseWriter)) *fakeServer {
	f := &fakeServer{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.bodies = append(f.bodies, b)
		f.mu.Unlock()
		n := f.requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		handler(n, w)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func sampleInput() engine.ExplainInput {
	v := 2207.6
	b := 1048.8
	d := 110.5
	return engine.ExplainInput{
		MeterID: "M-109", Type: domain.RealAnomaly, Severity: domain.SeverityHigh, Confidence: 0.98,
		WindowStart: "2026-09-12", WindowEnd: "2026-09-14", BaselineWindow: "2026-09-01..2026-09-07",
		Evidence:      []engine.EvidenceItem{{Position: 1, Metric: "DAILY_CONSUMPTION", Observed: &v, Baseline: &b, DeltaPct: &d, Unit: "kWh", Window: "2026-09-14", Detail: "x"}},
		ActionDefault: "Investigar el medidor y la instalación.",
		Facts:         engine.Facts{DeltaPct: 110.5, Baseline: 1048.8, Observed: 2207.6, Days: 3, WindowStart: "2026-09-12", WindowEnd: "2026-09-14", PFObserved: 0.744, PFBaseline: 0.939, CurrentDeltaPct: 109.7},
	}
}

func newExplainer(t *testing.T, f *fakeServer, key string, timeout time.Duration) (*Explainer, *bytes.Buffer) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	url := ""
	if f != nil {
		url = f.srv.URL
	}
	e := NewExplainer(Config{APIKey: key, BaseURL: url, Model: "deepseek-flash", Timeout: timeout}, log)
	e.retryWait = 10 * time.Millisecond
	return e, &buf
}

func TestExplainerScenarios(t *testing.T) {
	cases := []struct {
		name     string
		handler  func(n int32, w http.ResponseWriter)
		wantReqs int32
		wantSrc  string
		wantLog  string
	}{
		{"ok", func(_ int32, w http.ResponseWriter) { _, _ = w.Write(completion(validAnswer, "stop")) }, 1, "llm", `"outcome":"ok"`},
		{"ok_extra_fields", func(_ int32, w http.ResponseWriter) {
			_, _ = w.Write(completion(strings.TrimSuffix(validAnswer, "}")+`,"severity":"LOW","type":"FALSE_POSITIVE"}`, "stop"))
		}, 1, "llm", `"outcome":"ok"`},
		{"invalid_json", func(_ int32, w http.ResponseWriter) { _, _ = w.Write(completion("Claro, aquí está la explicación...", "stop")) }, 2, "template", `"cause":"invalid_json"`},
		{"schema_invalid", func(_ int32, w http.ResponseWriter) {
			_, _ = w.Write(completion(`{"reason":"Texto suficientemente largo para validar.","recommended_action":"Investigar el medidor.","explanation_points":["solo un punto aquí"]}`, "stop"))
		}, 2, "template", `"cause":"schema_invalid"`},
		{"finish_length", func(_ int32, w http.ResponseWriter) { _, _ = w.Write(completion(validAnswer, "length")) }, 2, "template", `"cause":"schema_invalid"`},
		{"unauthorized", func(_ int32, w http.ResponseWriter) {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error":{"message":"bad key","type":"auth"}}`))
		}, 1, "template", `"cause":"http_error"`},
		{"rate_limited_then_ok", func(n int32, w http.ResponseWriter) {
			if n == 1 {
				w.WriteHeader(429)
				_, _ = w.Write([]byte(`{"error":{"message":"slow down"}}`))
				return
			}
			_, _ = w.Write(completion(validAnswer, "stop"))
		}, 2, "llm", `"outcome":"ok"`},
		{"server_error", func(_ int32, w http.ResponseWriter) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
		}, 2, "template", `"cause":"http_error"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFake(t, c.handler)
			e, logs := newExplainer(t, f, "test", 2*time.Second)
			out := e.Explain(context.Background(), sampleInput())
			if got := f.requests.Load(); got != c.wantReqs {
				t.Errorf("requests = %d, want %d", got, c.wantReqs)
			}
			if out.Source != c.wantSrc {
				t.Errorf("source = %s, want %s", out.Source, c.wantSrc)
			}
			if out.Reason == "" || out.RecommendedAction == "" || len(out.Points) < 2 {
				t.Errorf("incomplete output %+v", out)
			}
			if !strings.Contains(logs.String(), c.wantLog) {
				t.Errorf("log missing %s: %s", c.wantLog, logs.String())
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			for _, b := range f.bodies {
				if len(b) > 8192 {
					t.Errorf("request too large: %d", len(b))
				}
			}
		})
	}
}

func TestExplainerTimeout(t *testing.T) {
	f := newFake(t, func(_ int32, w http.ResponseWriter) {
		time.Sleep(400 * time.Millisecond)
		_, _ = w.Write(completion(validAnswer, "stop"))
	})
	e, logs := newExplainer(t, f, "test", 100*time.Millisecond)
	start := time.Now()
	out := e.Explain(context.Background(), sampleInput())
	if out.Source != "template" || f.requests.Load() != 2 || !strings.Contains(logs.String(), `"cause":"timeout"`) {
		t.Fatalf("source=%s reqs=%d logs=%s", out.Source, f.requests.Load(), logs.String())
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("took %s", time.Since(start))
	}
}

func TestExplainerNoKey(t *testing.T) {
	e, logs := newExplainer(t, nil, "", time.Second)
	out := e.Explain(context.Background(), sampleInput())
	if out.Source != "template" || !strings.Contains(logs.String(), `"outcome":"fallback"`) || !strings.Contains(logs.String(), CauseNoAPIKey) {
		t.Fatalf("out=%+v logs=%s", out, logs.String())
	}
}

// CB-E-16: tras un 401 el resto de anomalías del análisis usa plantilla sin llamar.
func TestExplainerDisabledAfter401(t *testing.T) {
	f := newFake(t, func(_ int32, w http.ResponseWriter) { w.WriteHeader(401); _, _ = w.Write([]byte(`{}`)) })
	e, _ := newExplainer(t, f, "test", time.Second)
	e.Explain(context.Background(), sampleInput())
	e.Explain(context.Background(), sampleInput())
	if f.requests.Load() != 1 {
		t.Fatalf("requests = %d", f.requests.Load())
	}
}

// RNF-E-03 / RNF-09: payload ≤ 4 KB y sin lecturas crudas.
func TestPayloadSize(t *testing.T) {
	b, err := BuildPayload(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > 4096 || strings.Contains(string(b), `"readings"`) {
		t.Fatalf("payload %d bytes: %s", len(b), b)
	}
}

func TestTemplatesPerType(t *testing.T) {
	for _, typ := range []domain.AnomalyType{domain.RealAnomaly, domain.ExplainableAnomaly, domain.FalsePositive, domain.DataQuality} {
		in := sampleInput()
		in.Type = typ
		in.ActionDefault = engine.ActionDefault[typ]
		out := Template(in)
		if out.Reason == "" || len(out.Points) < 2 || out.RecommendedAction != engine.ActionDefault[typ] || strings.Contains(out.Reason, "{") {
			t.Errorf("%s: %+v", typ, out)
		}
	}
}
