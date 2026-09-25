package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/jsotelo/energy-platform/backend/internal/config"
	"github.com/jsotelo/energy-platform/backend/internal/ingest"
	"github.com/jsotelo/energy-platform/backend/internal/store"
)

// Los tests de integración usan TEST_DATABASE_URL (spec de pruebas RF-T-05). Sin ella se saltan.

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type env struct {
	t      *testing.T
	srv    *Server
	h      http.Handler
	st     *store.Store
	logs   *syncBuffer
	token  string
	cfg    config.Config
}

func testConfig(dbURL string) config.Config {
	return config.Config{DatabaseURL: dbURL, JWTSecret: "test-secret", DemoUserEmail: "analista@energy.local", DemoUserPassword: "Demo1234!",
		CORSOrigins: []string{"http://localhost:5173"}, BaselineDays: 7, LLMTimeout: time.Second}
}

var dbMu sync.Mutex

// newEnv reinicia la BD, aplica migraciones, ingesta los CSV reales y siembra el usuario demo.
func newEnv(t *testing.T, opts Options) *env {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("skipped: TEST_DATABASE_URL not set")
	}
	dbMu.Lock()
	t.Cleanup(dbMu.Unlock)
	ctx := context.Background()
	logs := &syncBuffer{}
	log := slog.New(slog.NewJSONHandler(logs, nil))
	st, err := store.Connect(ctx, dbURL, store.ConnectOptions{Attempts: 3, Interval: time.Second}, log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if _, err := st.Pool().Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(dbURL); err != nil {
		t.Fatal(err)
	}
	tr := ingest.NewTracker()
	dataDir, _ := filepath.Abs(filepath.Join("..", "..", "..", "data"))
	if s := ingest.Run(ctx, st, dataDir, tr, log); s.Status != ingest.StatusCompleted {
		t.Fatalf("ingest %+v", s)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("Demo1234!"), 4)
	if err := st.SeedUser(ctx, "analista@energy.local", "Analista Demo", string(hash)); err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(dbURL)
	srv := NewServer(cfg, st, tr, log, opts)
	e := &env{t: t, srv: srv, h: srv.Router(), st: st, logs: logs, cfg: cfg}
	t.Cleanup(srv.WaitJobs)
	e.token = e.login()
	return e
}

type resp struct {
	Code   int
	Header http.Header
	Body   []byte
}

func (r resp) json(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("invalid json %q: %v", r.Body, err)
	}
	return m
}

func (e *env) do(method, path string, body any, headers map[string]string) resp {
	e.t.Helper()
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rd = bytes.NewBufferString(b)
	default:
		buf, _ := json.Marshal(b)
		rd = bytes.NewBuffer(buf)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return resp{Code: rec.Code, Header: rec.Header(), Body: rec.Body.Bytes()}
}

func (e *env) get(path string) resp {
	return e.do("GET", path, nil, map[string]string{"Authorization": "Bearer " + e.token})
}

func (e *env) login() string {
	r := e.do("POST", "/auth/login", map[string]string{"email": "analista@energy.local", "password": "Demo1234!"}, nil)
	if r.Code != 200 {
		e.t.Fatalf("login %d %s", r.Code, r.Body)
	}
	return r.json(e.t)["token"].(string)
}

// runAnalysis dispara un análisis y espera a que el job termine.
func (e *env) runAnalysis() map[string]any {
	e.t.Helper()
	r := e.do("POST", "/ai/analyze", nil, map[string]string{"Authorization": "Bearer " + e.token})
	if r.Code != 202 {
		e.t.Fatalf("analyze %d %s", r.Code, r.Body)
	}
	id := r.json(e.t)["analysis_id"].(string)
	e.srv.WaitJobs()
	g := e.get("/ai/analysis/" + id)
	return g.json(e.t)
}

func items(t *testing.T, m map[string]any) []map[string]any {
	t.Helper()
	raw, _ := m["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, x := range raw {
		out = append(out, x.(map[string]any))
	}
	return out
}

func ids(t *testing.T, m map[string]any) []string {
	var out []string
	for _, it := range items(t, m) {
		out = append(out, it["meter_id"].(string))
	}
	return out
}
