package session

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
)

func encodeRefreshToken(selector uuid.UUID, validator []byte) string {
	return base64.RawURLEncoding.EncodeToString(selector[:]) + "." +
		base64.RawURLEncoding.EncodeToString(validator)
}

func parseRefreshToken(token string) (uuid.UUID, []byte, error) {
	parts := strings.Split(token, ".")

	if len(parts) != 2 {
		return uuid.Nil, nil, fmt.Errorf("%w: %s", serviceerr.InvalidInput, "token couldn't be decoded")
	}

	selectorBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("%w: %s", serviceerr.InvalidInput, "token couldn't be decoded")
	}
	selector, err := uuid.FromBytes(selectorBytes)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("%w: %s", serviceerr.InvalidInput, "token couldn't be decoded")
	}

	validator, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("%w: %s", serviceerr.InvalidInput, "token couldn't be decoded")
	}

	return selector, validator, nil
}
