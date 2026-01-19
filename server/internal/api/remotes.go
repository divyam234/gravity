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

func (h *RemoteHandler) List(w http.ResponseWriter, r *http.Request) {
	remotes, err := h.engine.ListRemotes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": remotes})
}

func (h *RemoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string            `json:"name"`
		Type   string            `json:"type"`
		Config map[string]string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.engine.CreateRemote(r.Context(), req.Name, req.Type, req.Config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *RemoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.engine.DeleteRemote(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RemoteHandler) Test(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.engine.TestRemote(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
