package http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

const tokenTTL = 8 * time.Hour

type loginLimiter struct {
	mu  sync.Mutex
	ips map[string]*rate.Limiter
}

func newLoginLimiter() *loginLimiter { return &loginLimiter{ips: map[string]*rate.Limiter{}} }

// get devuelve el limitador de la IP: 10 intentos fallidos por minuto (RF-B-04, DEBERÍA).
func (l *loginLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.ips[ip]
	if !ok {
		lim = rate.NewLimiter(rate.Every(6*time.Second), 10)
		l.ips[ip] = lim
	}
	return lim
}

type claims struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

func requireJSON(w http.ResponseWriter, r *http.Request) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, CodeUnsupportedMediaType, "Content-Type debe ser application/json", map[string]any{"expected": "application/json"})
		return false
	}
	return true
}

// decodeJSON decodifica con DisallowUnknownFields y responde el error adecuado.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if isMaxBytes(err) {
			writeError(w, http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "Cuerpo demasiado grande", map[string]any{"max_bytes": maxBodyBytes})
			return false
		}
		if msg := err.Error(); strings.HasPrefix(msg, "json: unknown field ") {
			field := strings.Trim(strings.TrimPrefix(msg, "json: unknown field "), `"`)
			invalidBody(w, field, nil, "unknown field")
			return false
		}
		invalidBody(w, "body", nil, "invalid JSON")
		return false
	}
	return true
}

// login implementa POST /auth/login (RF-B-04).
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	lim := s.limiter.get(remoteIP(r))
	if lim.Tokens() < 1 {
		writeError(w, http.StatusTooManyRequests, CodeTooManyAttempts, "Demasiados intentos; espera un minuto", map[string]any{"retry_after_seconds": 60})
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	user, err := s.store.GetUser(r.Context(), body.Email)
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	hash := s.dummyHash
	if user != nil {
		hash = []byte(user.PasswordHash)
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(body.Password)) != nil || user == nil {
		lim.Allow()
		writeError(w, http.StatusUnauthorized, CodeInvalidCredentials, "Credenciales inválidas", nil)
		return
	}
	now := time.Now().UTC().Truncate(time.Second)
	exp := now.Add(tokenTTL)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{Name: user.Name, RegisteredClaims: jwt.RegisteredClaims{
		Subject: user.Email, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(exp)}})
	signed, err := tok.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		s.dbError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      signed,
		"expires_at": exp.Format(time.RFC3339),
		"user":       map[string]string{"email": user.Email, "name": user.Name},
	})
}

// auth (7): valida Bearer HS256 (RF-B-05). La causa va al log, no al body.
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reason := ""
		h := r.Header.Get("Authorization")
		var raw string
		switch {
		case h == "":
			reason = "missing"
		case !strings.HasPrefix(h, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(h, "Bearer ")) == "":
			reason = "malformed"
		default:
			raw = strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		}
		var sub string
		if reason == "" {
			reason, sub = s.verifyToken(raw)
		}
		if reason != "" {
			s.log.Warn("auth_failed", "request_id", info(r).id, "reason", reason, "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, CodeUnauthorized, "Token inválido o expirado", nil)
			return
		}
		info(r).user = sub
		next.ServeHTTP(w, r)
	})
}

func (s *Server) verifyToken(raw string) (reason, sub string) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return "malformed", ""
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "malformed", ""
	}
	var hdr struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(hb, &hdr) != nil {
		return "malformed", ""
	}
	if hdr.Alg != "HS256" {
		return "bad_alg", ""
	}
	var c claims
	_, err = jwt.ParseWithClaims(raw, &c, func(*jwt.Token) (any, error) { return []byte(s.cfg.JWTSecret), nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithLeeway(0))
	switch {
	case err == nil:
	case errors.Is(err, jwt.ErrTokenExpired):
		return "expired", ""
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return "bad_signature", ""
	default:
		return "malformed", ""
	}
	if c.Subject == "" {
		return "malformed", ""
	}
	return "", c.Subject
}
