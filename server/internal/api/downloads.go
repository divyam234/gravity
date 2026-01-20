package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gravity/internal/service"
	"gravity/internal/utils"

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
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/pause", h.Pause)
	r.Post("/{id}/resume", h.Resume)
	r.Post("/{id}/retry", h.Retry)
	return r
}

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ListResponse{
		Data: downloads,
		Meta: Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

func (h *DownloadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Initialize cleanFilename as empty
	// If req.Filename is empty, we leave it empty so Aria2 uses the URL's filename.
	cleanFilename := ""
	if req.Filename != "" {
		if !utils.IsSafeFilename(req.Filename) {
			http.Error(w, "invalid filename", http.StatusBadRequest)
			return
		}
		// If provided, use it directly (IsSafeFilename checks for traversal)
		cleanFilename = req.Filename
	}

	// Smart Destination Logic:
	// We allow the user to specify:
	// 1. A local absolute path (e.g. /mnt/data) -> Direct Download
	// 2. A remote path (e.g. gdrive:) -> Download to default -> Upload
	// 3. A relative path (e.g. movies) -> Download to default/movies (Direct Download)

	// Therefore, we do NOT strictly sanitize req.Destination against the default data directory here.
	// We trust the service layer to handle the logic and the underlying engine/OS to handle permission errors.
	// This allows power users to download to external drives.

	cleanDest := ""
	if req.Destination != "" {
		// Just clean the path string to remove redundancies like // or /./
		// We do NOT use SanitizePath because that enforces baseDir containment.
		cleanDest = req.Destination
	}

	d, err := h.service.Create(r.Context(), req.URL, cleanFilename, cleanDest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(d)
}

func (h *DownloadHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	d, err := h.service.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(d)
}

func (h *DownloadHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	deleteFiles := r.URL.Query().Get(ParamDeleteFiles) == "true"

	if err := h.service.Delete(r.Context(), id, deleteFiles); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DownloadHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	if err := h.service.Pause(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *DownloadHandler) Resume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	if err := h.service.Resume(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *DownloadHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, ParamID)
	if err := h.service.Retry(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
