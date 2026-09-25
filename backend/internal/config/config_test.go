package config

import (
	"errors"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"JWT_SECRET", "DEMO_USER_PASSWORD", "CORS_ALLOWED_ORIGINS", "LLM_TIMEOUT_SECONDS", "BASELINE_DAYS", "DEEPSEEK_MODEL"} {
		t.Setenv(k, "")
	}
	// t.Setenv con "" deja la variable definida y vacía: JWT_SECRET vacío es error (RF-B-02).
	_, err := Load()
	var ie *InvalidError
	if !errors.As(err, &ie) || ie.Field != "JWT_SECRET" {
		t.Fatalf("expected JWT_SECRET error, got %v", err)
	}
	t.Setenv("JWT_SECRET", "s")
	if _, err := Load(); !errors.As(err, &ie) || ie.Field != "DEMO_USER_PASSWORD" {
		t.Fatalf("expected DEMO_USER_PASSWORD error, got %v", err)
	}
	t.Setenv("DEMO_USER_PASSWORD", "p")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.CORSDefaulted || c.CORSOrigins[0] != "http://localhost:5173" || c.LLMTimeout != 20*time.Second || c.DeepSeekModel != "deepseek-flash" || c.BaselineDays != 7 {
		t.Fatalf("%+v", c)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("DEMO_USER_PASSWORD", "p")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a.test, http://b.test")
	t.Setenv("LLM_TIMEOUT_SECONDS", "5")
	t.Setenv("BASELINE_DAYS", "1")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.CORSOrigins) != 2 || c.CORSOrigins[1] != "http://b.test" || c.LLMTimeout != 5*time.Second || !c.BaselineInvalid || c.BaselineDays != 7 {
		t.Fatalf("%+v", c)
	}
}
