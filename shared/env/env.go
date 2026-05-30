package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type LookupFunc func(key string) (string, bool)

type Helper struct {
	lookup LookupFunc
	IsDev  bool
	IsProd bool
}

func NewHelper(lookup LookupFunc) *Helper {
	raw, ok := lookup("ENV")

	mode := "production"
	if ok {
		mode = strings.ToLower(raw)
	}

	isDev := mode == "dev" || mode == "development"
	isProd := mode == "prod" || mode == "production"

	if !isProd && !isDev {
		fmt.Fprintf(os.Stderr, "FATAL: ENV.Env must be: [dev]elopment/[prod]uction, got '%s'", mode)
		os.Exit(1)
	}

	return &Helper{
		lookup: lookup,
		IsDev:  isDev,
		IsProd: isProd,
	}
}

func (h *Helper) GetString(key string, fallback string) string {
	val, ok := h.lookup(key)

	if !ok {
		return fallback
	}

	return val
}

func (h *Helper) GetStringRequired(key string) string {
	val, ok := h.lookup(key)

	if !ok {
		panic(fmt.Sprintf("FATAL: %s is required", key))
	}

	return val
}

func (h *Helper) GetInt(key string, fallback int) int {
	s := h.GetString(key, strconv.Itoa(fallback))

	i, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("FATAL: %s must be an integer, got %q", key, s))
	}

	return i
}

func (h *Helper) GetIntRequired(key string) int {
	s := h.GetStringRequired(key)

	i, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("FATAL: %s must be an integer, got %q", key, s))
	}

	return i
}

func (h *Helper) GetBool(key string, fallback bool) bool {
	s := h.GetString(key, strconv.FormatBool(fallback))

	b, err := strconv.ParseBool(s)
	if err != nil {
		panic(fmt.Sprintf("FATAL: %s must be a boolean, got %q", key, s))
	}

	return b
}

func (h *Helper) GetBoolRequired(key string) bool {
	s := h.GetStringRequired(key)

	b, err := strconv.ParseBool(s)
	if err != nil {
		panic(fmt.Sprintf("FATAL: %s must be a boolean, got %q", key, s))
	}

	return b
}

func (h *Helper) GetDuration(key string, fallback string) time.Duration {
	s := h.GetString(key, fallback)

	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Sprintf("FATAL: %s must be a duration, got %q", key, s))
	}

	return d
}

func (h *Helper) GetDurationRequired(key string) time.Duration {
	s := h.GetStringRequired(key)

	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Sprintf("FATAL: %s must be a duration, got %q", key, s))
	}

	return d
}

func (h *Helper) GetStringSlice(key string, fallback []string) []string {
	s := h.GetString(key, "")

	if s == "" {
		return fallback
	}

	return strings.Split(s, ",")
}

func (h *Helper) GetStringSliceRequired(key string) []string {
	s := h.GetStringRequired(key)
	return strings.Split(s, ",")
}
