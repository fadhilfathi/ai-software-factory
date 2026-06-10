package handler

import (
	"encoding/json"
	"net/http"

	"github.com/example/project/internal/service"
)

// APIResponse wraps a successful payload.
type APIResponse struct {
	Data interface{} `json:"data,omitempty"`
}

// ErrorResponse matches the spec's error format.
type ErrorResponse struct {
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"request_id,omitempty"`
}

// ErrorBody holds structured error details.
type ErrorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorDetail represents a single field-level validation error.
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Pagination matches the spec's pagination envelope.
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// PaginatedResponse wraps a list endpoint.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
	})
}

func writeErrorWithDetails(w http.ResponseWriter, status int, code, message string, details []ErrorDetail) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message, Details: details},
	})
}

func writeServiceError(w http.ResponseWriter, err *service.Error) {
	if err == nil {
		return
	}
	details := make([]ErrorDetail, len(err.Details))
	for i, d := range err.Details {
		details[i] = ErrorDetail{Field: d.Field, Message: d.Message}
	}
	if len(details) > 0 {
		writeErrorWithDetails(w, err.Status, err.Code, err.Message, details)
	} else {
		writeError(w, err.Status, err.Code, err.Message)
	}
}
