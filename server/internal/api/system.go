package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"gravity/internal/engine"

	"github.com/go-chi/chi/v5"
)

type SystemHandler struct {
	downloadEngine engine.DownloadEngine
	uploadEngine   engine.UploadEngine
}

func NewSystemHandler(de engine.DownloadEngine, ue engine.UploadEngine) *SystemHandler {
	return &SystemHandler{downloadEngine: de, uploadEngine: ue}
}

func (h *SystemHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/version", h.Version)
	r.Post("/restart/aria2", h.RestartAria2)
	r.Post("/restart/rclone", h.RestartRclone)
	r.Post("/restart/server", h.RestartServer)
	return r
}

func (h *SystemHandler) Version(w http.ResponseWriter, r *http.Request) {
	dv, _ := h.downloadEngine.Version(r.Context())
	uv, _ := h.uploadEngine.Version(r.Context())

	json.NewEncoder(w).Encode(map[string]interface{}{
		"version": "0.1.0", // Gravity version
		"aria2":   dv,
		"rclone":  uv,
	})
}

func (h *SystemHandler) RestartAria2(w http.ResponseWriter, r *http.Request) {
	if err := h.downloadEngine.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.downloadEngine.Start(context.Background()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SystemHandler) RestartRclone(w http.ResponseWriter, r *http.Request) {
	if err := h.uploadEngine.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.uploadEngine.Start(context.Background()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SystemHandler) RestartServer(w http.ResponseWriter, r *http.Request) {
	// Signal the process to restart
	// In a real scenario, this might send a signal to a supervisor or just exit
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
	w.WriteHeader(http.StatusOK)
}
