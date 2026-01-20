package api

import (
	"context"
	"encoding/json"
	"net/http"

	"gravity/internal/engine"
	"gravity/internal/store"

	"github.com/go-chi/chi/v5"
)

type SettingsHandler struct {
	repo         *store.SettingsRepo
	providerRepo *store.ProviderRepo
	engine       engine.DownloadEngine
	uploadEngine engine.UploadEngine
}

func NewSettingsHandler(repo *store.SettingsRepo, providerRepo *store.ProviderRepo, engine engine.DownloadEngine, uploadEngine engine.UploadEngine) *SettingsHandler {
	return &SettingsHandler{
		repo:         repo,
		providerRepo: providerRepo,
		engine:       engine,
		uploadEngine: uploadEngine,
	}
}

func (h *SettingsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	r.Patch("/", h.Update)
	r.Get("/status", h.GetStatus)
	r.Post("/export", h.Export)
	r.Post("/import", h.Import)
	r.Post("/reset", h.Reset)
	return r
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings, err := h.repo.Get(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	remotes, _ := h.uploadEngine.ListRemotes(ctx)
	providers, _ := h.providerRepo.List(ctx)

	status := map[string]interface{}{
		"downloads": map[string]interface{}{
			"configured":   settings["download_dir"] != "",
			"downloadPath": settings["download_dir"],
		},
		"cloud": map[string]interface{}{
			"configured":         len(remotes) > 0,
			"remoteCount":        len(remotes),
			"defaultDestination": settings["default_remote"],
		},
		"premium": map[string]interface{}{
			"providers": providers, // Simplify for now, maybe map to specific fields
		},
		"network": map[string]interface{}{
			"proxyEnabled": settings["all-proxy"] != "",
		},
		"torrents": map[string]interface{}{
			"seedingEnabled": settings["seed-ratio"] != "0",
		},
	}

	json.NewEncoder(w).Encode(status)
}

func (h *SettingsHandler) Export(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=gravity-settings.json")
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) Import(w http.ResponseWriter, r *http.Request) {
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.repo.SetMany(r.Context(), settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Re-apply to engine
	go h.engine.Configure(context.Background(), settings)

	w.WriteHeader(http.StatusOK)
}

func (h *SettingsHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.DeleteAll(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	for k, v := range req {
		if err := h.repo.Set(r.Context(), k, v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Apply settings to engine
	if err := h.engine.Configure(r.Context(), req); err != nil {
		// Log warning but don't fail request? Or return partial error?
		// For now, let's treat it as success but maybe log it.
		// Since I don't have logger here easily, I'll return 500 if engine fails.
		http.Error(w, "Saved but failed to apply to download engine: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.uploadEngine.Configure(r.Context(), req); err != nil {
		http.Error(w, "Saved but failed to apply to upload engine: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
