// Package http expone la API (spec backend RF-B-03..RF-B-18).
package http

import (
	"encoding/json"
	"net/http"
)

// Códigos de error (RF-B-17).
const (
	CodeInvalidCredentials   = "INVALID_CREDENTIALS"
	CodeUnauthorized         = "UNAUTHORIZED"
	CodeTooManyAttempts      = "TOO_MANY_ATTEMPTS"
	CodeInvalidQuery         = "INVALID_QUERY"
	CodeInvalidBody          = "INVALID_BODY"
	CodeUnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"
	CodePayloadTooLarge      = "PAYLOAD_TOO_LARGE"
	CodeMeterNotFound        = "METER_NOT_FOUND"
	CodeAnomalyNotFound      = "ANOMALY_NOT_FOUND"
	CodeAnalysisNotFound     = "ANALYSIS_NOT_FOUND"
	CodeAnalysisInProgress   = "ANALYSIS_IN_PROGRESS"
	CodeIngestInProgress     = "INGEST_IN_PROGRESS"
	CodeRequestTimeout       = "REQUEST_TIMEOUT"
	CodeInternal             = "INTERNAL_ERROR"
	CodeNotFound             = "NOT_FOUND"
	CodeMethodNotAllowed     = "METHOD_NOT_ALLOWED"
)

const maxBodyBytes = 1 << 20

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details any    `json:"details"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string, details any) {
	var b errorBody
	b.Error.Code, b.Error.Message, b.Error.Details = code, message, details
	writeJSON(w, status, b)
}

func invalidQuery(w http.ResponseWriter, param string, allowed []string, reason string) {
	d := map[string]any{"param": param}
	if allowed != nil {
		d["allowed"] = allowed
	} else {
		d["reason"] = reason
	}
	writeError(w, http.StatusBadRequest, CodeInvalidQuery, "Parámetro de consulta inválido: "+param, d)
}

func invalidBody(w http.ResponseWriter, field string, allowed []string, reason string) {
	d := map[string]any{"field": field}
	if allowed != nil {
		d["allowed"] = allowed
	} else {
		d["reason"] = reason
	}
	writeError(w, http.StatusBadRequest, CodeInvalidBody, "Cuerpo de la petición inválido", d)
}
