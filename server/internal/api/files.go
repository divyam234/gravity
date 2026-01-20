package api

import (
	"encoding/json"
	"net/http"

	"gravity/internal/engine"

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
	files, err := h.engine.List(r.Context(), path)
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

	if err := h.engine.Mkdir(r.Context(), req.Path); err != nil {
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

	if err := h.engine.Delete(r.Context(), req.Path); err != nil {
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

	var err error
	var jobID string

	switch req.Op {
	case "rename":
		err = h.engine.Rename(r.Context(), req.Src, req.Dst)
	case "copy":
		jobID, err = h.engine.Copy(r.Context(), req.Src, req.Dst)
	case "move":
		jobID, err = h.engine.Move(r.Context(), req.Src, req.Dst)
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
