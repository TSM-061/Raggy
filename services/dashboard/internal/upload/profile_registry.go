package upload

import (
	"fmt"
	"strings"
)

type ProfileRegistry struct {
	profiles map[string]struct{}
}

func NewProfileRegistry(initialProfiles []string) *ProfileRegistry {
	r := &ProfileRegistry{
		profiles: make(map[string]struct{}),
	}

	for _, profile := range initialProfiles {
		normalized, err := normalizeProfileKey(profile)
		if err != nil {
			panic(err)
		}

		r.profiles[normalized] = struct{}{}
	}

	return r
}

func normalizeProfileKey(profile string) (string, error) {
	normalized := strings.TrimSpace(profile)
	if normalized == "" {
		return "", fmt.Errorf("profile key is required")
	}

	return normalized, nil
}

func (r *ProfileRegistry) IsRegistered(profile string) bool {
	normalized, err := normalizeProfileKey(profile)
	if err != nil {
		return false
	}

	_, ok := r.profiles[normalized]

	return ok
}
