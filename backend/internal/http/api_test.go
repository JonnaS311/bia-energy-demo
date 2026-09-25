package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/jsotelo/energy-platform/backend/internal/domain"
	"github.com/jsotelo/energy-platform/backend/internal/engine"
)

func TestHealthAndIngest(t *testing.T) { // RF-B-16, RF-D-07
	e := newEnv(t, Options{})
	r := e.do("GET", "/health", nil, nil)
	if r.Code != 200 {
		t.Fatalf("health %d", r.Code)
	}
	ing := r.json(t)["ingest"].(map[string]any)
	if ing["status"] != "COMPLETED" || ing["readings"] != 4032.0 || ing["events"] != 4.0 || ing["meters"] != 12.0 || ing["rejected"] != 0.0 || ing["duplicates"] != 0.0 {
		t.Fatalf("ingest %+v", ing)
	}
	// RF-D-05: una segunda ingesta deja los mismos conteos.
	rd, ev, me, _ := e.st.Counts(context.Background())
	if rd != 4032 || ev != 4 || me != 12 {
		t.Fatalf("counts %d %d %d", rd, ev, me)
	}
}

func TestLogin(t *testing.T) { // RF-B-04, RF-B-05
	e := newEnv(t, Options{})
	r := e.do("POST", "/auth/login", map[string]string{"email": "analista@energy.local", "password": "Demo1234!"}, nil)
	body := r.json(t)
	tok, _ := jwt.Parse(body["token"].(string), func(*jwt.Token) (any, error) { return []byte("test-secret"), nil })
	c := tok.Claims.(jwt.MapClaims)
	if c["sub"] != "analista@energy.local" || c["exp"].(float64)-c["iat"].(float64) != 28800 || c["name"] != "Analista Demo" {
		t.Fatalf("claims %+v", c)
	}
	bad := e.do("POST", "/auth/login", map[string]string{"email": "analista@energy.local", "password": "incorrecta"}, nil)
	none := e.do("POST", "/auth/login", map[string]string{"email": "nadie@energy.local", "password": "x"}, nil)
	want := `{"error":{"code":"INVALID_CREDENTIALS","message":"Credenciales inválidas","details":null}}`
	if bad.Code != 401 || strings.TrimSpace(string(bad.Body)) != want || strings.TrimSpace(string(none.Body)) != want {
		t.Fatalf("bad=%d %s none=%s", bad.Code, bad.Body, none.Body)
	}
	if strings.Contains(e.logs.String(), "Demo1234!") {
		t.Fatal("password leaked to logs")
	}
	// Rate limit: 11 intentos fallidos desde la misma IP → 429.
	var last resp
	for i := 0; i < 11; i++ {
		last = e.do("POST", "/auth/login", map[string]string{"email": "x@y", "password": "z"}, map[string]string{"X-Forwarded-For": "10.9.9.9"})
	}
	if last.Code != 429 || last.json(t)["error"].(map[string]any)["code"] != CodeTooManyAttempts {
		t.Fatalf("rate limit %d %s", last.Code, last.Body)
	}
}

func TestAuthReasons(t *testing.T) { // RF-B-05
	e := newEnv(t, Options{})
	other, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "a", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("otro"))
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "a", "exp": time.Now().Add(-time.Hour).Unix()}).SignedString([]byte("test-secret"))
	noneTok, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "a", "exp": time.Now().Add(time.Hour).Unix()}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	cases := map[string]string{"Bearer " + other: "bad_signature", "Bearer " + expired: "expired", "Bearer " + noneTok: "bad_alg", "Basic xxx": "malformed", "": "missing"}
	for header, reason := range cases {
		h := map[string]string{}
		if header != "" {
			h["Authorization"] = header
		}
		r := e.do("GET", "/meters", nil, h)
		if r.Code != 401 || r.json(t)["error"].(map[string]any)["message"] != "Token inválido o expirado" {
			t.Errorf("%s → %d %s", reason, r.Code, r.Body)
		}
		if !strings.Contains(e.logs.String(), `"reason":"`+reason+`"`) {
			t.Errorf("log without reason %s", reason)
		}
	}
	if r := e.do("GET", "/health", nil, nil); r.Code != 200 {
		t.Errorf("health without auth = %d", r.Code)
	}
}

func TestMiddleware(t *testing.T) { // RF-B-03, RF-B-17, CB-B-*
	e := newEnv(t, Options{})
	r := e.get("/meters")
	if r.Header.Get("X-Request-Id") == "" || !strings.Contains(e.logs.String(), r.Header.Get("X-Request-Id")) {
		t.Error("X-Request-Id missing or not logged")
	}
	if !strings.Contains(e.logs.String(), `"user":"analista@energy.local"`) || !strings.Contains(e.logs.String(), `"path":"/meters"`) {
		t.Error("access log incomplete")
	}
	// CORS.
	ok := e.do("OPTIONS", "/meters", nil, map[string]string{"Origin": "http://localhost:5173"})
	evil := e.do("OPTIONS", "/meters", nil, map[string]string{"Origin": "http://evil.local"})
	if ok.Code != 204 || ok.Header.Get("Access-Control-Allow-Origin") != "http://localhost:5173" || evil.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("cors ok=%d %v evil=%v", ok.Code, ok.Header, evil.Header)
	}
	// Recover.
	rec := httptest.NewRecorder()
	e.srv.recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 500 || strings.TrimSpace(rec.Body.String()) != `{"error":{"code":"INTERNAL_ERROR","message":"Error interno","details":null}}` || !strings.Contains(e.logs.String(), "boom") {
		t.Errorf("recover %d %s", rec.Code, rec.Body)
	}
	// 404 / 405.
	if r := e.get("/no-existe"); r.Code != 404 || r.json(t)["error"].(map[string]any)["code"] != CodeNotFound {
		t.Errorf("404 %d %s", r.Code, r.Body)
	}
	if r := e.do("DELETE", "/meters", nil, map[string]string{"Authorization": "Bearer " + e.token}); r.Code != 405 {
		t.Errorf("405 %d", r.Code)
	}
	// 415 / 400 / 413.
	if r := e.do("POST", "/auth/login", nil, map[string]string{"Content-Type": "text/plain"}); r.Code != 415 {
		t.Errorf("415 %d", r.Code)
	}
	if r := e.do("POST", "/auth/login", "{bad", nil); r.Code != 400 || !strings.Contains(string(r.Body), `"reason":"invalid JSON"`) {
		t.Errorf("malformed %d %s", r.Code, r.Body)
	}
	big := `{"email":"` + strings.Repeat("a", 1<<20) + `"}`
	if r := e.do("POST", "/auth/login", big, nil); r.Code != 413 {
		t.Errorf("413 %d", r.Code)
	}
	// Timeout 503.
	ts := NewServer(e.cfg, e.st, e.srv.tracker, e.srv.log, Options{RequestTimeout: 50 * time.Millisecond})
	rec = httptest.NewRecorder()
	ts.timeout(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), CodeRequestTimeout) {
		t.Errorf("timeout %d %s", rec.Code, rec.Body)
	}
}

func TestMetersBeforeAnalysis(t *testing.T) { // RF-B-06, RF-B-07, CB-M06
	e := newEnv(t, Options{})
	m := e.get("/meters").json(t)
	if m["total"] != 12.0 {
		t.Fatalf("total %v", m["total"])
	}
	for _, it := range items(t, m) {
		if it["status"] != "OK" || it["anomaly"] != nil {
			t.Fatalf("before analysis %+v", it)
		}
	}
	// "10" aparece en M-101…M-109 y también en M-110 (subcadena "1-10"): 10 medidores.
	if got := ids(t, e.get("/meters?q=10").json(t)); len(got) != 10 || got[0] != "M-101" || got[9] != "M-110" {
		t.Errorf("q=10 %v", got)
	}
	cons := ids(t, e.get("/meters?sort=consumption&order=desc").json(t))
	if cons[0] != "M-109" || cons[11] != "M-107" {
		t.Errorf("consumption %v", cons)
	}
	r := e.get("/meters?sort=foo")
	d := r.json(t)["error"].(map[string]any)["details"].(map[string]any)
	if r.Code != 400 || d["param"] != "sort" || !reflect.DeepEqual(d["allowed"], []any{"consumption", "variation", "severity"}) {
		t.Errorf("sort=foo %d %s", r.Code, r.Body)
	}
	if r := e.get("/meters?q=" + strings.Repeat("x", 21)); r.Code != 400 {
		t.Errorf("q length %d", r.Code)
	}

	det := e.get("/meters/M-109").json(t)
	daily := det["daily"].([]any)
	last := daily[13].(map[string]any)
	elec := det["electrical"].(map[string]any)["current_a"].(map[string]any)
	if len(daily) != 14 || last["date"] != "2026-09-14" || last["consumption_kwh"] != 2207.6 || last["deviation_pct"] != 110.5 ||
		elec["current_avg"] != 420.5 || elec["baseline_avg"] != 200.5 || len(det["events"].([]any)) != 1 ||
		det["baseline_daily_kwh"] != 1048.8 || det["variation_pct"] != 110.5 || det["current_consumption_kwh"] != 2207.6 {
		t.Errorf("detail %s", e.get("/meters/M-109").Body)
	}
	if r := e.get("/meters/m-109"); r.Code != 404 {
		t.Errorf("lowercase %d", r.Code)
	}
	if r := e.get("/meters/M-999"); r.Code != 404 || r.json(t)["error"].(map[string]any)["message"] != "No existe el medidor M-999" {
		t.Errorf("M-999 %d %s", r.Code, r.Body)
	}
}

func TestReadings(t *testing.T) { // RF-B-08
	e := newEnv(t, Options{})
	all := e.get("/meters/M-109/readings").json(t)
	prof := all["baseline_profile"].([]any)
	if len(items(t, all)) != 336 || len(prof) != 24 || prof[0].(map[string]any)["mean_kwh"] != 31.7 || prof[0].(map[string]any)["std_kwh"] != 1.6 {
		t.Fatalf("hourly items=%d profile=%v", len(items(t, all)), prof[0])
	}
	day := e.get("/meters/M-109/readings?from=2026-09-14&to=2026-09-14").json(t)
	found := false
	for _, it := range items(t, day) {
		if it["timestamp"] == "2026-09-14T13:00:00Z" {
			found = it["consumption_kwh"] == 117.51 && it["is_outlier"] == true
		}
	}
	if len(items(t, day)) != 24 || !found {
		t.Errorf("one day: %d found=%v", len(items(t, day)), found)
	}
	d := e.get("/meters/M-109/readings?granularity=day")
	if len(items(t, d.json(t))) != 14 || strings.Contains(string(d.Body), "baseline_profile") {
		t.Errorf("day granularity %s", d.Body)
	}
	if n := len(items(t, e.get("/meters/M-109/readings?from=2026-09-20&to=2026-09-21").json(t))); n != 0 {
		t.Errorf("empty range %d", n)
	}
	if r := e.get("/meters/M-109/readings?from=2026-09-14&to=2026-09-01"); r.Code != 400 {
		t.Errorf("from>to %d", r.Code)
	}
	if r := e.get("/meters/M-109/readings?from=14/09/2026"); r.Code != 400 || r.json(t)["error"].(map[string]any)["details"].(map[string]any)["param"] != "from" {
		t.Errorf("bad from %d %s", r.Code, r.Body)
	}
	// M-101 tiene exactamente 1 outlier en el periodo (2026-09-11 22:00, z = 3,13).
	var outliers []string
	for _, it := range items(t, e.get("/meters/M-101/readings").json(t)) {
		if it["is_outlier"] == true {
			outliers = append(outliers, it["timestamp"].(string))
		}
	}
	if !reflect.DeepEqual(outliers, []string{"2026-09-11T22:00:00Z"}) {
		t.Errorf("M-101 outliers %v", outliers)
	}
}

func TestAnalysisEndToEnd(t *testing.T) { // RF-B-09..RF-B-15, RF-M02, RF-M05
	e := newEnv(t, Options{})
	// Sin análisis.
	if r := e.get("/anomalies"); strings.TrimSpace(string(r.Body)) != `{"analysis_id":null,"items":[],"total":0}` {
		t.Errorf("empty anomalies %s", r.Body)
	}
	dash := e.get("/dashboard/summary").json(t)
	if dash["anomalies_total"] != 0.0 || dash["ai_confidence_avg"] != nil || dash["last_analysis"] != nil || len(dash["top_anomalies"].([]any)) != 0 {
		t.Errorf("dashboard before %v", dash)
	}

	start := time.Now()
	a := e.runAnalysis()
	if d := time.Since(start); d > 15*time.Second {
		t.Errorf("RNF-01: analysis took %s", d)
	}
	t.Logf("analysis duration: %s (limit 15s)", time.Since(start))
	sum := a["summary"].(map[string]any)
	if a["status"] != "COMPLETED" || sum["anomalies_detected"] != 4.0 || sum["high_priority"] != 2.0 || sum["meters_analyzed"] != 12.0 || sum["explanation_source"] != "template" {
		t.Fatalf("analysis %+v", a)
	}
	for i, s := range a["steps"].([]any) {
		st := s.(map[string]any)
		if st["status"] != "COMPLETED" || st["started_at"] == nil || st["finished_at"] == nil || st["key"] != domain.StepDefs[i].Key {
			t.Errorf("step %d %+v", i, st)
		}
	}
	if n := strings.Count(e.logs.String(), `"outcome":"fallback"`); n != 4 {
		t.Errorf("RF-B-18: %d fallback log lines, want 4", n)
	}

	// Anomalías en orden.
	an := e.get("/anomalies").json(t)
	if got := ids(t, an); !reflect.DeepEqual(got, []string{"M-109", "M-112", "M-104", "M-106"}) {
		t.Fatalf("anomalies order %v", got)
	}
	if got := ids(t, e.get("/anomalies?severity=HIGH").json(t)); !reflect.DeepEqual(got, []string{"M-109", "M-112"}) {
		t.Errorf("severity=HIGH %v", got)
	}
	if got := ids(t, e.get("/anomalies?type=DATA_QUALITY").json(t)); !reflect.DeepEqual(got, []string{"M-112"}) {
		t.Errorf("type filter %v", got)
	}

	// Medidores tras el análisis.
	if got := ids(t, e.get("/meters").json(t)); !reflect.DeepEqual(got, []string{"M-109", "M-112", "M-104", "M-106", "M-101", "M-102", "M-103", "M-105", "M-107", "M-108", "M-110", "M-111"}) {
		t.Errorf("meters order %v", got)
	}
	if got := ids(t, e.get("/meters?status=CRITICAL").json(t)); !reflect.DeepEqual(got, []string{"M-109"}) {
		t.Errorf("CRITICAL %v", got)
	}
	if got := ids(t, e.get("/meters?status=CRITICAL&status=ALERT").json(t)); !reflect.DeepEqual(got, []string{"M-109", "M-112", "M-104"}) {
		t.Errorf("CRITICAL|ALERT %v", got)
	}

	// Detalle M-109 y M-104.
	list := items(t, an)
	id109, id104 := list[0]["id"].(string), list[2]["id"].(string)
	det := e.get("/anomalies/" + id109).json(t)
	ev := det["evidence"].([]any)
	cmp := det["comparison"].(map[string]any)["consumption_kwh"].(map[string]any)
	if len(ev) < 3 || ev[0].(map[string]any)["position"] != 1.0 || cmp["observed"] != 2207.6 || det["related_event"] != nil || det["superseded"] != false || det["confidence"] != 0.98 {
		t.Errorf("detail 109 %s", e.get("/anomalies/"+id109).Body)
	}
	if det104 := e.get("/anomalies/" + id104).json(t); det104["related_event"].(map[string]any)["type"] != "OPERATIONAL_CHANGE" {
		t.Errorf("104 related %v", det104["related_event"])
	}
	if r := e.get("/anomalies/no-es-uuid"); r.Code != 404 {
		t.Errorf("bad uuid %d", r.Code)
	}

	// Dashboard.
	dash = e.get("/dashboard/summary").json(t)
	daily := dash["daily_consumption"].([]any)
	bs := dash["meters_by_status"].(map[string]any)
	total := dash["total_consumption_kwh"].(float64)
	if dash["meters_total"] != 12.0 || total < 155250.3 || total > 155251.3 || dash["anomalies_total"] != 4.0 || dash["high_priority_total"] != 2.0 ||
		dash["ai_confidence_avg"] != 0.93 || bs["OK"] != 9.0 || bs["ALERT"] != 2.0 || bs["CRITICAL"] != 1.0 || len(daily) != 14 ||
		daily[0].(map[string]any)["consumption_kwh"] != 10776.0 || daily[13].(map[string]any)["consumption_kwh"] != 12502.4 ||
		dash["top_anomalies"].([]any)[0].(map[string]any)["meter_id"] != "M-109" {
		t.Errorf("dashboard %s", e.get("/dashboard/summary").Body)
	}
	t.Logf("total_consumption_kwh = %v", total)

	// PATCH.
	auth := map[string]string{"Authorization": "Bearer " + e.token}
	p := e.do("PATCH", "/anomalies/"+id109+"/status", map[string]string{"status": "INVESTIGATING"}, auth)
	if p.Code != 200 || p.json(t)["status"] != "INVESTIGATING" || e.get("/anomalies/"+id109).json(t)["status"] != "INVESTIGATING" {
		t.Errorf("patch %d %s", p.Code, p.Body)
	}
	if p := e.do("PATCH", "/anomalies/"+id109+"/status", map[string]string{"status": "CLOSED"}, auth); p.Code != 400 ||
		!reflect.DeepEqual(p.json(t)["error"].(map[string]any)["details"].(map[string]any)["allowed"], []any{"OPEN", "INVESTIGATING", "RESOLVED", "DISMISSED"}) {
		t.Errorf("patch invalid %d %s", p.Code, p.Body)
	}
	if p := e.do("PATCH", "/anomalies/"+id109+"/status", `{"status":"OPEN","extra":1}`, auth); p.Code != 400 || p.json(t)["error"].(map[string]any)["details"].(map[string]any)["field"] != "extra" {
		t.Errorf("patch extra %d %s", p.Code, p.Body)
	}

	// Segundo análisis: supersede.
	a2 := e.runAnalysis()
	an2 := e.get("/anomalies").json(t)
	if an2["analysis_id"] != a2["id"] || an2["total"] != 4.0 {
		t.Errorf("second analysis %v", an2["analysis_id"])
	}
	if all := e.get("/anomalies?include_superseded=true").json(t); all["total"] != 8.0 {
		t.Errorf("include_superseded total %v", all["total"])
	}
	old := e.get("/anomalies/" + id109).json(t)
	if old["superseded"] != true {
		t.Errorf("old should be superseded")
	}
	if p := e.do("PATCH", "/anomalies/"+id109+"/status", map[string]string{"status": "DISMISSED"}, auth); p.Code != 200 {
		t.Errorf("patch superseded %d", p.Code)
	}
	// RNF-B-09: determinismo entre análisis.
	for i, x := range items(t, an) {
		y := items(t, an2)[i]
		for _, k := range []string{"meter_id", "type", "severity", "confidence", "priority_rank", "window_start", "window_end"} {
			if x[k] != y[k] {
				t.Errorf("determinism %s: %v vs %v", k, x[k], y[k])
			}
		}
	}
	if r := e.get("/ai/analysis/00000000-0000-0000-0000-000000000000"); r.Code != 404 || r.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("analysis 404 %d %v", r.Code, r.Header)
	}
}

func TestAnalyzeConcurrency(t *testing.T) { // RF-B-11, CB-B-10
	block := make(chan struct{})
	e := newEnv(t, Options{Analyzer: func(*slog.Logger) Analyzer { return blockingAnalyzer{block} }})
	var wg sync.WaitGroup
	codes := make(chan int, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- e.do("POST", "/ai/analyze", nil, map[string]string{"Authorization": "Bearer " + e.token}).Code
		}()
	}
	wg.Wait()
	close(codes)
	count := map[int]int{}
	for c := range codes {
		count[c]++
	}
	if count[202] != 1 || count[409] != 9 {
		t.Fatalf("codes %v", count)
	}
	r := e.do("POST", "/ai/analyze", nil, map[string]string{"Authorization": "Bearer " + e.token})
	if r.Code != 409 || r.json(t)["error"].(map[string]any)["details"].(map[string]any)["analysis_id"] == "" {
		t.Errorf("409 details %s", r.Body)
	}
	close(block)
}

type blockingAnalyzer struct{ ch chan struct{} }

func (b blockingAnalyzer) Run(ctx context.Context, in engine.Input, cb engine.StepCallback) (engine.Result, error) {
	<-b.ch
	return engine.Result{}, errors.New("stopped")
}

type failingAnalyzer struct{ panicky bool }

func (f failingAnalyzer) Run(_ context.Context, _ engine.Input, cb engine.StepCallback) (engine.Result, error) {
	for _, k := range []string{"READINGS", "BASELINE"} {
		_ = cb(k, domain.StepRunning, "")
		_ = cb(k, domain.StepCompleted, "ok")
	}
	_ = cb("DETECTION", domain.StepRunning, "")
	if f.panicky {
		panic("engine exploded")
	}
	return engine.Result{}, fmt.Errorf("detection failed")
}

func TestJobFailureKeepsPreviousAnomalies(t *testing.T) { // RF-B-12, CB-M04
	e := newEnv(t, Options{})
	e.runAnalysis()
	for _, panicky := range []bool{false, true} {
		fe := NewServer(e.cfg, e.st, e.srv.tracker, e.srv.log, Options{Analyzer: func(*slog.Logger) Analyzer { return failingAnalyzer{panicky} }})
		r := httptestDo(fe.Router(), "POST", "/ai/analyze", e.token)
		if r.Code != 202 {
			t.Fatalf("analyze %d %s", r.Code, r.Body)
		}
		var body map[string]any
		_ = json.Unmarshal(r.Body.Bytes(), &body)
		fe.WaitJobs()
		a := e.get("/ai/analysis/" + body["analysis_id"].(string)).json(t)
		steps := a["steps"].([]any)
		if a["status"] != "FAILED" || steps[2].(map[string]any)["status"] != "FAILED" || steps[3].(map[string]any)["status"] != "PENDING" || a["error_message"] == "" {
			t.Errorf("panicky=%v analysis %+v", panicky, a)
		}
		if an := e.get("/anomalies").json(t); an["total"] != 4.0 {
			t.Errorf("previous anomalies lost: %v", an["total"])
		}
		dash := e.get("/dashboard/summary").json(t)
		if dash["last_analysis"].(map[string]any)["status"] != "FAILED" || dash["anomalies_total"] != 4.0 {
			t.Errorf("dashboard after FAILED %v", dash["last_analysis"])
		}
	}
	if r := e.do("GET", "/health", nil, nil); r.Code != 200 {
		t.Error("process should stay healthy after panic")
	}
}

func TestOrphanAnalysis(t *testing.T) { // RF-B-02 paso 4, CB-B-09
	e := newEnv(t, Options{})
	ctx := context.Background()
	id, err := e.st.CreateAnalysis(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_ = e.st.MarkRunning(ctx, id)
	if n, _ := e.st.FailOrphanAnalyses(ctx); n != 1 {
		t.Fatalf("orphans %d", n)
	}
	a, _ := e.st.GetAnalysis(ctx, id)
	if a.Status != domain.AnalysisFailed || a.ErrorMessage == nil || *a.ErrorMessage != "process restarted" || a.FinishedAt == nil {
		t.Fatalf("orphan %+v", a)
	}
}

func TestIngestGate(t *testing.T) { // RF-D-01, RF-B-16
	e := newEnv(t, Options{})
	st := e.srv.tracker.Get()
	st.Status = "RUNNING"
	e.srv.tracker.Set(st)
	if r := e.get("/meters"); r.Code != 503 || r.json(t)["error"].(map[string]any)["code"] != CodeIngestInProgress {
		t.Errorf("gate %d %s", r.Code, r.Body)
	}
	if r := e.do("GET", "/health", nil, nil); r.Code != 200 || r.json(t)["ingest"].(map[string]any)["status"] != "RUNNING" {
		t.Errorf("health during ingest %d %s", r.Code, r.Body)
	}
}

func httptestDo(h http.Handler, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
