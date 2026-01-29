package api

import (
	"context"
	"net/http"
	"strconv"

	"gravity/internal/model"
	"gravity/internal/service"

	"github.com/go-chi/chi/v5"
)

type SearchHandler struct {
	service *service.SearchService
	appCtx  context.Context
}

func NewSearchHandler(ctx context.Context, s *service.SearchService) *SearchHandler {
	return &SearchHandler{service: s, appCtx: ctx}
}

func (h *SearchHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Search)
	r.Get("/config", h.GetConfigs)
	r.Post("/config", h.BatchUpdateConfig)
	r.Post("/config/{remote}", h.UpdateConfig)
	r.Post("/index/{remote}", h.IndexRemote)
	return r
}

// BatchUpdateConfig godoc
// @Summary Batch update indexing configurations
// @Description Update indexing settings for multiple remotes at once
// @Tags search
// @Accept json
// @Param request body BatchUpdateConfigRequest true "Batch configuration request"
// @Success 200 "OK"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /search/config [post]
func (h *SearchHandler) BatchUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req BatchUpdateConfigRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	configs := make(map[string]model.RemoteIndexConfig)
	for remote, cfg := range req.Configs {
		configs[remote] = model.RemoteIndexConfig{
			AutoIndexIntervalMin: cfg.Interval,
			ExcludedPatterns:     cfg.ExcludedPatterns,
			IncludedExtensions:   cfg.IncludedExtensions,
			MinSizeBytes:         cfg.MinSizeBytes,
			Status:               "idle",
		}
	}

	if err := h.service.BatchUpdateConfig(r.Context(), configs); err != nil {
		sendAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Search godoc
// @Summary Search indexed files
// @Description Global search across all indexed cloud remotes
// @Tags search
// @Produce json
// @Param query query string true "Search query string"
// @Param limit query int false "Max number of results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} IndexedFileListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /search [get]
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get(ParamQuery)
	if q == "" {
		q = r.URL.Query().Get("query")
	}

	if q == "" {
		sendError(w, "missing query", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get(ParamLimit))
	if limit <= 0 {
		limit = DefaultLimit
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get(ParamOffset))
	if offset < 0 {
		offset = DefaultOffset
	}

	results, total, err := h.service.Search(r.Context(), q, limit, offset)
	if err != nil {
		sendAppError(w, err)
		return
	}

	if results == nil {
		results = make([]model.IndexedFile, 0)
	}

	sendJSON(w, IndexedFileListResponse{
		Data: results,
		Meta: &Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// GetConfigs godoc
// @Summary Get all indexing configurations
// @Description Get indexing settings and status for all remotes
// @Tags search
// @Produce json
// @Success 200 {object} RemoteIndexConfigListResponse
// @Failure 500 {object} ErrorResponse
// @Router /search/config [get]
func (h *SearchHandler) GetConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.GetConfigs(r.Context())
	if err != nil {
		sendAppError(w, err)
		return
	}
	sendJSON(w, RemoteIndexConfigListResponse{Data: configs})
}

// UpdateConfig godoc
// @Summary Update indexing configuration
// @Description Update indexing settings for a specific remote
// @Tags search
// @Accept json
// @Param remote path string true "Remote name"
// @Param request body UpdateConfigRequest true "Configuration request"
// @Success 200 "OK"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /search/config/{remote} [post]
func (h *SearchHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	remote := chi.URLParam(r, ParamRemote)
	var req UpdateConfigRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	if err := h.service.UpdateConfig(r.Context(), remote, req.Interval, req.ExcludedPatterns, req.IncludedExtensions, req.MinSizeBytes); err != nil {
		sendAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// IndexRemote godoc
// @Summary Trigger indexing
// @Description Manually start indexing process for a specific remote
// @Tags search
// @Param remote path string true "Remote name"
// @Success 202 "Accepted"
// @Router /search/index/{remote} [post]
func (h *SearchHandler) IndexRemote(w http.ResponseWriter, r *http.Request) {
	remote := chi.URLParam(r, ParamRemote)
	go h.service.IndexRemote(h.appCtx, remote)
	w.WriteHeader(http.StatusAccepted)
}
