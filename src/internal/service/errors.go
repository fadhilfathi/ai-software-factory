package service

import (
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
)

// Error represents an error from the service layer with an HTTP status code
// and structured body suitable for direct serialization.
type Error struct {
	Status  int
	Code    string
	Message string
	Details []validation.Error
}

func (e *Error) Error() string {
	return e.Message
}

func validationError(errs validation.Errors) *Error {
	details := make([]validation.Error, len(errs))
	copy(details, errs)
	return &Error{
		Status:  http.StatusBadRequest,
		Code:    "VALIDATION_ERROR",
		Message: "Validation failed",
		Details: details,
	}
}

func validationSingle(field, message string) *Error {
	return &Error{
		Status:  http.StatusBadRequest,
		Code:    "VALIDATION_ERROR",
		Message: message,
		Details: []validation.Error{{Field: field, Message: message}},
	}
}

func unauthorized(message string) *Error {
	return &Error{
		Status:  http.StatusUnauthorized,
		Code:    "UNAUTHORIZED",
		Message: message,
	}
}

func notFound(message string) *Error {
	return &Error{
		Status:  http.StatusNotFound,
		Code:    "NOT_FOUND",
		Message: message,
	}
}

func conflict(message string) *Error {
	return &Error{
		Status:  http.StatusConflict,
		Code:    "CONFLICT",
		Message: message,
	}
}

func internalError(message string) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_ERROR",
		Message: message,
	}
}
