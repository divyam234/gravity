package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"gravity/internal/service"
	"gravity/internal/store"

	"github.com/go-chi/chi/v5"
)

type SearchHandler struct {
	service *service.SearchService
}

func NewSearchHandler(s *service.SearchService) *SearchHandler {
	return &SearchHandler{service: s}
}

func (h *SearchHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Search)
	r.Get("/config", h.GetConfigs)
	r.Post("/config/{remote}", h.UpdateConfig)
	r.Post("/index/{remote}", h.IndexRemote)
	return r
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get(ParamQuery)
	if q == "" {
		http.Error(w, "missing query", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if results == nil {
		results = make([]store.IndexedFile, 0)
	}

	json.NewEncoder(w).Encode(ListResponse{
		Data: results,
		Meta: Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

func (h *SearchHandler) GetConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.GetConfigs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": configs})
}

func (h *SearchHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	remote := chi.URLParam(r, ParamRemote)
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateConfig(r.Context(), remote, req.Interval); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SearchHandler) IndexRemote(w http.ResponseWriter, r *http.Request) {
	remote := chi.URLParam(r, ParamRemote)
	go h.service.IndexRemote(context.Background(), remote)
	w.WriteHeader(http.StatusAccepted)
}
