package web

import (
	"fmt"
	"net/http"
	"time"
)

// Events carry invalidations only; authenticated REST snapshots remain authoritative.
func (s *Server) HandleEvents(w http.ResponseWriter, r *http.Request) {
	if s.progress == nil {
		http.Error(w, "progress unavailable", http.StatusServiceUnavailable)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	changes, unsubscribe := s.progress.Subscribe()
	defer unsubscribe()
	controller := http.NewResponseController(w)
	write := func(message string) bool {
		_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := fmt.Fprint(w, message); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}
	if !write("event: uploads.changed\ndata: {}\n\n") {
		return
	}
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	// Reauthenticate periodically rather than keeping an expired session forever.
	lifetime := time.NewTimer(5 * time.Minute)
	defer lifetime.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-lifetime.C:
			return
		case <-heartbeat.C:
			if !write(": heartbeat\n\n") {
				return
			}
		case <-changes:
			if !write("event: uploads.changed\ndata: {}\n\n") {
				return
			}
		}
	}
}
