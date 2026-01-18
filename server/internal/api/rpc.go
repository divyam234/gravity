package api

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"aria2-rclone-ui/internal/rclone"
	"aria2-rclone-ui/internal/store"
)

type RPCHandler struct {
	aria2URL     *url.URL
	rcloneURL    *url.URL
	rcloneClient *rclone.Client
	db           *store.DB
	aria2Proxy   *httputil.ReverseProxy
	rpcSecret    string
}

func NewRPCHandler(aria2Addr, rcloneAddr, rpcSecret string, db *store.DB) *RPCHandler {
	aURL, _ := url.Parse(aria2Addr)
	rURL, _ := url.Parse(rcloneAddr)
	proxy := httputil.NewSingleHostReverseProxy(aURL)

	// Modify outgoing request to prevent compressed responses from Aria2
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Remove Accept-Encoding to get plain text responses we can parse
		req.Header.Del("Accept-Encoding")
	}

	proxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Del("Access-Control-Allow-Origin")
		return nil
	}

	return &RPCHandler{
		aria2URL:     aURL,
		rcloneURL:    rURL,
		rcloneClient: rclone.NewClient(rcloneAddr),
		db:           db,
		aria2Proxy:   proxy,
		rpcSecret:    rpcSecret,
	}
}

type JsonRpcRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Id      interface{}   `json:"id"`
	Params  []interface{} `json:"params"`
}

type JsonRpcResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Id      interface{} `json:"id"`
}

func (h *RPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	// Restore body for proxy
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	var req JsonRpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// Might be batch, or invalid. For MVP, assume single request or fallback to proxy
		h.aria2Proxy.ServeHTTP(w, r)
		return
	}

	// Inject Secret if needed (for non-rclone requests)
	if h.rpcSecret != "" && !strings.HasPrefix(req.Method, "rclone.") && req.Method != "system.multicall" {
		hasToken := false
		if len(req.Params) > 0 {
			if s, ok := req.Params[0].(string); ok && strings.HasPrefix(s, "token:") {
				hasToken = true
			}
		}
		if !hasToken {
			req.Params = append([]interface{}{"token:" + h.rpcSecret}, req.Params...)
		}
	}

	// Re-serialize body for default proxy case if we modified it
	// But we only need to do this if we fall through to default.
	// However, handlers use 'req' which is passed by value copy if we are not careful?
	// ServeHTTP uses 'req'. We modified 'req.Params'. 'Params' is a slice (reference type), so modifications to the underlying array might be visible,
	// BUT 'append' might allocate a new array. So we must re-assign 'req.Params'.
	// We did re-assign req.Params above.
	// Since 'req' is local to this function, passing it to handlers works fine with the new params.

	// CRITICAL: We must update the request body for the PROXY fallthrough too!
	newBody, _ := json.Marshal(req)
	r.Body = io.NopCloser(bytes.NewBuffer(newBody))
	r.ContentLength = int64(len(newBody))

	if strings.HasPrefix(req.Method, "rclone.") {
		h.handleRcloneRequest(w, req)
		return
	}

	if req.Method == "aria2.addUri" {
		h.handleAddUri(w, r, req)
		return
	}

	if req.Method == "aria2.addTorrent" {
		h.handleAddTorrent(w, r, req)
		return
	}

	if req.Method == "aria2.addMetalink" {
		h.handleAddMetalink(w, r, req)
		return
	}

	if req.Method == "aria2.changeOption" {
		h.handleChangeOption(w, r, req)
		return
	}

	if req.Method == "aria2.remove" || req.Method == "aria2.forceRemove" {
		h.handleRemove(w, r, req)
		return
	}

	if req.Method == "aria2.tellStatus" {
		h.handleTellStatus(w, r, req)
		return
	}

	if req.Method == "aria2.getGlobalStat" {
		h.handleGetGlobalStat(w, r, req)
		return
	}

	// Default: forward to Aria2
	h.aria2Proxy.ServeHTTP(w, r)
}

func (h *RPCHandler) handleRcloneRequest(w http.ResponseWriter, req JsonRpcRequest) {
	method := strings.TrimPrefix(req.Method, "rclone.")

	rcloneMethod := method
	switch method {
	case "listRemotes":
		rcloneMethod = "config/listremotes"
	case "createRemote":
		rcloneMethod = "config/create"
	case "deleteRemote":
		rcloneMethod = "config/delete"
	case "getStats":
		rcloneMethod = "core/stats"
	case "getVersion":
		rcloneMethod = "core/version"
	}

	if strings.Contains(method, "/") {
		rcloneMethod = method
	}

	var params interface{}
	if len(req.Params) > 0 {
		params = req.Params[0]
	}

	res, err := h.rcloneClient.Call(rcloneMethod, params)
	h.writeResponse(w, req.Id, res, err)
}

func (h *RPCHandler) handleAddTorrent(w http.ResponseWriter, r *http.Request, req JsonRpcRequest) {
	if len(req.Params) < 2 {
		h.aria2Proxy.ServeHTTP(w, r)
		return
	}

	var torrent string
	var uris []string
	var options map[string]interface{}
	var optionsIdx int = -1

	if t, ok := req.Params[1].(string); ok {
		torrent = t
	}

	for i := 2; i < len(req.Params); i++ {
		p := req.Params[i]
		if uList, ok := p.([]interface{}); ok {
			for _, u := range uList {
				if s, ok := u.(string); ok {
					uris = append(uris, s)
				}
			}
		} else if m, ok := p.(map[string]interface{}); ok {
			options = m
			optionsIdx = i
		}
	}

	if options == nil {
		options = make(map[string]interface{})
	}

	var targetRemote string
	if val, ok := options["rclone-target"]; ok {
		targetRemote = val.(string)
		delete(options, "rclone-target")
	}

	gidBytes := make([]byte, 8)
	rand.Read(gidBytes)
	gid := hex.EncodeToString(gidBytes)
	options["gid"] = gid

	if optionsIdx != -1 {
		req.Params[optionsIdx] = options
	} else {
		req.Params = append(req.Params, options)
	}

	h.db.SaveUpload(store.UploadState{
		Gid:          gid,
		TargetRemote: targetRemote,
		Status:       "pending",
		StartedAt:    time.Now(),
		Torrent:      torrent,
		URIs:         uris,
		Options:      options,
	})
	log.Printf("Captured addTorrent for GID %s", gid)

	h.proxyWithModifiedBody(w, r, req)
}

func (h *RPCHandler) handleAddMetalink(w http.ResponseWriter, r *http.Request, req JsonRpcRequest) {
	if len(req.Params) < 2 {
		h.aria2Proxy.ServeHTTP(w, r)
		return
	}

	var metalink string
	var options map[string]interface{}
	var optionsIdx int = -1

	if m, ok := req.Params[1].(string); ok {
		metalink = m
	}

	if len(req.Params) > 2 {
		if m, ok := req.Params[2].(map[string]interface{}); ok {
			options = m
			optionsIdx = 2
		}
	}

	if options == nil {
		options = make(map[string]interface{})
	}

	var targetRemote string
	if val, ok := options["rclone-target"]; ok {
		targetRemote = val.(string)
		delete(options, "rclone-target")
	}

	gidBytes := make([]byte, 8)
	rand.Read(gidBytes)
	gid := hex.EncodeToString(gidBytes)
	options["gid"] = gid

	if optionsIdx != -1 {
		req.Params[optionsIdx] = options
	} else {
		req.Params = append(req.Params, options)
	}

	h.db.SaveUpload(store.UploadState{
		Gid:          gid,
		TargetRemote: targetRemote,
		Status:       "pending",
		StartedAt:    time.Now(),
		Metalink:     metalink,
		Options:      options,
	})
	log.Printf("Captured addMetalink for GID %s", gid)

	h.proxyWithModifiedBody(w, r, req)
}

func (h *RPCHandler) handleChangeOption(w http.ResponseWriter, r *http.Request, req JsonRpcRequest) {
	rec := NewResponseBuffer()
	h.aria2Proxy.ServeHTTP(rec, r)

	respBody, err := rec.Bytes()
	if err != nil {
		log.Printf("ChangeOption: Failed to read response: %v", err)
		rec.WriteRawTo(w)
		return
	}

	// Write response to client
	rec.CopyHeadersTo(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rec.Code)
	w.Write(respBody)

	// Update DB if successful
	var res JsonRpcResponse
	if err := json.Unmarshal(respBody, &res); err == nil && res.Error == nil {
		if len(req.Params) >= 3 {
			if gid, ok := req.Params[1].(string); ok {
				if opts, ok := req.Params[2].(map[string]interface{}); ok {
					h.db.UpdateOptions(gid, opts)
					log.Printf("Updated options for GID %s", gid)
				}
			}
		}
	}
}

func (h *RPCHandler) handleRemove(w http.ResponseWriter, r *http.Request, req JsonRpcRequest) {
	rec := NewResponseBuffer()
	h.aria2Proxy.ServeHTTP(rec, r)

	respBody, err := rec.Bytes()
	if err != nil {
		log.Printf("Remove: Failed to read response: %v", err)
		rec.WriteRawTo(w)
		return
	}

	// Write response to client
	rec.CopyHeadersTo(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rec.Code)
	w.Write(respBody)

	// Cleanup if successful
	var res JsonRpcResponse
	if err := json.Unmarshal(respBody, &res); err == nil && res.Error == nil {
		if len(req.Params) >= 2 {
			if gid, ok := req.Params[1].(string); ok {
				// Stop active upload if any
				if upload, err := h.db.GetUpload(gid); err == nil && upload.Status == "uploading" && upload.JobID != "" {
					log.Printf("Stopping Rclone Job %s for removed task %s", upload.JobID, gid)
					h.rcloneClient.StopJob(upload.JobID)
				}
				// Mark as removed
				h.db.UpdateStatus(gid, "removed", "")
				log.Printf("Marked GID %s as removed", gid)
			}
		}
	}
}

func (h *RPCHandler) proxyWithModifiedBody(w http.ResponseWriter, r *http.Request, req JsonRpcRequest) {
	newBody, _ := json.Marshal(req)
	r.Body = io.NopCloser(bytes.NewBuffer(newBody))
	r.ContentLength = int64(len(newBody))
	h.aria2Proxy.ServeHTTP(w, r)
}

func (h *RPCHandler) handleAddUri(w http.ResponseWriter, r *http.Request, req JsonRpcRequest) {
	var options map[string]interface{}
	var uris []string
	var optionsIdx int = -1

	for i, p := range req.Params {
		if m, ok := p.(map[string]interface{}); ok {
			options = m
			optionsIdx = i
		} else if u, ok := p.([]interface{}); ok {
			for _, uri := range u {
				if s, ok := uri.(string); ok {
					uris = append(uris, s)
				}
			}
		}
	}

	if options == nil {
		options = make(map[string]interface{})
	}

	var targetRemote string
	if val, ok := options["rclone-target"]; ok {
		targetRemote = val.(string)
		delete(options, "rclone-target")
	}

	gidBytes := make([]byte, 8)
	if _, err := rand.Read(gidBytes); err != nil {
		log.Printf("Failed to generate GID: %v", err)
		h.aria2Proxy.ServeHTTP(w, r)
		return
	}
	gid := hex.EncodeToString(gidBytes)
	options["gid"] = gid

	if optionsIdx != -1 {
		req.Params[optionsIdx] = options
	} else {
		req.Params = append(req.Params, options)
	}

	h.db.SaveUpload(store.UploadState{
		Gid:          gid,
		TargetRemote: targetRemote,
		Status:       "pending",
		StartedAt:    time.Now(),
		URIs:         uris,
		Options:      options,
	})
	log.Printf("Session saved for GID %s (Remote: %s)", gid, targetRemote)

	h.proxyWithModifiedBody(w, r, req)
}

func (h *RPCHandler) handleTellStatus(w http.ResponseWriter, r *http.Request, req JsonRpcRequest) {
	rec := NewResponseBuffer()
	h.aria2Proxy.ServeHTTP(rec, r)

	respBody, err := rec.Bytes()
	if err != nil {
		log.Printf("TellStatus: Failed to read response: %v", err)
		rec.WriteRawTo(w)
		return
	}

	var res JsonRpcResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		log.Printf("TellStatus: Failed to unmarshal (len=%d): %v", len(respBody), err)
		rec.WriteRawTo(w)
		return
	}

	// Augment with rclone state if available
	if res.Result != nil {
		if task, ok := res.Result.(map[string]interface{}); ok {
			if gidVal, ok := task["gid"]; ok {
				if gid, ok := gidVal.(string); ok {
					upload, err := h.db.GetUpload(gid)
					if err == nil {
						rcloneState := map[string]interface{}{
							"status":       upload.Status,
							"targetRemote": upload.TargetRemote,
						}
						if upload.JobID != "" {
							rcloneState["jobId"] = upload.JobID

							// If uploading, get live stats from Rclone
							if upload.Status == "uploading" {
								if jobStats, err := h.rcloneClient.Call("core/stats", nil); err == nil {
									if speed, ok := jobStats["speed"].(float64); ok {
										rcloneState["speed"] = speed
									}
									if bytes, ok := jobStats["bytes"].(float64); ok {
										rcloneState["bytes"] = bytes
									}
									if totalBytes, ok := jobStats["totalBytes"].(float64); ok {
										rcloneState["totalBytes"] = totalBytes
									}
									if eta, ok := jobStats["eta"].(float64); ok {
										rcloneState["eta"] = eta
									}
									// Get transfer progress if available
									if transfers, ok := jobStats["transferring"].([]interface{}); ok && len(transfers) > 0 {
										if t, ok := transfers[0].(map[string]interface{}); ok {
											if pct, ok := t["percentage"].(float64); ok {
												rcloneState["percentage"] = pct
											}
										}
									}
								}
							}
						}
						task["rclone"] = rcloneState
					}
				}
			}
		}
	}

	h.writeJSON(w, res)
}

func (h *RPCHandler) handleGetGlobalStat(w http.ResponseWriter, r *http.Request, req JsonRpcRequest) {
	rec := NewResponseBuffer()
	h.aria2Proxy.ServeHTTP(rec, r)

	respBody, err := rec.Bytes()
	if err != nil {
		rec.WriteRawTo(w)
		return
	}

	var res JsonRpcResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		rec.WriteRawTo(w)
		return
	}

	// Augment with Rclone stats and DB stats
	if res.Result != nil {
		if stats, ok := res.Result.(map[string]interface{}); ok {
			// Get Rclone stats
			rcloneStats, err := h.rcloneClient.Call("core/stats", nil)
			if err == nil {
				// Add cloud upload speed
				if speed, ok := rcloneStats["speed"].(float64); ok {
					stats["cloudUploadSpeed"] = fmt.Sprintf("%.0f", speed)
				}
				// Add number of active transfers
				if transfers, ok := rcloneStats["transfers"].(float64); ok {
					stats["numUploading"] = fmt.Sprintf("%.0f", transfers)
				}
			}

			// Get uploading count from DB
			uploadingCount, err := h.db.GetUploadingCount()
			if err == nil && uploadingCount > 0 {
				stats["numUploading"] = fmt.Sprintf("%d", uploadingCount)
			}

			// Get persistent stats from DB
			dbStats, err := h.db.GetGlobalStats()
			if err == nil && dbStats != nil {
				stats["totalDownloaded"] = fmt.Sprintf("%d", dbStats.TotalDownloaded)
				stats["totalUploaded"] = fmt.Sprintf("%d", dbStats.TotalUploaded)
				stats["totalTasks"] = fmt.Sprintf("%d", dbStats.TotalTasks)
				stats["completedTasks"] = fmt.Sprintf("%d", dbStats.CompletedTasks)
				stats["uploadedTasks"] = fmt.Sprintf("%d", dbStats.UploadedTasks)
			}
		}
	}

	h.writeJSON(w, res)
}

func (h *RPCHandler) writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (h *RPCHandler) writeResponse(w http.ResponseWriter, id interface{}, result interface{}, err error) {
	resp := JsonRpcResponse{
		JsonRPC: "2.0",
		Id:      id,
		Result:  result,
	}
	if err != nil {
		resp.Error = map[string]interface{}{
			"code":    -32000,
			"message": err.Error(),
		}
	}
	h.writeJSON(w, resp)
}

// ResponseBuffer captures HTTP responses for interception/modification.
// Custom implementation to avoid httptest dependency.
type ResponseBuffer struct {
	Code      int
	HeaderMap http.Header
	Body      *bytes.Buffer
}

func NewResponseBuffer() *ResponseBuffer {
	return &ResponseBuffer{
		Code:      http.StatusOK,
		HeaderMap: make(http.Header),
		Body:      new(bytes.Buffer),
	}
}

func (rb *ResponseBuffer) Header() http.Header {
	return rb.HeaderMap
}

func (rb *ResponseBuffer) Write(b []byte) (int, error) {
	return rb.Body.Write(b)
}

func (rb *ResponseBuffer) WriteHeader(statusCode int) {
	rb.Code = statusCode
}

// Bytes returns the response body, automatically decompressing gzip if needed.
func (rb *ResponseBuffer) Bytes() ([]byte, error) {
	raw := rb.Body.Bytes()
	if len(raw) == 0 {
		return raw, nil
	}

	// Check Content-Encoding header
	if strings.EqualFold(rb.HeaderMap.Get("Content-Encoding"), "gzip") {
		return decompressGzip(raw)
	}

	// Also detect gzip by magic bytes (0x1f 0x8b) as fallback
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		return decompressGzip(raw)
	}

	return raw, nil
}

// decompressGzip decompresses gzip-encoded data.
func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader error: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("gzip decompress error: %w", err)
	}
	return decompressed, nil
}

// CopyHeadersTo copies headers from the buffer to a ResponseWriter,
// excluding Content-Encoding and Content-Length (since we may modify the body).
func (rb *ResponseBuffer) CopyHeadersTo(w http.ResponseWriter) {
	for k, v := range rb.HeaderMap {
		// Skip encoding/length headers - we handle these ourselves
		if strings.EqualFold(k, "Content-Encoding") || strings.EqualFold(k, "Content-Length") {
			continue
		}
		w.Header()[k] = v
	}
}

// WriteRawTo writes the original (possibly compressed) response to w.
func (rb *ResponseBuffer) WriteRawTo(w http.ResponseWriter) {
	for k, v := range rb.HeaderMap {
		w.Header()[k] = v
	}
	w.WriteHeader(rb.Code)
	w.Write(rb.Body.Bytes())
}
