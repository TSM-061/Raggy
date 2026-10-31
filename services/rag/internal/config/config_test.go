package config

import (
	"os"
	"testing"
	"time"
)

func TestAIProviderConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, provider, key string
		unset, valid        bool
	}{
		{"fake needs no key", "fake", "", false, true},
		{"Gemini requires key", "gemini", "", false, false},
		{"Gemini configured", "gemini", "test-key", false, true},
		{"default is Gemini", "", "test-key", true, true},
		{"default requires key", "", "", true, false},
		{"unknown provider", "invalid", "test-key", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ACCESS_TOKEN_PUBLIC_KEY", "0qw/WOYvsmc3TJKKebk3vawfmo8aKGM6lpFchb6XviI=")
			t.Setenv("KAFKA_SEED_BROKERS", "localhost:9092")
			t.Setenv("TELEMETRY_COLLECTOR_ENDPOINT", "localhost:4317")
			t.Setenv("AI_PROVIDER", tc.provider)
			t.Setenv("FAKE_CHUNK_DELAY", "1s")
			if tc.unset {
				_ = os.Unsetenv("AI_PROVIDER")
			}
			t.Setenv("GEMINI_API_KEY", tc.key)
			cfg, err := LoadConfig()
			if (err == nil) != tc.valid {
				t.Fatalf("cfg=%+v err=%v", cfg, err)
			}
			if tc.unset && tc.valid && cfg.AIProvider != "gemini" {
				t.Fatal("live provider default changed")
			}
		})
	}
}

func TestFakeChunkDelayConfiguration(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_PUBLIC_KEY", "0qw/WOYvsmc3TJKKebk3vawfmo8aKGM6lpFchb6XviI=")
	t.Setenv("KAFKA_SEED_BROKERS", "localhost:9092")
	t.Setenv("TELEMETRY_COLLECTOR_ENDPOINT", "localhost:4317")
	t.Setenv("AI_PROVIDER", "fake")
	t.Setenv("GEMINI_API_KEY", "")
	for _, tc := range []struct {
		value    string
		duration time.Duration
		valid    bool
	}{{"1s", time.Second, true}, {"250ms", 250 * time.Millisecond, true}, {"0s", 0, true}, {"-1s", 0, false}, {"invalid", 0, false}} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("FAKE_CHUNK_DELAY", tc.value)
			cfg, err := LoadConfig()
			if (err == nil) != tc.valid {
				t.Fatalf("cfg=%+v err=%v", cfg, err)
			}
			if tc.valid && cfg.FakeChunkDelay != tc.duration {
				t.Fatalf("delay=%v", cfg.FakeChunkDelay)
			}
		})
	}
}
