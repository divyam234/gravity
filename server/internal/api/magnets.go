package api

import (
	"encoding/json"
	"net/http"

	"gravity/internal/model"
	"gravity/internal/service"

	"github.com/go-chi/chi/v5"
)

type MagnetHandler struct {
	magnetService *service.MagnetService
}

func NewMagnetHandler(s *service.MagnetService) *MagnetHandler {
	return &MagnetHandler{magnetService: s}
}

func (h *MagnetHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/check", h.Check)
	r.Post("/check-torrent", h.CheckTorrent)
	r.Post("/download", h.Download)
	return r
}

func (h *MagnetHandler) Check(w http.ResponseWriter, r *http.Request) {
	var req CheckMagnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Magnet == "" {
		http.Error(w, "magnet is required", http.StatusBadRequest)
		return
	}

	info, err := h.magnetService.CheckMagnet(r.Context(), req.Magnet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *MagnetHandler) CheckTorrent(w http.ResponseWriter, r *http.Request) {
	var req CheckTorrentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.TorrentBase64 == "" {
		http.Error(w, "torrentBase64 is required", http.StatusBadRequest)
		return
	}

	info, err := h.magnetService.CheckTorrent(r.Context(), req.TorrentBase64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *MagnetHandler) Download(w http.ResponseWriter, r *http.Request) {
	var req DownloadMagnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Convert files to model
	var allFiles []model.MagnetFile
	for _, f := range req.Files {
		allFiles = append(allFiles, model.MagnetFile{
			ID:    f.ID,
			Name:  f.Name,
			Path:  f.Path,
			Size:  f.Size,
			Link:  f.Link,
			Index: f.Index,
		})
	}

	download, err := h.magnetService.DownloadMagnet(r.Context(), service.MagnetDownloadRequest{
		Magnet:        req.Magnet,
		TorrentBase64: req.TorrentBase64,
		Source:        req.Source,
		MagnetID:      req.MagnetID,
		Name:          req.Name,
		SelectedFiles: req.SelectedFiles,
		Destination:   req.Destination,
		AllFiles:      allFiles,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(download)
}
