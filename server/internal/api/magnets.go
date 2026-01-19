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
	r.Post("/download", h.Download)
	return r
}

// Check handles POST /api/v1/magnets/check
func (h *MagnetHandler) Check(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Magnet string `json:"magnet"`
	}
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

// Download handles POST /api/v1/magnets/download
func (h *MagnetHandler) Download(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Magnet        string   `json:"magnet"`
		Source        string   `json:"source"`
		MagnetID      string   `json:"magnetId"`
		Name          string   `json:"name"`
		SelectedFiles []string `json:"selectedFiles"`
		Destination   string   `json:"destination"`
		Files         []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Path  string `json:"path"`
			Size  int64  `json:"size"`
			Link  string `json:"link"`
			Index int    `json:"index"`
		} `json:"files"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Magnet == "" {
		http.Error(w, "magnet is required", http.StatusBadRequest)
		return
	}

	if len(req.SelectedFiles) == 0 {
		http.Error(w, "selectedFiles is required", http.StatusBadRequest)
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
		Source:        req.Source,
		MagnetID:      req.MagnetID,
		Name:          req.Name,
		SelectedFiles: req.SelectedFiles,
		Destination:   req.Destination,
		Files:         allFiles,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(download)
}
