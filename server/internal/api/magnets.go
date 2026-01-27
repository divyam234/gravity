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

// Check godoc
// @Summary Check magnet link
// @Description Check availability and retrieve file list for a magnet link
// @Tags magnets
// @Accept json
// @Produce json
// @Param request body CheckMagnetRequest true "Magnet check request"
// @Success 200 {object} MagnetInfoResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /magnets/check [post]
func (h *MagnetHandler) Check(w http.ResponseWriter, r *http.Request) {
	var req CheckMagnetRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	info, err := h.magnetService.CheckMagnet(r.Context(), req.Magnet)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, MagnetInfoResponse{Data: info})
}

// CheckTorrent godoc
// @Summary Check torrent file
// @Description Check availability and retrieve file list for a .torrent file (base64)
// @Tags magnets
// @Accept json
// @Produce json
// @Param request body CheckTorrentRequest true "Torrent check request"
// @Success 200 {object} MagnetInfoResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /magnets/check-torrent [post]
func (h *MagnetHandler) CheckTorrent(w http.ResponseWriter, r *http.Request) {
	var req CheckTorrentRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	info, err := h.magnetService.CheckTorrent(r.Context(), req.TorrentBase64)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, MagnetInfoResponse{Data: info})
}

// Download godoc
// @Summary Download from magnet/torrent
// @Description Start a download using selected files from a magnet or torrent
// @Tags magnets
// @Accept json
// @Produce json
// @Param request body DownloadMagnetRequest true "Magnet download request"
// @Success 201 {object} DownloadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /magnets/download [post]
func (h *MagnetHandler) Download(w http.ResponseWriter, r *http.Request) {
	var req DownloadMagnetRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	// Convert files to model
	var allFiles []*model.MagnetFile
	for _, f := range req.Files {
		allFiles = append(allFiles, &model.MagnetFile{
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
		DownloadDir:   req.DownloadDir,
		Destination:   req.Destination,
		AllFiles:      allFiles,
	})

	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(DownloadResponse{Data: download})
}
