package progress

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Hub coalesces invalidations. PostgreSQL delivers post-commit notifications to
// every dashboard replica, regardless of which replica consumed the Kafka event.
type Hub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func NewHub() *Hub { return &Hub{clients: make(map[chan struct{}]struct{})} }
func (h *Hub) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() { h.mu.Lock(); delete(h.clients, ch); h.mu.Unlock() }
}
func (h *Hub) Broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
func (h *Hub) Run(ctx context.Context, config *pgx.ConnConfig) {
	for ctx.Err() == nil {
		conn, err := pgx.ConnectConfig(ctx, config.Copy())
		if err == nil {
			_, err = conn.Exec(ctx, "LISTEN upload_progress")
			if err == nil {
				h.Broadcast() // Reconcile anything missed during a listener outage.
				for ctx.Err() == nil {
					if _, err = conn.WaitForNotification(ctx); err != nil {
						break
					}
					h.Broadcast()
				}
			}
			closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = conn.Close(closeCtx)
			cancel()
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
