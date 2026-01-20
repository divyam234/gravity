package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/pause", h.Pause)
	r.Post("/{id}/resume", h.Resume)
	r.Post("/{id}/retry", h.Retry)
	return r
}

func (h *DownloadHandler) List(w http.ResponseWriter, r *http.Request) {
	statusStr := r.URL.Query().Get("status")
	var status []string
	if statusStr != "" {
		status = strings.Split(statusStr, ",")
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	downloads, total, err := h.service.List(r.Context(), status, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": downloads,
		"meta": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (h *DownloadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL         string `json:"url"`
		Filename    string `json:"filename"`
		Destination string `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	d, err := h.service.Create(r.Context(), req.URL, req.Filename, req.Destination)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(d)
}

func (h *DownloadHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, err := h.service.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(d)
}

func (h *DownloadHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deleteFiles := r.URL.Query().Get("deleteFiles") == "true"

	if err := h.service.Delete(r.Context(), id, deleteFiles); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DownloadHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Pause(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *DownloadHandler) Resume(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Resume(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *DownloadHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Retry(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
