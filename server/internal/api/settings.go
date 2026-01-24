package api

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"gravity/internal/engine"
	"gravity/internal/event"
	"gravity/internal/model"
	"gravity/internal/store"

	"github.com/go-chi/chi/v5"
)

type SettingsHandler struct {
	repo         *store.SettingsRepo
	providerRepo *store.ProviderRepo
	engine       engine.DownloadEngine
	uploadEngine engine.UploadEngine
	bus          *event.Bus
}

func NewSettingsHandler(repo *store.SettingsRepo, providerRepo *store.ProviderRepo, engine engine.DownloadEngine, uploadEngine engine.UploadEngine, bus *event.Bus) *SettingsHandler {
	return &SettingsHandler{
		repo:         repo,
		providerRepo: providerRepo,
		engine:       engine,
		uploadEngine: uploadEngine,
		bus:          bus,
	}
}

func (h *SettingsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	r.Patch("/", h.Update)
	r.Get("/status", h.GetStatus)
	r.Post("/export", h.Export)
	r.Post("/import", h.Import)
	r.Post("/reset", h.Reset)
	return r
}

// Get godoc
// @Summary Get application settings
// @Description Retrieve all application settings organized by category
// @Tags settings
// @Produce json
// @Success 200 {object} model.Settings
// @Failure 500 {string} string "Internal Server Error"
// @Router /settings [get]
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if settings == nil {
		// Provide defaults if not found
		settings = model.DefaultSettings()
	}

	json.NewEncoder(w).Encode(settings)
}

// Update godoc
// @Summary Update application settings
// @Description Update configuration settings category by category and emit change events
// @Tags settings
// @Accept json
// @Param request body model.Settings true "Structured settings object"
// @Success 200 "OK"
// @Failure 400 {string} string "Invalid Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /settings [patch]
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var newSettings model.Settings
	if !decodeAndValidate(w, r, &newSettings) {
		return
	}

	ctx := r.Context()
	oldSettings, err := h.repo.Get(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if oldSettings == nil {
		oldSettings = &model.Settings{}
	}

	// Detect changes
	changedFields := h.getChangedFields(*oldSettings, newSettings)

	// Save to DB
	if err := h.repo.Save(ctx, &newSettings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply settings to engines (Sync)
	h.engine.Configure(ctx, &newSettings)
	h.uploadEngine.Configure(ctx, &newSettings)

	// Emit event
	if len(changedFields) > 0 {
		h.bus.Publish(event.Event{
			Type:      event.SettingsUpdated,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"changes": changedFields,
			},
		})
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SettingsHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings, _ := h.repo.Get(ctx)
	remotes, _ := h.uploadEngine.ListRemotes(ctx)
	providers, _ := h.providerRepo.List(ctx)

	status := map[string]interface{}{
		"downloads": map[string]interface{}{
			"configured": settings != nil && settings.Download.DownloadDir != "",
		},
		"cloud": map[string]interface{}{
			"remoteCount": len(remotes),
		},
		"premium": map[string]interface{}{
			"providers": providers,
		},
	}

	json.NewEncoder(w).Encode(status)
}

func (h *SettingsHandler) Export(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=gravity-settings.json")
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) Import(w http.ResponseWriter, r *http.Request) {
	var settings model.Settings
	if !decodeAndValidate(w, r, &settings) {
		return
	}

	if err := h.repo.Save(r.Context(), &settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply
	h.engine.Configure(r.Context(), &settings)

	w.WriteHeader(http.StatusOK)
}

func (h *SettingsHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.DeleteAll(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save defaults back to DB
	defaults := model.DefaultSettings()
	if err := h.repo.Save(r.Context(), defaults); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SettingsHandler) getChangedFields(old, new model.Settings) []string {
	var changes []string

	valOld := reflect.ValueOf(old)
	valNew := reflect.ValueOf(new)
	typ := valOld.Type()

	for i := 0; i < valOld.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "UpdatedAt" || field.Name == "ID" {
			continue
		}

		if !reflect.DeepEqual(valOld.Field(i).Interface(), valNew.Field(i).Interface()) {
			changes = append(changes, field.Tag.Get("json"))
		}
	}

	return changes
}
