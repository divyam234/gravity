package api

import (
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
// @Success 200 {object} ProviderListResponse
// @Failure 500 {object} ErrorResponse
// @Router /providers [get]
func (h *ProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.List(r.Context())
	if err != nil {
		sendAppError(w, err)
		return
	}
	sendJSON(w, ProviderListResponse{Data: list})
}

// Delete godoc
// @Summary Remove provider configuration
// @Description Delete configuration for a specific provider
// @Tags providers
// @Param name path string true "Provider name"
// @Success 204 "No Content"
// @Failure 500 {object} ErrorResponse
// @Router /providers/{name} [delete]
func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.service.Delete(r.Context(), name); err != nil {
		sendAppError(w, err)
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
// @Success 200 {object} AccountInfoResponse
// @Failure 500 {object} ErrorResponse
// @Router /providers/{name}/status [get]
func (h *ProviderHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	status, err := h.service.GetStatus(r.Context(), name)
	if err != nil {
		sendAppError(w, err)
		return
	}
	sendJSON(w, AccountInfoResponse{Data: status})
}

// GetHosts godoc
// @Summary Get supported hosts
// @Description Get list of file hosts supported by a debrid provider
// @Tags providers
// @Produce json
// @Param name path string true "Provider name"
// @Success 200 {object} ProviderHostsResponse
// @Failure 500 {object} ErrorResponse
// @Router /providers/{name}/hosts [get]
func (h *ProviderHandler) GetHosts(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	hosts, err := h.service.GetHosts(r.Context(), name)
	if err != nil {
		sendAppError(w, err)
		return
	}
	sendJSON(w, ProviderHostsResponse{
		Data: ProviderHosts{Hosts: hosts},
	})
}

// Get godoc
// @Summary Get provider config schema
// @Description Get the configuration schema for a specific provider
// @Tags providers
// @Produce json
// @Param name path string true "Provider name"
// @Success 200 {object} ProviderSchemaResponse
// @Failure 404 {object} ErrorResponse
// @Router /providers/{name} [get]
func (h *ProviderHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	schema, err := h.service.GetConfigSchema(name)
	if err != nil {
		sendAppError(w, err)
		return
	}

	sendJSON(w, ProviderSchemaResponse{
		Data: ProviderSchema{
			Name:         name,
			ConfigSchema: schema,
		},
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
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /providers/{name} [put]
func (h *ProviderHandler) Configure(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var req ConfigureProviderRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	if err := h.service.Configure(r.Context(), name, req.Config, req.Enabled); err != nil {
		sendAppError(w, err)
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
// @Success 200 {object} ResolveResultResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /providers/resolve [post]
func (h *ProviderHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	var req ResolveURLRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	res, provider, err := h.service.Resolve(r.Context(), req.URL, req.Headers, req.TorrentBase64)
	if err != nil {
		sendAppError(w, err)
		return
	}

	sendJSON(w, ResolveResultResponse{
		Data: ResolveResult{
			URL:      req.URL,
			Provider: provider,
			Result:   res,
		},
	})
}
