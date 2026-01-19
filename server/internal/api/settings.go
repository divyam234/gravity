package api

import (
	"encoding/json"
	"net/http"

	"gravity/internal/engine"
	"gravity/internal/store"

	"github.com/go-chi/chi/v5"
)

type SettingsHandler struct {
	repo   *store.SettingsRepo
	engine engine.DownloadEngine
}

func NewSettingsHandler(repo *store.SettingsRepo, engine engine.DownloadEngine) *SettingsHandler {
	return &SettingsHandler{repo: repo, engine: engine}
}

func (h *SettingsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	r.Patch("/", h.Update)
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
		http.Error(w, "Saved but failed to apply: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
