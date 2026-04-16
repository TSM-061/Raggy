package env

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const helperProcessEnv = "GO_HELPER_PROCESS"

func mapLookup(values map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func mustPanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic, got none")
		}
	}()

	fn()
}

func mustNotPanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("expected no panic, got %v", recovered)
		}
	}()

	fn()
}

func runNewAndCaptureExit(t *testing.T, testName string, envValue string, setEnv bool) (string, int) {
	t.Helper()

	if os.Getenv(helperProcessEnv) == "1" {
		values := map[string]string{}
		if setEnv {
			values["ENV"] = os.Getenv("HELPER_ENV_VALUE")
		}

		NewHelper(mapLookup(values))
		t.Fatal("expected process to exit")
	}

	cmd := exec.Command(os.Args[0], "-test.run=^"+testName+"$")
	cmd.Env = append(os.Environ(), helperProcessEnv+"=1")
	if setEnv {
		cmd.Env = append(cmd.Env, "HELPER_ENV_VALUE="+envValue)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit error, got %v", err)
	}

	return stderr.String(), exitErr.ExitCode()
}

func TestNew_ExitsWhenEnvMissing(t *testing.T) {
	stderr, exitCode := runNewAndCaptureExit(t, "TestNew_ExitsWhenEnvMissing", "", false)
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}

	if !strings.Contains(stderr, "FATAL: ENV variable required") {
		t.Fatalf("unexpected stderr output: %q", stderr)
	}
}

func TestNew_ExitsWhenEnvInvalid(t *testing.T) {
	stderr, exitCode := runNewAndCaptureExit(t, "TestNew_ExitsWhenEnvInvalid", "staging", true)
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}

	if !strings.Contains(stderr, "FATAL: ENV.Env must be") {
		t.Fatalf("unexpected stderr output: %q", stderr)
	}
}

func TestNew_SetsModeFlags(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		isDev  bool
		isProd bool
	}{
		{name: "dev short", env: "dev", isDev: true, isProd: false},
		{name: "dev long", env: "development", isDev: true, isProd: false},
		{name: "prod short", env: "prod", isDev: false, isProd: true},
		{name: "prod long", env: "production", isDev: false, isProd: true},
		{name: "case insensitive", env: "PRODUCTION", isDev: false, isProd: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHelper(mapLookup(map[string]string{"ENV": tt.env}))
			if h.IsDev != tt.isDev {
				t.Fatalf("expected IsDev=%t, got %t", tt.isDev, h.IsDev)
			}
			if h.IsProd != tt.isProd {
				t.Fatalf("expected IsProd=%t, got %t", tt.isProd, h.IsProd)
			}
		})
	}
}

func TestRequiredGetters_PanicWhenKeyMissing(t *testing.T) {
	h := NewHelper(mapLookup(map[string]string{"ENV": "dev"}))

	mustPanic(t, func() {
		h.GetStringRequired("MISSING_STRING")
	})

	mustPanic(t, func() {
		h.GetIntRequired("MISSING_INT")
	})

	mustPanic(t, func() {
		h.GetBoolRequired("MISSING_BOOL")
	})
}

func TestFallbackGetters_ReturnFallbackWhenKeyMissing(t *testing.T) {
	h := NewHelper(mapLookup(map[string]string{"ENV": "dev"}))

	mustNotPanic(t, func() {
		got := h.GetString("MISSING_STRING", "fallback-value")
		if got != "fallback-value" {
			t.Fatalf("expected fallback string, got %q", got)
		}
	})

	mustNotPanic(t, func() {
		got := h.GetInt("MISSING_INT", 42)
		if got != 42 {
			t.Fatalf("expected fallback int 42, got %d", got)
		}
	})

	mustNotPanic(t, func() {
		got := h.GetBool("MISSING_BOOL", true)
		if got != true {
			t.Fatalf("expected fallback bool true, got %t", got)
		}
	})
}

func TestGetters_ReturnActualValuesWhenPresent(t *testing.T) {
	h := NewHelper(mapLookup(map[string]string{
		"ENV":              "dev",
		"STRING_VALUE":     "configured-value",
		"INT_VALUE":        "123",
		"BOOL_VALUE_TRUE":  "true",
		"BOOL_VALUE_FALSE": "false",
	}))

	mustNotPanic(t, func() {
		got := h.GetString("STRING_VALUE", "fallback-value")
		if got != "configured-value" {
			t.Fatalf("expected configured string, got %q", got)
		}
	})

	mustNotPanic(t, func() {
		got := h.GetStringRequired("STRING_VALUE")
		if got != "configured-value" {
			t.Fatalf("expected configured required string, got %q", got)
		}
	})

	mustNotPanic(t, func() {
		got := h.GetInt("INT_VALUE", 42)
		if got != 123 {
			t.Fatalf("expected configured int 123, got %d", got)
		}
	})

	mustNotPanic(t, func() {
		got := h.GetIntRequired("INT_VALUE")
		if got != 123 {
			t.Fatalf("expected configured required int 123, got %d", got)
		}
	})

	mustNotPanic(t, func() {
		got := h.GetBool("BOOL_VALUE_TRUE", false)
		if got != true {
			t.Fatalf("expected configured bool true, got %t", got)
		}
	})

	mustNotPanic(t, func() {
		got := h.GetBoolRequired("BOOL_VALUE_FALSE")
		if got != false {
			t.Fatalf("expected configured required bool false, got %t", got)
		}
	})
}

func TestIntGetters_PanicOnInvalidValue(t *testing.T) {
	h := NewHelper(mapLookup(map[string]string{
		"ENV":       "dev",
		"INT_VALUE": "abc",
	}))

	mustPanic(t, func() {
		h.GetInt("INT_VALUE", 42)
	})

	mustPanic(t, func() {
		h.GetIntRequired("INT_VALUE")
	})
}

func TestBoolGetters_PanicOnInvalidValue(t *testing.T) {
	h := NewHelper(mapLookup(map[string]string{
		"ENV":        "dev",
		"BOOL_VALUE": "maybe",
	}))

	mustPanic(t, func() {
		h.GetBool("BOOL_VALUE", true)
	})

	mustPanic(t, func() {
		h.GetBoolRequired("BOOL_VALUE")
	})
}
