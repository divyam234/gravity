package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"gravity/internal/engine"
	"gravity/internal/model"
	"gravity/internal/store"

	"github.com/go-chi/chi/v5"
)

type SettingsHandler struct {
	repo         *store.SettingsRepo
	providerRepo *store.ProviderRepo
	engine       engine.DownloadEngine
	uploadEngine engine.UploadEngine
}

func NewSettingsHandler(repo *store.SettingsRepo, providerRepo *store.ProviderRepo, engine engine.DownloadEngine, uploadEngine engine.UploadEngine) *SettingsHandler {
	return &SettingsHandler{
		repo:         repo,
		providerRepo: providerRepo,
		engine:       engine,
		uploadEngine: uploadEngine,
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

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	flat, err := h.repo.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings := h.mapToModel(flat)
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) mapToModel(flat map[string]string) model.Settings {
	s := model.Settings{}

	// Download
	s.Download.DownloadDir = flat["download_dir"]
	s.Download.MaxConcurrentDownloads, _ = strconv.Atoi(flat["max_concurrent_downloads"])
	s.Download.MaxDownloadSpeed = flat["max_download_speed"]
	s.Download.MaxUploadSpeed = flat["max_upload_speed"]
	s.Download.MaxConnectionPerServer, _ = strconv.Atoi(flat["max_connection_per_server"])
	s.Download.Split, _ = strconv.Atoi(flat["split"])
	s.Download.UserAgent = flat["user_agent"]
	s.Download.ConnectTimeout, _ = strconv.Atoi(flat["connect_timeout"])
	s.Download.MaxTries, _ = strconv.Atoi(flat["max_tries"])
	s.Download.CheckCertificate, _ = strconv.ParseBool(flat["check_certificate"])

	// Upload
	s.Upload.DefaultRemote = flat["default_remote"]
	s.Upload.AutoUpload, _ = strconv.ParseBool(flat["auto_upload"])
	s.Upload.RemoveLocal, _ = strconv.ParseBool(flat["remove_local"])

	// Network
	s.Network.ProxyEnabled, _ = strconv.ParseBool(flat["proxy_enabled"])
	s.Network.ProxyUrl = flat["proxy_url"]
	s.Network.ProxyUser = flat["proxy_user"]
	s.Network.ProxyPassword = flat["proxy_password"]

	// Torrent
	s.Torrent.SeedRatio = flat["seed_ratio"]
	s.Torrent.SeedTime, _ = strconv.Atoi(flat["seed_time"])
	s.Torrent.ListenPort, _ = strconv.Atoi(flat["listen_port"])
	s.Torrent.ForceSave, _ = strconv.ParseBool(flat["force_save"])
	s.Torrent.EnablePex, _ = strconv.ParseBool(flat["enable_pex"])
	s.Torrent.EnableDht, _ = strconv.ParseBool(flat["enable_dht"])
	s.Torrent.EnableLpd, _ = strconv.ParseBool(flat["enable_lpd"])
	s.Torrent.Encryption = flat["bt_encryption"]

	// VFS
	s.Vfs.CacheMode = flat["vfs_cache_mode"]
	s.Vfs.CacheMaxSize = flat["vfs_cache_max_size"]
	s.Vfs.CacheMaxAge = flat["vfs_cache_max_age"]
	s.Vfs.WriteBack = flat["vfs_write_back"]
	s.Vfs.ReadChunkSize = flat["vfs_read_chunk_size"]
	s.Vfs.ReadChunkSizeLimit = flat["vfs_read_chunk_size_limit"]
	s.Vfs.ReadAhead = flat["vfs_read_ahead"]
	s.Vfs.DirCacheTime = flat["vfs_dir_cache_time"]
	s.Vfs.PollInterval = flat["vfs_poll_interval"]
	s.Vfs.ReadChunkStreams, _ = strconv.Atoi(flat["vfs_read_chunk_streams"])

	return s
}

func (h *SettingsHandler) mapFromModel(s model.Settings) map[string]string {
	flat := make(map[string]string)

	// Download
	flat["download_dir"] = s.Download.DownloadDir
	flat["max_concurrent_downloads"] = strconv.Itoa(s.Download.MaxConcurrentDownloads)
	flat["max_download_speed"] = s.Download.MaxDownloadSpeed
	flat["max_upload_speed"] = s.Download.MaxUploadSpeed
	flat["max_connection_per_server"] = strconv.Itoa(s.Download.MaxConnectionPerServer)
	flat["split"] = strconv.Itoa(s.Download.Split)
	flat["user_agent"] = s.Download.UserAgent
	flat["connect_timeout"] = strconv.Itoa(s.Download.ConnectTimeout)
	flat["max_tries"] = strconv.Itoa(s.Download.MaxTries)
	flat["check_certificate"] = strconv.FormatBool(s.Download.CheckCertificate)

	// Upload
	flat["default_remote"] = s.Upload.DefaultRemote
	flat["auto_upload"] = strconv.FormatBool(s.Upload.AutoUpload)
	flat["remove_local"] = strconv.FormatBool(s.Upload.RemoveLocal)

	// Network
	flat["proxy_enabled"] = strconv.FormatBool(s.Network.ProxyEnabled)
	flat["proxy_url"] = s.Network.ProxyUrl
	flat["proxy_user"] = s.Network.ProxyUser
	flat["proxy_password"] = s.Network.ProxyPassword

	// Torrent
	flat["seed_ratio"] = s.Torrent.SeedRatio
	flat["seed_time"] = strconv.Itoa(s.Torrent.SeedTime)
	flat["listen_port"] = strconv.Itoa(s.Torrent.ListenPort)
	flat["force_save"] = strconv.FormatBool(s.Torrent.ForceSave)
	flat["enable_pex"] = strconv.FormatBool(s.Torrent.EnablePex)
	flat["enable_dht"] = strconv.FormatBool(s.Torrent.EnableDht)
	flat["enable_lpd"] = strconv.FormatBool(s.Torrent.EnableLpd)
	flat["bt_encryption"] = s.Torrent.Encryption

	// VFS
	flat["vfs_cache_mode"] = s.Vfs.CacheMode
	flat["vfs_cache_max_size"] = s.Vfs.CacheMaxSize
	flat["vfs_cache_max_age"] = s.Vfs.CacheMaxAge
	flat["vfs_write_back"] = s.Vfs.WriteBack
	flat["vfs_read_chunk_size"] = s.Vfs.ReadChunkSize
	flat["vfs_read_chunk_size_limit"] = s.Vfs.ReadChunkSizeLimit
	flat["vfs_read_ahead"] = s.Vfs.ReadAhead
	flat["vfs_dir_cache_time"] = s.Vfs.DirCacheTime
	flat["vfs_poll_interval"] = s.Vfs.PollInterval
	flat["vfs_read_chunk_streams"] = strconv.Itoa(s.Vfs.ReadChunkStreams)

	return flat
}

func (h *SettingsHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings, err := h.repo.Get(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	remotes, _ := h.uploadEngine.ListRemotes(ctx)
	providers, _ := h.providerRepo.List(ctx)

	status := map[string]interface{}{
		"downloads": map[string]interface{}{
			"configured":   settings["download_dir"] != "",
			"downloadPath": settings["download_dir"],
		},
		"cloud": map[string]interface{}{
			"configured":         len(remotes) > 0,
			"remoteCount":        len(remotes),
			"defaultDestination": settings["default_remote"],
		},
		"premium": map[string]interface{}{
			"providers": providers,
		},
		"network": map[string]interface{}{
			"proxyEnabled": settings["proxy_enabled"] == "true",
		},
		"torrents": map[string]interface{}{
			"seedingEnabled": settings["seed_ratio"] != "0",
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
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := h.repo.SetMany(r.Context(), settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Re-apply to engine
	go h.engine.Configure(context.Background(), settings)

	w.WriteHeader(http.StatusOK)
}

func (h *SettingsHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.DeleteAll(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var s model.Settings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		// Fallback to flat map for partial updates or legacy support?
		// User specifically asked for refactoring, so let's try to support the new model.
		// If it's a flat map, the Decode might fail or result in empty object.
		// Let's support both if possible or just stick to the new model.
		http.Error(w, "invalid request: expected structured settings object", http.StatusBadRequest)
		return
	}

	flat := h.mapFromModel(s)

	if err := h.repo.SetMany(r.Context(), flat); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply settings to engine
	if err := h.engine.Configure(r.Context(), flat); err != nil {
		http.Error(w, "Saved but failed to apply to download engine: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.uploadEngine.Configure(r.Context(), flat); err != nil {
		http.Error(w, "Saved but failed to apply to upload engine: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

