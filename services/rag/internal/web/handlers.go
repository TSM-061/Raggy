package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	rag "github.com/TSM-061/Raggy/rag/internal/rag"
)

type queryResponse struct {
	Response string `json:"response"`
}

func (s *Server) HandleQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryParams := r.URL.Query()

	q := queryParams.Get("q")
	if q == "" {
		http.Error(w, "invalid param 'query' must not be empty", http.StatusBadRequest)
		return
	}

	limitStr := queryParams.Get("limit")
	limit := 5
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid param 'limit' not an int", http.StatusBadRequest)
		}
		limit = limitInt
	}

	response, err := s.rag.Search(ctx, &rag.SearchQuery{
		Value: q,
		Limit: limit,
	})
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&queryResponse{
		Response: response,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
