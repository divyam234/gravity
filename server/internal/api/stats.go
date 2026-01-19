package api

import (
	"encoding/json"
	"net/http"

	"gravity/internal/service"

	"github.com/go-chi/chi/v5"
)

type StatsHandler struct {
	service *service.StatsService
}

func NewStatsHandler(s *service.StatsService) *StatsHandler {
	return &StatsHandler{service: s}
}

func (h *StatsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetCurrent)
	return r
}

func (h *StatsHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetCurrent(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stats)
}
