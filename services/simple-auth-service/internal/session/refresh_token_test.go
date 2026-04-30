package session

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
)

func TestEncodeRefreshToken_RoundTrip(t *testing.T) {
	selector := uuid.New()
	validator := []byte("validator-bytes")

	token := encodeRefreshToken(selector, validator)

	parsedSelector, parsedValidator, err := parseRefreshToken(token)
	if err != nil {
		t.Fatalf("parseRefreshToken() unexpected error: %v", err)
	}

	if parsedSelector != selector {
		t.Fatalf("parseRefreshToken() selector: got %v, want %v", parsedSelector, selector)
	}

	if string(parsedValidator) != string(validator) {
		t.Fatalf("parseRefreshToken() validator: got %q, want %q", parsedValidator, validator)
	}
}

func TestParseRefreshToken_InvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
				name:  "MissingSeparator",
			token: "not-a-token",
		},
		{
				name:  "TooManyParts",
			token: "a.b.c",
		},
		{
				name:  "InvalidSelectorBase64",
			token: "!!!.aGVsbG8",
		},
		{
				name:  "SelectorIsNotUUIDBytes",
			token: base64.RawURLEncoding.EncodeToString([]byte("short")) + ".aGVsbG8",
		},
		{
				name:  "InvalidValidatorBase64",
			token: base64.RawURLEncoding.EncodeToString(uuid.Nil[:]) + ".!!!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, validator, err := parseRefreshToken(tt.token)
			if err == nil {
				t.Fatalf("parseRefreshToken() expected error for token %q, got nil", tt.token)
			}

			if !errors.Is(err, serviceerr.InvalidInput) {
				t.Fatalf("parseRefreshToken() error: got %v, want wrapped %v", err, serviceerr.InvalidInput)
			}

			if selector != uuid.Nil {
				t.Fatalf("parseRefreshToken() selector on error: got %v, want %v", selector, uuid.Nil)
			}

			if validator != nil {
				t.Fatalf("parseRefreshToken() validator on error: got %v, want nil", validator)
			}
		})
	}
}
