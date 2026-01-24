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

// GetCurrent godoc
// @Summary Get live system statistics
// @Description Get real-time download/upload speeds, storage usage, and active task counts
// @Tags stats
// @Produce json
// @Success 200 {object} model.Stats
// @Failure 500 {string} string "Internal Server Error"
// @Router /stats [get]
func (h *StatsHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetCurrent(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stats)
}
