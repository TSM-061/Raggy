package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/shared/serviceerr"
)

type fakeSearch struct {
	result *rag.SearchResult
	err    error
	calls  int
	query  *rag.SearchQuery
}

func (f *fakeSearch) Search(_ context.Context, q *rag.SearchQuery) (*rag.SearchResult, error) {
	f.calls++
	f.query = q
	return f.result, f.err
}

func TestHandleQuery(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		result    *rag.SearchResult
		err       error
		code      int
		calls     int
	}{
		{name: "answer", url: "/api/search?q=question", result: &rag.SearchResult{Status: rag.Answered, Response: "Supported [S1].", Sources: []rag.Source{{Reference: "S1", Text: "Evidence"}}}, code: 200, calls: 1},
		{name: "abstention", url: "/api/search?q=question", result: &rag.SearchResult{Status: rag.InsufficientEvidence, Response: "Insufficient evidence", Sources: []rag.Source{}}, code: 200, calls: 1},
		{name: "missing question", url: "/api/search", code: 400},
		{name: "zero limit", url: "/api/search?q=q&limit=0", code: 400},
		{name: "large limit", url: "/api/search?q=q&limit=6", code: 400},
		{name: "invalid limit", url: "/api/search?q=q&limit=x", code: 400},
		{name: "service input", url: "/api/search?q=q", err: serviceerr.InvalidInput, code: 400, calls: 1},
		{name: "internal", url: "/api/search?q=q", err: errors.New("private provider output"), code: 500, calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeSearch{result: tc.result, err: tc.err}
			s := &Server{rag: fake}
			w := httptest.NewRecorder()
			s.HandleQuery(w, httptest.NewRequest("GET", tc.url, nil))
			if w.Code != tc.code || fake.calls != tc.calls {
				t.Fatalf("code=%d calls=%d body=%s", w.Code, fake.calls, w.Body.String())
			}
			if tc.code == 200 {
				var result rag.SearchResult
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Status != tc.result.Status || result.Response != tc.result.Response || result.Sources == nil {
					t.Fatalf("bad result: %+v", result)
				}
				if fake.query.Limit != 5 {
					t.Fatal("default limit changed")
				}
			}
			if tc.code == 500 && w.Body.String() != "search failed\n" {
				t.Fatal("internal error leaked")
			}
		})
	}
}
