package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gravity/internal/model"
	"gravity/internal/service"

	"github.com/go-chi/chi/v5"
)

type DownloadHandler struct {
	service *service.DownloadService
}

func NewDownloadHandler(s *service.DownloadService) *DownloadHandler {
	return &DownloadHandler{service: s}
}

func (h *DownloadHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Post("/batch", h.Batch)
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Delete)
	r.Patch("/{id}", h.Update)
	r.Post("/{id}/pause", h.Pause)
	r.Post("/{id}/resume", h.Resume)
	r.Post("/{id}/retry", h.Retry)
	r.Patch("/{id}/priority", h.UpdatePriority)
	return r
}

type UpdatePriorityRequest struct {
	Priority int `json:"priority" validate:"min=1,max=10"`
}

func (h *DownloadHandler) UpdatePriority(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	var req UpdatePriorityRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	if err := h.service.UpdatePriority(r.Context(), id, req.Priority); err != nil {
		sendAppError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Batch godoc
// @Summary Batch operations
// @Description Perform batch operations (pause, resume, delete, retry) on multiple downloads
// @Tags downloads
// @Accept json
// @Produce json
// @Param request body BatchActionRequest true "Batch request"
// @Success 200 "OK"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /downloads/batch [post]
func (h *DownloadHandler) Batch(w http.ResponseWriter, r *http.Request) {
	var req BatchActionRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	if err := h.service.Batch(r.Context(), req.IDs, req.Action); err != nil {
		sendAppError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// List godoc
// @Summary List downloads
// @Description Get a paginated list of downloads with optional status filtering
// @Tags downloads
// @Produce json
// @Param status query string false "Comma-separated statuses to filter by"
// @Param limit query int false "Max number of items to return"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} DownloadListResponse
// @Failure 500 {object} ErrorResponse
// @Router /downloads [get]
func (h *DownloadHandler) List(w http.ResponseWriter, r *http.Request) {
	statusStr := r.URL.Query().Get(ParamStatus)
	var status []string
	if statusStr != "" {
		status = strings.Split(statusStr, ",")
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get(ParamLimit))
	if limit == 0 {
		limit = DefaultLimit
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get(ParamOffset))

	downloads, total, err := h.service.List(r.Context(), status, limit, offset)
	if err != nil {
		sendAppError(w, err)
		return
	}

	sendJSON(w, DownloadListResponse{
		Data: downloads,
		Meta: &Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Update godoc
// @Summary Update download
// @Description Update download properties
// @Tags downloads
// @Accept json
// @Produce json
// @Param id path string true "Download ID"
// @Param request body UpdateDownloadRequest true "Update request"
// @Success 200 "OK"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /downloads/{id} [patch]
func (h *DownloadHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	var req UpdateDownloadRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	if err := h.service.Update(r.Context(), id, req.Filename, req.Destination, req.Priority, req.MaxRetries); err != nil {
		sendAppError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Create godoc

// @Summary Create download
// @Description Start a new download from a URL
// @Tags downloads
// @Accept json
// @Produce json
// @Param request body CreateDownloadRequest true "Download request"
// @Success 201 {object} DownloadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /downloads [post]
func (h *DownloadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDownloadRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	// 1. Map request to carrier model
	opts := model.Download{
		URL:           req.URL,
		Filename:      req.Filename,
		Dir:           req.Dir,
		Destination:   req.Destination,
		Split:         req.Split,
		RemoveLocal:   req.RemoveLocal,
		Headers:       req.Headers,
		Engine:        req.Engine,
		TorrentData:   req.TorrentData,
		MagnetHash:    req.Hash,
		SelectedFiles: req.SelectedFiles,

		// Overrides
		MaxDownloadSpeed: req.MaxDownloadSpeed,
		ConnectTimeout:   req.ConnectTimeout,
	}

	if req.Priority != nil {
		opts.Priority = *req.Priority
	}
	if req.MaxRetries != nil {
		opts.MaxRetries = *req.MaxRetries
	}

	if req.Hash != "" || req.TorrentData != "" || strings.HasPrefix(req.URL, "magnet:") {
		opts.IsMagnet = true
	}

	// Map Proxies
	opts.Proxies = req.Proxies

	// 2. Call service
	d, err := h.service.Create(r.Context(), &opts)
	if err != nil {
		sendAppError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(DownloadResponse{Data: d})
}

// Get godoc
// @Summary Get download
// @Description Get details of a specific download by ID
// @Tags downloads
// @Produce json
// @Param id path string true "Download ID"
// @Success 200 {object} DownloadResponse
// @Failure 404 {object} ErrorResponse
// @Router /downloads/{id} [get]
func (h *DownloadHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	d, err := h.service.Get(r.Context(), id)
	if err != nil {
		sendAppError(w, err)
		return
	}
	sendJSON(w, DownloadResponse{Data: d})
}

// Delete godoc
// @Summary Delete download
// @Description Stop and delete a download, optionally removing files
// @Tags downloads
// @Param id path string true "Download ID"
// @Param delete_files query bool false "If true, deletes files from disk"
// @Success 204 "No Content"
// @Failure 500 {object} ErrorResponse
// @Router /downloads/{id} [delete]
func (h *DownloadHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	deleteFiles := r.URL.Query().Get(ParamDeleteFiles) == "true"

	if err := h.service.Delete(r.Context(), id, deleteFiles); err != nil {
		sendAppError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Pause godoc
// @Summary Pause download
// @Description Pause an active download
// @Tags downloads
// @Param id path string true "Download ID"
// @Success 200 "OK"
// @Failure 500 {object} ErrorResponse
// @Router /downloads/{id}/pause [post]
func (h *DownloadHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	if err := h.service.Pause(r.Context(), id); err != nil {
		sendAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Resume godoc
// @Summary Resume download
// @Description Resume a paused download
// @Tags downloads
// @Param id path string true "Download ID"
// @Success 200 "OK"
// @Failure 500 {object} ErrorResponse
// @Router /downloads/{id}/resume [post]
func (h *DownloadHandler) Resume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	if err := h.service.Resume(r.Context(), id); err != nil {
		sendAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Retry godoc
// @Summary Retry download
// @Description Retry a failed or completed download
// @Tags downloads
// @Param id path string true "Download ID"
// @Success 200 "OK"
// @Failure 500 {object} ErrorResponse
// @Router /downloads/{id}/retry [post]
func (h *DownloadHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	if err := h.service.Retry(r.Context(), id); err != nil {
		sendAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
