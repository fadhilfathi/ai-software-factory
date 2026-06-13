package service

import (
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
)

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

// forbidden returns a 403 *Error. Reserved for role-mismatch
// rejections (F-021 / TASK-425). For cross-tenant misses use
// crossTenantBlocked() instead, so the existence of the resource
// in another project is not leaked.
func forbidden(message string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: message}
}

// crossTenantBlocked returns a 404 *Error. Used when the caller
// is authenticated to a different project than the resource. Per
// the Sprint 5 design (Security-01, Lead-accepted 2026-06-13),
// the miss is reported as 404 with code CROSS_TENANT_BLOCKED so
// the existence of the resource in another project is not leaked.
func crossTenantBlocked() *Error {
	return &Error{
		Status:  http.StatusNotFound,
		Code:    "CROSS_TENANT_BLOCKED",
		Message: "resource not found in caller's project",
	}
}

// missingProjectHeader returns a 400 *Error. Used when the
// X-Project-ID header (or path-implied project) is absent on a
// route that requires project scoping.
func missingProjectHeader() *Error {
	return &Error{
		Status:  http.StatusBadRequest,
		Code:    "MISSING_PROJECT_HEADER",
		Message: "X-Project-ID header is required for this request",
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

func unprocessableEntity(code, message string) *Error {
	return &Error{
		Status:  http.StatusUnprocessableEntity,
		Code:    code,
		Message: message,
	}
}

// capabilityMismatch returns a 409 with code CAPABILITY_MISMATCH per
// api-spec.md §3.1. The missing slice carries the capability names the
// agent does not hold, for client diagnostics.
//
// Surfaced by CapabilityService.ValidateAgentHasCapabilities (TASK-403)
// and translated from the validation-seam call in AssignmentService.
func capabilityMismatch(missing []string) *Error {
	return &Error{
		Status:  http.StatusConflict,
		Code:    "CAPABILITY_MISMATCH",
		Message: "agent does not hold all required capabilities",
		Details: []validation.Error{{
			Field:   "required_capabilities",
			Message: "missing capabilities: " + joinStrings(missing, ", "),
		}},
	}
}

// payloadTooLarge returns a 413 with code PAYLOAD_TOO_LARGE. Used by
// the deliverable service (TASK-424 / F-023) to cap the markdown
// content size at MaxDeliverableContentBytes (1 MiB by default) and
// by the handler when http.MaxBytesReader trips on the raw request
// body. The Details field carries the byte limit so the client can
// right-size its request.
func payloadTooLarge(message string, maxBytes int64) *Error {
	return &Error{
		Status:  http.StatusRequestEntityTooLarge,
		Code:    "PAYLOAD_TOO_LARGE",
		Message: message,
		Details: []validation.Error{{
			Field:   "content",
			Message: "exceeds maximum allowed size of " + formatBytes(maxBytes) + " bytes",
		}},
	}
}

// formatBytes turns an int64 byte count into a human-readable
// string (e.g. 1048576 -> "1.00 MiB"). Kept here so errors.go
// stays the single source of truth for error-shape helpers.
func formatBytes(n int64) string {
	const (
		KiB = 1024
		MiB = 1024 * KiB
	)
	switch {
	case n >= MiB && n%MiB == 0:
		// integer MiB
		return intToString(n/MiB) + ".00 MiB"
	case n >= MiB:
		// fractional MiB
		whole := n / MiB
		frac := (n % MiB) * 100 / MiB
		return intToString(whole) + "." + padTwo(frac) + " MiB"
	case n >= KiB && n%KiB == 0:
		return intToString(n/KiB) + " KiB"
	case n >= KiB:
		whole := n / KiB
		frac := (n % KiB) * 100 / KiB
		return intToString(whole) + "." + padTwo(frac) + " KiB"
	default:
		return intToString(n) + " B"
	}
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func padTwo(n int64) string {
	if n < 10 {
		return "0" + intToString(n)
	}
	return intToString(n)
}

// joinStrings is a small helper to keep errors.go free of the strings
// package import (the rest of this file prefers validation.Error shapes).
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += sep + p
	}
	return out
}
