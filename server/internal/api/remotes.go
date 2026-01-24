package api

import (
	"encoding/json"
	"net/http"

	"gravity/internal/engine"

	"github.com/go-chi/chi/v5"
)

type RemoteHandler struct {
	engine engine.UploadEngine
}

func NewRemoteHandler(e engine.UploadEngine) *RemoteHandler {
	return &RemoteHandler{engine: e}
}

func (h *RemoteHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Delete("/{name}", h.Delete)
	r.Post("/{name}/test", h.Test)
	return r
}

// List godoc
// @Summary List cloud remotes
// @Description Get a list of all configured rclone remotes
// @Tags remotes
// @Produce json
// @Success 200 {object} map[string][]engine.Remote
// @Failure 500 {string} string "Internal Server Error"
// @Router /remotes [get]
func (h *RemoteHandler) List(w http.ResponseWriter, r *http.Request) {
	remotes, err := h.engine.ListRemotes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": remotes})
}

// Create godoc
// @Summary Create remote
// @Description Add a new cloud storage remote configuration
// @Tags remotes
// @Accept json
// @Param request body CreateRemoteRequest true "Remote creation request"
// @Success 201 "Created"
// @Failure 400 {string} string "Invalid Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /remotes [post]
func (h *RemoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRemoteRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	if err := h.engine.CreateRemote(r.Context(), req.Name, req.Type, req.Config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Delete godoc
// @Summary Delete remote
// @Description Remove a cloud storage remote configuration
// @Tags remotes
// @Param name path string true "Remote name"
// @Success 204 "No Content"
// @Failure 500 {string} string "Internal Server Error"
// @Router /remotes/{name} [delete]
func (h *RemoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.engine.DeleteRemote(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Test godoc
// @Summary Test remote connection
// @Description Verify connectivity to a configured cloud storage remote
// @Tags remotes
// @Param name path string true "Remote name"
// @Success 200 "OK"
// @Failure 500 {string} string "Internal Server Error"
// @Router /remotes/{name}/test [post]
func (h *RemoteHandler) Test(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.engine.TestRemote(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
