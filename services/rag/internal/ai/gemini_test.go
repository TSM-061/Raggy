package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TSM-061/Raggy/rag/internal/rag"
	"google.golang.org/genai"
)

func TestGeminiStructuredGeneration(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		status       int
		valid        bool
	}{
		{"answer", `{"status":"answered","answer":"Supported [S1].","references":["S1"]}`, 200, true},
		{"abstain", `{"status":"insufficient_evidence","answer":"","references":[]}`, 200, true},
		{"malformed", `not json`, 200, false},
		{"extra fields", `{"status":"answered","answer":"x","references":[],"text":"invented source"}`, 200, false},
		{"provider failure", "", 400, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Contents []struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"contents"`
					GenerationConfig struct {
						ResponseMIMEType string         `json:"responseMimeType"`
						ResponseSchema   map[string]any `json:"responseSchema"`
					} `json:"generationConfig"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if request.GenerationConfig.ResponseMIMEType != "application/json" || request.GenerationConfig.ResponseSchema["properties"] == nil {
					t.Error("missing structured-output configuration")
				}
				if len(request.Contents) != 1 || len(request.Contents[0].Parts) != 1 || request.Contents[0].Parts[0].Text != "question and context" {
					t.Error("prompt changed")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if tc.status != 200 {
					_, _ = w.Write([]byte(`{"error":{"code":400,"message":"failed","status":"INVALID_ARGUMENT"}}`))
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"role": "model", "parts": []any{map[string]any{"text": tc.output}}}, "finishReason": "STOP"}}})
			}))
			defer server.Close()
			client, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: "test-key", Backend: genai.BackendGeminiAPI, HTTPOptions: genai.HTTPOptions{BaseURL: server.URL}})
			if err != nil {
				t.Fatal(err)
			}
			g := &GeminiClient{client: client, generationModel: "test-model"}
			result, err := g.GenerateContent(context.Background(), "question and context")
			if (err == nil) != tc.valid {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if !tc.valid && tc.status == 200 && !errors.Is(err, rag.ErrInvalidGeneration) {
				t.Fatalf("wrong format error: %v", err)
			}
		})
	}
}
