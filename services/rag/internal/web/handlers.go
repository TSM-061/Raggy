package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/shared/logger"
)

type queryResponse struct {
	Response string `json:"response"`
}

func (s *Server) HandleQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)

	params := r.URL.Query()

	q := params.Get("q")
	if q == "" {
		http.Error(w, "invalid param 'q' must not be empty", http.StatusBadRequest)
		return
	}

	limit := 5
	if limitStr := params.Get("limit"); limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid queryparam 'limit' must be an integer", http.StatusBadRequest)
			return
		}

		if limitInt <= 0 {
			http.Error(w, "invalid queryparam 'limit' must be > 0", http.StatusBadRequest)
			return
		}
		limit = limitInt
	}

	response, err := s.rag.Search(ctx, &rag.SearchQuery{
		Value: q,
		Limit: limit,
	})
	if err != nil {

		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&queryResponse{Response: response}); err != nil {
		log.ErrorContext(
			ctx,
			`failed to encode query response JSON`,
			slog.Any("error", err),
		)
		http.Error(
			w,
			fmt.Sprintf("failed to encode response: %v", err),
			http.StatusInternalServerError,
		)
	}
}
