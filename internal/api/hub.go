package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

// EventHub manages connected Server-Sent Events (SSE) client channels.
type EventHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

// NewEventHub creates a new SSE EventHub.
func NewEventHub() *EventHub {
	return &EventHub{
		clients: make(map[chan []byte]struct{}),
	}
}

// Subscribe registers a new SSE listener.
func (h *EventHub) Subscribe() chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan []byte, 64)
	h.clients[ch] = struct{}{}
	return ch
}

// Unsubscribe removes an SSE listener.
func (h *EventHub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
	close(ch)
}

// Broadcast sends an SSE event to all connected listeners.
func (h *EventHub) Broadcast(eventType string, payload any) {
	evt := model.SSEEvent{
		Type:    eventType,
		Payload: payload,
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
	bytesMsg := []byte(msg)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients {
		select {
		case ch <- bytesMsg:
		default:
			// Non-blocking drop if client is saturated
		}
	}
}

// ServeSSE handles incoming HTTP SSE connections.
func (h *EventHub) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := h.Subscribe()
	defer h.Unsubscribe(ch)

	// Send initial ping
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write(msg)
			flusher.Flush()
		}
	}
}
