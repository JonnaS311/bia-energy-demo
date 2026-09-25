// Package config lee y valida las variables de entorno (maestro §9).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config es la configuración del proceso.
type Config struct {
	DatabaseURL      string
	Port             string
	DataDir          string
	JWTSecret        string
	DemoUserEmail    string
	DemoUserPassword string
	DeepSeekAPIKey   string
	DeepSeekModel    string
	DeepSeekBaseURL  string
	LLMTimeout       time.Duration
	CORSOrigins      []string
	CORSDefaulted    bool
	LogLevel         string
	BaselineDays     int
	BaselineInvalid  bool
}

// InvalidError es un error de configuración con el campo afectado (CONFIG_INVALID).
type InvalidError struct{ Field string }

func (e *InvalidError) Error() string { return fmt.Sprintf("CONFIG_INVALID: %s", e.Field) }

func get(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// Load lee la configuración. JWT_SECRET y DEMO_USER_PASSWORD no pueden ser vacíos.
func Load() (Config, error) {
	c := Config{
		DatabaseURL:      get("DATABASE_URL", "postgres://energy:energy@localhost:5432/energy?sslmode=disable"),
		Port:             get("PORT", "8080"),
		DataDir:          get("DATA_DIR", "/app/data"),
		DemoUserEmail:    get("DEMO_USER_EMAIL", "analista@energy.local"),
		DeepSeekAPIKey:   os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekModel:    get("DEEPSEEK_MODEL", "deepseek-flash"),
		DeepSeekBaseURL:  get("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		LogLevel:         strings.ToLower(get("LOG_LEVEL", "info")),
		BaselineDays:     7,
	}
	// JWT_SECRET y DEMO_USER_PASSWORD: el default solo aplica si la variable no existe; vacía explícita es error.
	c.JWTSecret = envOrDefaultStrict("JWT_SECRET", "change-me-in-prod")
	c.DemoUserPassword = envOrDefaultStrict("DEMO_USER_PASSWORD", "Demo1234!")
	if c.JWTSecret == "" {
		return c, &InvalidError{Field: "JWT_SECRET"}
	}
	if c.DemoUserPassword == "" {
		return c, &InvalidError{Field: "DEMO_USER_PASSWORD"}
	}
	secs, err := strconv.Atoi(get("LLM_TIMEOUT_SECONDS", "20"))
	if err != nil || secs <= 0 {
		secs = 20
	}
	c.LLMTimeout = time.Duration(secs) * time.Second
	if v := os.Getenv("BASELINE_DAYS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 2 {
			c.BaselineInvalid = true // CB-E-22: se usa 7 y se registra CONFIG_INVALID
		} else {
			c.BaselineDays = n
		}
	}
	origins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if origins == "" {
		c.CORSOrigins = []string{"http://localhost:5173"}
		c.CORSDefaulted = true
	} else {
		for _, o := range strings.Split(origins, ",") {
			if o = strings.TrimSpace(o); o != "" {
				c.CORSOrigins = append(c.CORSOrigins, o)
			}
		}
	}
	return c, nil
}

func envOrDefaultStrict(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
