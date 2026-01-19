package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"gravity/internal/event"

	"nhooyr.io/websocket"
)

type WSHandler struct {
	bus *event.Bus
}

func NewWSHandler(bus *event.Bus) *WSHandler {
	return &WSHandler{bus: bus}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // For development, update later
	})
	if err != nil {
		log.Printf("WS: Failed to accept connection: %v", err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "closing")

	ctx := r.Context()
	events := h.bus.Subscribe()
	defer h.bus.Unsubscribe(events)

	// Ping ticker to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			err := c.Ping(ctx)
			cancel()
			if err != nil {
				return
			}
		case ev := <-events:
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}

			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			err = c.Write(ctx, websocket.MessageText, data)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
