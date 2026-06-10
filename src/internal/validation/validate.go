package validation

import (
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Error holds a single field-level validation failure.
type Error struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Errors is a collection of validation errors that can be accumulated.
type Errors []Error

// Add appends a validation error.
func (ve *Errors) Add(field, message string) {
	*ve = append(*ve, Error{Field: field, Message: message})
}

// HasErrors returns true if any errors have been accumulated.
func (ve Errors) HasErrors() bool {
	return len(ve) > 0
}

// Error implements the error interface for use as a return type.
func (ve Errors) Error() string {
	if len(ve) == 0 {
		return "validation: no errors"
	}
	var b strings.Builder
	b.WriteString("validation: ")
	for i, e := range ve {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Field)
		b.WriteString(": ")
		b.WriteString(e.Message)
	}
	return b.String()
}

// --- Pre-defined validators ---

var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 _\-.'()]+$`)

// NotEmpty checks that a string is non-empty after trimming whitespace.
func NotEmpty(value, field, label string, errs *Errors) {
	if strings.TrimSpace(value) == "" {
		errs.Add(field, label+" is required")
	}
}

// MaxLength checks that a string's rune count does not exceed max.
func MaxLength(value string, max int, field, label string, errs *Errors) {
	if utf8.RuneCountInString(value) > max {
		errs.Add(field, label+" exceeds maximum length of "+strconv.Itoa(max))
	}
}

// Email validates a basic email format.
func Email(value, field string, errs *Errors) {
	if value == "" {
		return // use NotEmpty separately if required
	}
	_, err := mail.ParseAddress(value)
	if err != nil {
		errs.Add(field, "Invalid email address")
	}
}

// Name validates a user/display name (letters, numbers, spaces, basic punctuation).
func Name(value, field string, errs *Errors) {
	if value == "" {
		return
	}
	if !validName.MatchString(value) {
		errs.Add(field, "Name contains invalid characters")
	}
}

// AllowedStrings checks a string is one of a set of allowed values.
func AllowedStrings(value string, allowed []string, field, label string, errs *Errors) {
	if value == "" {
		return
	}
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	errs.Add(field, label+" must be one of: "+strings.Join(allowed, ", "))
}

// MinValue checks an integer is >= a minimum.
func MinValue(value, min int, field, label string, errs *Errors) {
	if value < min {
		errs.Add(field, label+" must be at least "+strconv.Itoa(min))
	}
}

// PositiveInt checks an integer is > 0.
func PositiveInt(value int, field, label string, errs *Errors) {
	if value <= 0 {
		errs.Add(field, label+" must be a positive integer")
	}
}
