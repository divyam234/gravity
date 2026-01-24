package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"gravity/internal/engine"
	"gravity/internal/engine/hybrid"

	"github.com/go-chi/chi/v5"
)

type SystemHandler struct {
	downloadEngine engine.DownloadEngine
	uploadEngine   engine.UploadEngine
	appCtx         context.Context
}

func NewSystemHandler(ctx context.Context, de engine.DownloadEngine, ue engine.UploadEngine) *SystemHandler {
	return &SystemHandler{downloadEngine: de, uploadEngine: ue, appCtx: ctx}
}

func (h *SystemHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/version", h.Version)
	r.Post("/restart/aria2", h.RestartAria2)
	r.Post("/restart/rclone", h.RestartRclone)
	r.Post("/restart/server", h.RestartServer)
	return r
}

// Version godoc
// @Summary Get system versions
// @Description Get versions of Gravity, Aria2, and Rclone
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /system/version [get]
func (h *SystemHandler) Version(w http.ResponseWriter, r *http.Request) {
	var aria2Ver, nativeVer string

	if router, ok := h.downloadEngine.(*hybrid.HybridRouter); ok {
		aria2Ver, nativeVer = router.GetVersions(r.Context())
	} else {
		// Fallback for non-hybrid setup
		aria2Ver, _ = h.downloadEngine.Version(r.Context())
	}

	uv, _ := h.uploadEngine.Version(r.Context())

	json.NewEncoder(w).Encode(map[string]interface{}{
		"version": "0.1.0", // Gravity version
		"aria2":   aria2Ver,
		"native":  nativeVer,
		"rclone":  uv,
	})
}

// RestartAria2 godoc
// @Summary Restart Aria2 engine
// @Description Stop and restart the underlying Aria2 download engine
// @Tags system
// @Success 200 "OK"
// @Failure 500 {string} string "Internal Server Error"
// @Router /system/restart/aria2 [post]
func (h *SystemHandler) RestartAria2(w http.ResponseWriter, r *http.Request) {
	if err := h.downloadEngine.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.downloadEngine.Start(h.appCtx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// RestartRclone godoc
// @Summary Restart Rclone engine
// @Description Stop and restart the underlying Rclone upload engine
// @Tags system
// @Success 200 "OK"
// @Failure 500 {string} string "Internal Server Error"
// @Router /system/restart/rclone [post]
func (h *SystemHandler) RestartRclone(w http.ResponseWriter, r *http.Request) {
	if err := h.uploadEngine.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.uploadEngine.Start(h.appCtx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// RestartServer godoc
// @Summary Restart Gravity server
// @Description Exit the current process (expects supervisor to restart)
// @Tags system
// @Success 200 "OK"
// @Router /system/restart/server [post]
func (h *SystemHandler) RestartServer(w http.ResponseWriter, r *http.Request) {
	// Signal the process to restart
	// In a real scenario, this might send a signal to a supervisor or just exit
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
	w.WriteHeader(http.StatusOK)
}
