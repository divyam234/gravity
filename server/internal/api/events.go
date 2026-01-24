package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/service"

	"github.com/go-chi/chi/v5"
)

type EventHandler struct {
	bus            *event.Bus
	downloadEngine engine.DownloadEngine
	statsService   *service.StatsService

	mu          sync.Mutex
	clientCount int
}

func NewEventHandler(bus *event.Bus, de engine.DownloadEngine, ss *service.StatsService) *EventHandler {
	return &EventHandler{
		bus:            bus,
		downloadEngine: de,
		statsService:   ss,
	}
}

func (h *EventHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Subscribe)
	return r
}

// Subscribe godoc
// @Summary Subscribe to real-time events
// @Description Open a Server-Sent Events (SSE) connection to receive real-time updates on downloads, uploads, and system status
// @Tags events
// @Produce text/event-stream
// @Success 200 {string} string "SSE connection established"
// @Router /events [get]
func (h *EventHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Increment client count and enable polling if needed
	h.mu.Lock()
	h.clientCount++
	if h.clientCount == 1 {
		if p, ok := h.downloadEngine.(interface{ ResumePolling() }); ok {
			p.ResumePolling()
		}
		h.statsService.ResumePolling()
	}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.clientCount--
		if h.clientCount == 0 {
			if p, ok := h.downloadEngine.(interface{ PausePolling() }); ok {
				p.PausePolling()
			}
			h.statsService.PausePolling()
		}
		h.mu.Unlock()
	}()

	events := h.bus.Subscribe()
	defer h.bus.Unsubscribe(events)

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\": \"connected\"}\n\n")
	flusher.Flush()

	// Keep alive ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case ev := <-events:
			evtMap := map[string]interface{}{
				"type":      ev.Type,
				"timestamp": ev.Timestamp,
				"data":      ev.Data,
			}

			jsonData, err := json.Marshal(evtMap)
			if err == nil {
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				flusher.Flush()
			}

		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}
