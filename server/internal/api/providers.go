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
	r.Delete("/{name}", h.Delete)
	r.Get("/{name}/status", h.GetStatus)
	r.Get("/{name}/hosts", h.GetHosts)
	r.Post("/resolve", h.Resolve)
	return r
}

// List godoc
// @Summary List providers
// @Description Get a list of all configured download providers
// @Tags providers
// @Produce json
// @Success 200 {object} map[string][]model.Provider
// @Failure 500 {string} string "Internal Server Error"
// @Router /providers [get]
func (h *ProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": list})
}

// Delete godoc
// @Summary Remove provider configuration
// @Description Delete configuration for a specific provider
// @Tags providers
// @Param name path string true "Provider name"
// @Success 204 "No Content"
// @Failure 500 {string} string "Internal Server Error"
// @Router /providers/{name} [delete]
func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.service.Delete(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetStatus godoc
// @Summary Get provider status
// @Description Get current status and account info for a provider
// @Tags providers
// @Produce json
// @Param name path string true "Provider name"
// @Success 200 {object} model.AccountInfo
// @Failure 500 {string} string "Internal Server Error"
// @Router /providers/{name}/status [get]
func (h *ProviderHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	status, err := h.service.GetStatus(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(status)
}

// GetHosts godoc
// @Summary Get supported hosts
// @Description Get list of file hosts supported by a debrid provider
// @Tags providers
// @Produce json
// @Param name path string true "Provider name"
// @Success 200 {object} map[string][]string
// @Failure 500 {string} string "Internal Server Error"
// @Router /providers/{name}/hosts [get]
func (h *ProviderHandler) GetHosts(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	hosts, err := h.service.GetHosts(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"hosts": hosts})
}

// Get godoc
// @Summary Get provider config schema
// @Description Get the configuration schema for a specific provider
// @Tags providers
// @Produce json
// @Param name path string true "Provider name"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {string} string "Not Found"
// @Router /providers/{name} [get]
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

// Configure godoc
// @Summary Update provider configuration
// @Description Enable/disable or update configuration settings for a provider
// @Tags providers
// @Accept json
// @Param name path string true "Provider name"
// @Param request body ConfigureProviderRequest true "Configuration request"
// @Success 200 "OK"
// @Failure 400 {string} string "Invalid Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /providers/{name} [put]
func (h *ProviderHandler) Configure(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var req ConfigureProviderRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	if err := h.service.Configure(r.Context(), name, req.Config, req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Resolve godoc
// @Summary Resolve URL
// @Description Resolve a download URL through providers to get direct links
// @Tags providers
// @Accept json
// @Produce json
// @Param request body ResolveURLRequest true "Resolve request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Invalid Request"
// @Failure 404 {string} string "Not Found"
// @Router /providers/resolve [post]

func (h *ProviderHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	var req ResolveURLRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	res, provider, err := h.service.Resolve(r.Context(), req.URL, req.Headers)
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
