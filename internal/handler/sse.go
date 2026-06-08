package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/sse"
)

type SSEHandler struct {
	broker *sse.Broker
}

func NewSSEHandler(broker *sse.Broker) *SSEHandler {
	return &SSEHandler{broker: broker}
}

func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send an immediate comment so headers reach the client without waiting
	// for the first real event. Without this, EventSource stays in
	// CONNECTING state until the 15s heartbeat fires — which causes any
	// event published during that window to be missed by subscribers that
	// connected just before the publish.
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ch := h.broker.Subscribe()
	defer h.broker.Unsubscribe(ch)

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-ch:
			if ev.Name != "" {
				fmt.Fprintf(w, "event: %s\n", ev.Name)
			}
			fmt.Fprintf(w, "data: %s\n\n", ev.Data)
			flusher.Flush()
		case <-heartbeat.C:
			// Emit a real (named) event, not a `:` comment. EventSource does
			// not surface comments to JS, so the client can't tell a silently
			// dead connection (TCP open, no data) from an idle one. A `ping`
			// event lets the client's watchdog reset and force a reconnect
			// when pings stop arriving.
			fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}
