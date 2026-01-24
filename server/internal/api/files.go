package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"gravity/internal/engine"
	"gravity/internal/utils"

	"github.com/go-chi/chi/v5"
)

type FileHandler struct {
	storage engine.StorageEngine
	upload  engine.UploadEngine
}

func NewFileHandler(s engine.StorageEngine, u engine.UploadEngine) *FileHandler {
	return &FileHandler{
		storage: s,
		upload:  u,
	}
}

func (h *FileHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/list", h.List)
	r.Get("/cat", h.Cat)
	r.Post("/mkdir", h.Mkdir)
	r.Post("/delete", h.Delete)
	r.Post("/operate", h.Operate)
	r.Post("/restart", h.Restart)
	return r
}

// Restart godoc
// @Summary Restart VFS
// @Description Restart the Virtual File System engine
// @Tags files
// @Success 200 "OK"
// @Failure 500 {string} string "Internal Server Error"
// @Router /files/restart [post]
func (h *FileHandler) Restart(w http.ResponseWriter, r *http.Request) {
	if err := h.upload.Restart(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Cat godoc
// @Summary Read file content
// @Description Stream file content (supports Range requests)
// @Tags files
// @Param path query string true "Virtual path to file"
// @Success 200 {file} file "File content"
// @Failure 400 {string} string "Invalid Path"
// @Failure 404 {string} string "Not Found"
// @Router /files/cat [get]
func (h *FileHandler) Cat(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	cleanPath, err := utils.SanitizePath(path, "/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	info, err := h.storage.Stat(r.Context(), cleanPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	rc, err := h.storage.Open(r.Context(), cleanPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	// Use http.ServeContent to support Range requests (Seeking)
	http.ServeContent(w, r, info.Name, info.ModTime, rc)
}

// List godoc
// @Summary List files
// @Description List files and directories at a path
// @Tags files
// @Produce json
// @Param path query string false "Virtual path (default root)"
// @Success 200 {object} map[string][]engine.FileInfo
// @Failure 500 {string} string "Internal Server Error"
// @Router /files/list [get]
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	cleanPath, err := utils.SanitizePath(path, "/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	files, err := h.storage.List(r.Context(), cleanPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": files})
}

// Mkdir godoc
// @Summary Create directory
// @Description Create a new directory
// @Tags files
// @Accept json
// @Param request body MkdirRequest true "Directory path"
// @Success 201 "Created"
// @Failure 400 {string} string "Invalid Path"
// @Failure 500 {string} string "Internal Server Error"
// @Router /files/mkdir [post]
func (h *FileHandler) Mkdir(w http.ResponseWriter, r *http.Request) {
	var req MkdirRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	cleanPath, err := utils.SanitizePath(req.Path, "/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.storage.Mkdir(r.Context(), cleanPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Delete godoc
// @Summary Delete file/directory
// @Description Delete a file or directory
// @Tags files
// @Accept json
// @Param request body DeleteFileRequest true "Path to delete"
// @Success 204 "No Content"
// @Failure 400 {string} string "Invalid Path"
// @Failure 500 {string} string "Internal Server Error"
// @Router /files/delete [post]
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteFileRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	cleanPath, err := utils.SanitizePath(req.Path, "/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.storage.Delete(r.Context(), cleanPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FileHandler) Operate(w http.ResponseWriter, r *http.Request) {
	var req FileOperationRequest
	if !decodeAndValidate(w, r, &req) {
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
	case "copy":
		jobID, err = h.upload.Copy(r.Context(), cleanSrc, cleanDst)
	case "move":
		jobID, err = h.upload.Move(r.Context(), cleanSrc, cleanDst)
	case "rename":
		err = h.storage.Rename(r.Context(), cleanSrc, filepath.Base(cleanDst))
	default:
		http.Error(w, "invalid operation", http.StatusBadRequest)
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
