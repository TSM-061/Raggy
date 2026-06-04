package problem

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/go-playground/validator/v10"
)

type ValidationDetails struct {
	Details
	Errors []FieldError `json:"errors"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func WriteValidationDetails(ctx context.Context, w http.ResponseWriter, err error) {
	log := logger.FromContext(ctx)

	var verrs validator.ValidationErrors

	if !errors.As(err, &verrs) {
		log.ErrorContext(ctx, "failed to validate struct", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fieldErrors := make([]FieldError, 0, len(verrs))
	for _, fe := range verrs {
		fieldErrors = append(fieldErrors, FieldError{
			Field:   fe.Field(),
			Message: fe.Tag(),
		})
	}

	problem := ValidationDetails{
		Details: Details{
			Type:   "https://datatracker.ietf.org/doc/html/rfc7807",
			Title:  "Validation Error",
			Status: http.StatusBadRequest,
			Detail: "One or more fields failed validation.",
		},
		Errors: fieldErrors,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		log.ErrorContext(
			ctx,
			"failed to encode problem details",
			slog.Any("error", err),
		)
	}
}
