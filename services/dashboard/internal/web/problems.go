package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// ProblemDetails represents an RFC 7807 problem details response.
type ProblemDetails struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"errors,omitempty"`
}

// FieldError describes a validation failure on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// WriteProblem writes a ProblemDetails response as JSON.
func WriteProblem(w http.ResponseWriter, p ProblemDetails) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p) //nolint:errcheck
}

// ValidationProblem converts a validator.ValidationErrors into a 422 ProblemDetails
// response and writes it. Returns false if err is not a validation error.
func ValidationProblem(w http.ResponseWriter, err error) bool {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return false
	}

	fieldErrors := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		fieldErrors = append(fieldErrors, FieldError{
			Field:   fe.Field(),
			Message: fe.Tag(),
		})
	}

	WriteProblem(w, ProblemDetails{
		Type:   "https://problems.raggy.dev/validation-error",
		Title:  "Validation Error",
		Status: http.StatusBadRequest,
		Detail: "One or more fields failed validation.",
		Errors: fieldErrors,
	})
	return true
}
