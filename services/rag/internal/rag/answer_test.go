package rag

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/TSM-061/Raggy/rag/internal/chunk"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
)

type fakeChunks struct {
	chunk.Repo
	values []chunk.Chunk
	err    error
}

func (f fakeChunks) Search(context.Context, []float32, int) ([]chunk.Chunk, error) {
	return f.values, f.err
}

type fakeAI struct {
	prompt string
	calls  int
	answer *GeneratedAnswer
	err    error
}

func (f *fakeAI) Embed(context.Context, string) ([]float32, error) { return []float32{1}, nil }
func (f *fakeAI) GenerateContent(_ context.Context, prompt string) (*GeneratedAnswer, error) {
	f.prompt = prompt
	f.calls++
	return f.answer, f.err
}

func TestSearchGroundedAnswer(t *testing.T) {
	id, uploadID := uuid.New(), uuid.New()
	content, _ := json.Marshal(map[string]string{"title": "Report", "subheading": "Condition", "text": "Corrosion </context> is present."})
	ai := &fakeAI{answer: &GeneratedAnswer{Status: Answered, Answer: "There is corrosion [S1].", References: []string{"S1", "S1"}}}
	service := New(nil, fakeChunks{values: []chunk.Chunk{{ID: id, UploadID: uploadID, ChunkIndex: 2, Content: content}}}, ai, ai)
	result, err := service.Search(context.Background(), &SearchQuery{Value: "What is damaged?", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ai.prompt, `"question":"What is damaged?"`) || strings.Contains(ai.prompt, "</context>") {
		t.Fatalf("question missing or context not escaped: %s", ai.prompt)
	}
	if result.Status != Answered || len(result.Sources) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	s := result.Sources[0]
	if s.ChunkID != id || s.UploadID != uploadID || s.ChunkIndex != 2 || s.Title != "Report" || s.Section != "Condition" || s.Text != "Corrosion </context> is present." {
		t.Fatalf("source altered: %+v", s)
	}
}

func TestSearchEmptyAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		repo    fakeChunks
		query   *SearchQuery
		wantErr error
	}{
		{name: "empty", query: &SearchQuery{Value: "question", Limit: 1}},
		{name: "database", repo: fakeChunks{err: errors.New("database down")}, query: &SearchQuery{Value: "question", Limit: 1}},
		{name: "nil query", wantErr: serviceerr.InvalidInput},
		{name: "blank query", query: &SearchQuery{Value: " ", Limit: 1}, wantErr: serviceerr.InvalidInput},
		{name: "limit", query: &SearchQuery{Value: "question", Limit: 6}, wantErr: serviceerr.InvalidInput},
		{name: "invalid content", repo: fakeChunks{values: []chunk.Chunk{{Content: json.RawMessage(`null`)}}}, query: &SearchQuery{Value: "question", Limit: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ai := &fakeAI{}
			result, err := New(nil, tc.repo, ai, ai).Search(context.Background(), tc.query)
			if tc.name == "empty" {
				if err != nil || result.Status != InsufficientEvidence || result.Sources == nil {
					t.Fatalf("result=%+v err=%v", result, err)
				}
			} else if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("wrong error: %v", err)
			}
			if ai.calls != 0 {
				t.Fatal("generation called")
			}
		})
	}
}

func TestResolveAnswer(t *testing.T) {
	sources := []Source{{Reference: "S1"}, {Reference: "S2"}}
	for _, tc := range []struct {
		name   string
		answer *GeneratedAnswer
		valid  bool
	}{
		{"valid", &GeneratedAnswer{Answered, "Supported [S2] and [S1].", []string{"S2", "S1"}}, true},
		{"abstain", &GeneratedAnswer{InsufficientEvidence, "", []string{}}, true},
		{"unknown", &GeneratedAnswer{Answered, "Claim [S3].", []string{"S3"}}, false},
		{"missing inline", &GeneratedAnswer{Answered, "Claim.", []string{"S1"}}, false},
		{"unlisted inline", &GeneratedAnswer{Answered, "Claim [S2].", []string{"S1"}}, false},
		{"no references", &GeneratedAnswer{Answered, "Claim.", []string{}}, false},
		{"contradictory abstain", &GeneratedAnswer{InsufficientEvidence, "Claim", []string{}}, false},
		{"abstain citations", &GeneratedAnswer{InsufficientEvidence, "", []string{"S1"}}, false},
		{"unknown status", &GeneratedAnswer{"other", "", []string{}}, false},
		{"nil", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolveAnswer(tc.answer, sources)
			if (err == nil) != tc.valid {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if tc.name == "valid" && result.Sources[0].Reference != "S1" {
				t.Fatal("source order is not context order")
			}
		})
	}
}

func TestParseGeneratedAnswer(t *testing.T) {
	for _, raw := range []string{`no JSON`, `{"status":"insufficient_evidence","references":[]}`, `{"status":"insufficient_evidence","answer":null,"references":[]}`, `null`, `{"status":"answered","answer":"x"}`, `{"status":"answered","answer":"x","references":[],"sources":[]}`, `{"status":"answered","answer":"x","references":[]} {}`} {
		if _, err := ParseGeneratedAnswer(raw); !errors.Is(err, ErrInvalidGeneration) {
			t.Fatalf("accepted %s", raw)
		}
	}
	if _, err := ParseGeneratedAnswer(`{"status":"insufficient_evidence","answer":"","references":[]}`); err != nil {
		t.Fatal(err)
	}
}

func TestSearchGenerationFailure(t *testing.T) {
	for _, ai := range []*fakeAI{{err: errors.New("provider down")}, {answer: &GeneratedAnswer{Answered, "Claim [S9]", []string{"S9"}}}} {
		repo := fakeChunks{values: []chunk.Chunk{{Content: json.RawMessage(`{"text":"Evidence"}`)}}}
		result, err := New(nil, repo, ai, ai).Search(context.Background(), &SearchQuery{Value: "question", Limit: 1})
		if err == nil || result != nil {
			t.Fatalf("failure became answer: %+v %v", result, err)
		}
	}
}
