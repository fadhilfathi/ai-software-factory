package handler

import (
	"errors"
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
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

func writeJSON(c *gin.Context, status int, v interface{}) {
	c.JSON(status, v)
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
	})
}

func writeErrorWithDetails(c *gin.Context, status int, code, message string, details []ErrorDetail) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message, Details: details},
	})
}

func writeServiceError(c *gin.Context, err *service.Error) {
	if err == nil {
		return
	}
	details := make([]ErrorDetail, len(err.Details))
	for i, d := range err.Details {
		details[i] = ErrorDetail{Field: d.Field, Message: d.Message}
	}
	if len(details) > 0 {
		writeErrorWithDetails(c, err.Status, err.Code, err.Message, details)
	} else {
		writeError(c, err.Status, err.Code, err.Message)
	}
}

// isMaxBytesError reports whether err originated from an
// http.MaxBytesReader trip. The Go 1.20+ stdlib returns the
// concrete *http.MaxBytesError type from a Read past the cap;
// json.Decoder surfaces that error verbatim via Unmarshal, and
// Gin's c.ShouldBindJSON forwards it unchanged. We also fall
// back to errors.As in case future stdlib versions wrap the
// error.
//
// Added in TASK-424 (F-023). The deliverable handler uses
// this to map an oversize request body to 413
// PAYLOAD_TOO_LARGE instead of the generic 400 INVALID_JSON.
func isMaxBytesError(err error) bool {
	if err == nil {
		return false
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}
	return false
}
