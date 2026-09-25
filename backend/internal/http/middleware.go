package http

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ctxKey int

const reqInfoKey ctxKey = 1

// reqInfo viaja en el contexto para que el log de acceso conozca el usuario autenticado.
type reqInfo struct {
	id   string
	user string
}

func info(r *http.Request) *reqInfo {
	if v, ok := r.Context().Value(reqInfoKey).(*reqInfo); ok {
		return v
	}
	return &reqInfo{}
}

var reqIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)

// requestID (1): reutiliza X-Request-Id válido o genera un UUID v4 (CB-B-19).
func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !reqIDRe.MatchString(id) {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), reqInfoKey, &reqInfo{id: id})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func remoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// accessLog (2): una línea JSON por petición (RF-B-18).
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		ri := info(r)
		s.log.Info("http", "request_id", ri.id, "method", r.Method, "path", r.URL.Path, "status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(), "user", ri.user, "remote_ip", remoteIP(r))
	})
}

// recoverer (3): pánico → 500 sin filtrar detalles; stack al log.
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				s.log.Error("panic", "request_id", info(r).id, "panic", rec, "stack", string(debug.Stack()))
				writeError(w, http.StatusInternalServerError, CodeInternal, "Error interno", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// cors (4): orígenes exactos de CORS_ALLOWED_ORIGINS; OPTIONS → 204.
func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.cfg.CORSOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")
			h.Set("Access-Control-Expose-Headers", "X-Request-Id")
			h.Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// bodyLimit (5): > 1 MB → 413 (CB-B-13).
func (s *Server) bodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBodyBytes {
			writeError(w, http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "Cuerpo demasiado grande", map[string]any{"max_bytes": maxBodyBytes})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// timeout (6): cancela a los 30 s y responde 503 REQUEST_TIMEOUT (CB-B-14).
func (s *Server) timeout(next http.Handler) http.Handler {
	body := `{"error":{"code":"REQUEST_TIMEOUT","message":"La petición excedió el tiempo máximo","details":null}}`
	th := http.TimeoutHandler(next, s.reqTimeout, body)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		th.ServeHTTP(w, r)
	})
}

// ingestGate: mientras la ingesta no termina, los endpoints de datos responden 503 (RF-D-01).
func (s *Server) ingestGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.tracker.Busy() {
			writeError(w, http.StatusServiceUnavailable, CodeIngestInProgress, "La ingesta de datos está en curso", map[string]any{"ingest_status": s.tracker.Get().Status})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isMaxBytes indica si un error de lectura se debe al límite de 1 MB.
func isMaxBytes(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

func (s *Server) dbError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("DB_ERROR", slog.String("request_id", info(r).id), slog.String("error", err.Error()))
	writeError(w, http.StatusInternalServerError, CodeInternal, "Error interno", nil)
}
