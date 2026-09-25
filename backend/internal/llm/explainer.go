package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	openai "github.com/sashabaranov/go-openai"

	"github.com/jsotelo/energy-platform/backend/internal/engine"
)

// DefaultModel es el id de DeepSeek-V4.1-Flash, el modelo fijado para el explicador.
// `deepseek-chat` fue retirado por DeepSeek en 2026 (verificado contra GET /models el 2026-09-25).
const DefaultModel = "deepseek-flash"

// Config del cliente DeepSeek (maestro §9).
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
}

// Causas de fallback (RF-E-17).
const (
	CauseNoAPIKey      = "no_api_key"
	CauseTimeout       = "timeout"
	CauseHTTPError     = "http_error"
	CauseInvalidJSON   = "invalid_json"
	CauseSchemaInvalid = "schema_invalid"
)

// Explainer usa DeepSeek y cae a plantillas ante cualquier fallo. Se crea uno por análisis:
// un 401/403 deshabilita el cliente para el resto de ese análisis (CB-E-16).
type Explainer struct {
	cfg      Config
	client   *openai.Client
	log      *slog.Logger
	disabled atomic.Bool
	// retryWait es la espera antes de reintentar tras 429/5xx (2 s; configurable en tests).
	retryWait time.Duration
}

// NewExplainer construye el Explainer. Con APIKey vacía no instancia el cliente (modo plantilla).
func NewExplainer(cfg Config, log *slog.Logger) *Explainer {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}
	e := &Explainer{cfg: cfg, log: log, retryWait: 2 * time.Second}
	if cfg.APIKey != "" {
		oc := openai.DefaultConfig(cfg.APIKey)
		oc.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
		e.client = openai.NewClientWithConfig(oc)
	}
	return e
}

type llmAnswer struct {
	Reason            string   `json:"reason"`
	RecommendedAction string   `json:"recommended_action"`
	ExplanationPoints []string `json:"explanation_points"`
}

type callError struct {
	cause string
	retry bool
	err   error
}

func (c *callError) Error() string { return c.cause + ": " + c.err.Error() }

// Explain implementa engine.Explainer.
func (e *Explainer) Explain(ctx context.Context, in engine.ExplainInput) engine.ExplainOutput {
	if e.client == nil {
		e.logCall(in.MeterID, 0, "fallback", CauseNoAPIKey, 0, 0)
		return Template(in)
	}
	if e.disabled.Load() {
		e.logCall(in.MeterID, 0, "fallback", CauseHTTPError, 0, 0)
		return Template(in)
	}
	body, err := BuildPayload(in)
	if err != nil {
		e.logCall(in.MeterID, 0, "fallback", CauseSchemaInvalid, 0, 0)
		return Template(in)
	}
	var last *callError
	for attempt := 0; attempt < 2; attempt++ {
		ans, cerr := e.call(ctx, in.MeterID, string(body))
		if cerr == nil {
			return engine.ExplainOutput{Reason: ans.Reason, RecommendedAction: ans.RecommendedAction, Points: ans.ExplanationPoints, Source: "llm"}
		}
		last = cerr
		if !cerr.retry || ctx.Err() != nil {
			break
		}
		if cerr.cause == CauseHTTPError {
			select {
			case <-time.After(e.retryWait):
			case <-ctx.Done():
			}
		}
	}
	e.log.Warn("LLM_FALLBACK", "meter_id", in.MeterID, "cause", last.cause, "error", last.err.Error())
	e.logCall(in.MeterID, 0, "fallback", last.cause, 0, 0)
	return Template(in)
}

func (e *Explainer) call(parent context.Context, meterID, userMsg string) (*llmAnswer, *callError) {
	ctx, cancel := context.WithTimeout(parent, e.cfg.Timeout)
	defer cancel()
	start := time.Now()
	req := openai.ChatCompletionRequest{
		Model:     e.cfg.Model,
		MaxTokens: 2000,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: SystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userMsg},
		},
	}
	// Modelos "reasoner" heredados no soportan response_format ni temperature (spec del motor §8.5).
	// deepseek-flash y deepseek-v4-pro sí los soportan; sus tokens de razonamiento cuentan en MaxTokens.
	if !strings.Contains(e.cfg.Model, "reasoner") {
		req.Temperature = 0
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	}
	resp, err := e.client.CreateChatCompletion(ctx, req)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			e.logCall(meterID, ms, "timeout", CauseTimeout, 0, 0)
			return nil, &callError{cause: CauseTimeout, retry: true, err: err}
		}
		var apiErr *openai.APIError
		var reqErr *openai.RequestError
		status := 0
		if errors.As(err, &apiErr) {
			status = apiErr.HTTPStatusCode
		} else if errors.As(err, &reqErr) {
			status = reqErr.HTTPStatusCode
		}
		e.logCall(meterID, ms, "error", CauseHTTPError, 0, 0)
		if status == 401 || status == 403 {
			e.disabled.Store(true)
			return nil, &callError{cause: CauseHTTPError, retry: false, err: err}
		}
		return nil, &callError{cause: CauseHTTPError, retry: status == 429 || status >= 500 || status == 0, err: err}
	}
	if len(resp.Choices) == 0 {
		e.logCall(meterID, ms, "invalid_json", CauseSchemaInvalid, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		return nil, &callError{cause: CauseSchemaInvalid, retry: true, err: errors.New("empty choices")}
	}
	ch := resp.Choices[0]
	if ch.FinishReason == openai.FinishReasonLength || ch.FinishReason == openai.FinishReasonContentFilter {
		e.logCall(meterID, ms, "invalid_json", CauseSchemaInvalid, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		return nil, &callError{cause: CauseSchemaInvalid, retry: true, err: fmt.Errorf("finish_reason=%s", ch.FinishReason)}
	}
	ans, cause, verr := ValidateAnswer(ch.Message.Content)
	if verr != nil {
		e.logCall(meterID, ms, "invalid_json", cause, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		return nil, &callError{cause: cause, retry: true, err: verr}
	}
	e.logCall(meterID, ms, "ok", "", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	return ans, nil
}

func (e *Explainer) logCall(meterID string, ms int64, outcome, cause string, pt, ct int) {
	attrs := []any{"meter_id", meterID, "model", e.cfg.Model, "duration_ms", ms, "outcome", outcome, "prompt_tokens", pt, "completion_tokens", ct}
	if cause != "" {
		attrs = append(attrs, "cause", cause)
	}
	e.log.Info("llm_call", attrs...)
}

// ValidateAnswer aplica la validación de la sección 8.3 del spec del motor.
func ValidateAnswer(content string) (*llmAnswer, string, error) {
	s := strings.TrimSpace(content)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	var ans llmAnswer
	if err := json.Unmarshal([]byte(s), &ans); err != nil {
		return nil, CauseInvalidJSON, err
	}
	n := func(x string) int { return utf8.RuneCountInString(x) }
	switch {
	case n(ans.Reason) < 20 || n(ans.Reason) > 600:
		return nil, CauseSchemaInvalid, errors.New("reason length")
	case sentences(ans.Reason) < 1 || sentences(ans.Reason) > 3:
		return nil, CauseSchemaInvalid, errors.New("reason sentences")
	case n(ans.RecommendedAction) < 10 || n(ans.RecommendedAction) > 200:
		return nil, CauseSchemaInvalid, errors.New("recommended_action length")
	case !strings.HasSuffix(strings.TrimSpace(ans.RecommendedAction), ".") || strings.ContainsAny(ans.RecommendedAction, "\r\n"):
		return nil, CauseSchemaInvalid, errors.New("recommended_action format")
	case len(ans.ExplanationPoints) < 2 || len(ans.ExplanationPoints) > 6:
		return nil, CauseSchemaInvalid, errors.New("explanation_points count")
	}
	for _, p := range ans.ExplanationPoints {
		if n(p) < 10 || n(p) > 300 {
			return nil, CauseSchemaInvalid, errors.New("explanation_point length")
		}
	}
	ans.RecommendedAction = strings.TrimSpace(ans.RecommendedAction)
	return &ans, "", nil
}

// sentences cuenta terminadores . ! ? seguidos de espacio o fin de texto.
func sentences(s string) int {
	s = strings.TrimSpace(s)
	count := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' || c == '!' || c == '?' {
			if i == len(s)-1 || s[i+1] == ' ' || s[i+1] == '\n' {
				count++
			}
		}
	}
	return count
}
