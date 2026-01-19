package api

import (
	"encoding/json"
	"net/http"

	"gravity/internal/service"

	"github.com/go-chi/chi/v5"
)

type ProviderHandler struct {
	service *service.ProviderService
}

func NewProviderHandler(s *service.ProviderService) *ProviderHandler {
	return &ProviderHandler{service: s}
}

func (h *ProviderHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{name}", h.Get)
	r.Put("/{name}", h.Configure)
	r.Post("/resolve", h.Resolve)
	return r
}

func (h *ProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": list})
}

func (h *ProviderHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	schema, err := h.service.GetConfigSchema(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Just return schema for now
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":         name,
		"configSchema": schema,
	})
}

func (h *ProviderHandler) Configure(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var req struct {
		Config  map[string]string `json:"config"`
		Enabled bool              `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.service.Configure(r.Context(), name, req.Config, req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ProviderHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	res, provider, err := h.service.Resolve(r.Context(), req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":      req.URL,
		"provider": provider,
		"result":   res,
	})
}
