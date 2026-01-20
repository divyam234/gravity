package api

import (
	"encoding/json"
	"net/http"

	"gravity/internal/engine"
	"gravity/internal/utils"

	"github.com/go-chi/chi/v5"
)

type FileHandler struct {
	engine engine.StorageEngine
}

func NewFileHandler(e engine.StorageEngine) *FileHandler {
	return &FileHandler{engine: e}
}

func (h *FileHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/list", h.List)
	r.Post("/mkdir", h.Mkdir)
	r.Post("/delete", h.Delete)
	r.Post("/operate", h.Operate)
	return r
}

func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	cleanPath, err := utils.SanitizePath(path, "/")
	// Special handling for remote paths like "remote:" which SanitizePath might mess up if treated as local
	// But rclone handles remotes. Here we seem to be listing files via rclone engine.
	// If the path is intended to be a remote path (e.g. "gdrive:folder"), filepath.Clean might not be appropriate if it thinks ':' is a volume separator on Windows, but on Linux it's fine.
	// However, if the engine expects a remote path, we should probably allow ':' but sanitize traversal.
	// Let's assume standard path sanitization is what we want for safety to prevent "../" traversal.
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	files, err := h.engine.List(r.Context(), cleanPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": files})
}

func (h *FileHandler) Mkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	cleanPath, err := utils.SanitizePath(req.Path, "/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.engine.Mkdir(r.Context(), cleanPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	cleanPath, err := utils.SanitizePath(req.Path, "/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.engine.Delete(r.Context(), cleanPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FileHandler) Operate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Op  string `json:"op"`
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	cleanSrc, err := utils.SanitizePath(req.Src, "/")
	if err != nil {
		http.Error(w, "invalid src path: "+err.Error(), http.StatusBadRequest)
		return
	}
	cleanDst, err := utils.SanitizePath(req.Dst, "/")
	if err != nil {
		http.Error(w, "invalid dst path: "+err.Error(), http.StatusBadRequest)
		return
	}

	var jobID string

	switch req.Op {
	case "rename":
		err = h.engine.Rename(r.Context(), cleanSrc, cleanDst)
	case "copy":
		jobID, err = h.engine.Copy(r.Context(), cleanSrc, cleanDst)
	case "move":
		jobID, err = h.engine.Move(r.Context(), cleanSrc, cleanDst)
	default:
		http.Error(w, "unknown operation", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if jobID != "" {
		json.NewEncoder(w).Encode(map[string]string{"jobId": jobID})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
