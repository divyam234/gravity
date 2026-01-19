package api

import (
	"encoding/json"
	"net/http"

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
