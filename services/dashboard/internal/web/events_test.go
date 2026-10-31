package web

import (
	"bufio"
	"github.com/TSM-061/Raggy/dashboard/internal/progress"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEventsReconcileAndPublish(t *testing.T) {
	hub := progress.NewHub()
	s := &Server{progress: hub}
	server := httptest.NewServer(http.HandlerFunc(s.HandleEvents))
	defer server.Close()
	client := server.Client()
	client.Timeout = 3 * time.Second
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") != "text/event-stream" || response.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatal("stream headers missing")
	}
	reader := bufio.NewReader(response.Body)
	read := func() {
		t.Helper()
		line, err := reader.ReadString('\n')
		if err != nil || line != "event: uploads.changed\n" {
			t.Fatalf("%q %v", line, err)
		}
		_, _ = reader.ReadString('\n')
		_, _ = reader.ReadString('\n')
	}
	read()
	hub.Broadcast()
	read()
}
