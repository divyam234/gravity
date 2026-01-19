# Magnet File Selection Implementation Plan

## Overview

Add support for **magnet links with file selection** - allowing users to choose which files to download from a torrent. This works for both:

1. **AllDebrid cached magnets** - Fast download via direct HTTP links
2. **Raw magnets** - Native aria2c BitTorrent with `--select-file`

The UI is unified - same FileTree component for both sources.

---

## User Flow

```
1. User pastes magnet link in Add Download page
   ↓
2. Frontend detects "magnet:" prefix
   ↓
3. POST /api/v1/magnets/check { magnet: "..." }
   ↓
4. Backend checks sources:
   ├── AllDebrid configured & cached? → Return files from AllDebrid API
   └── Otherwise → Fetch metadata via aria2c
   ↓
5. Frontend displays FileTree with checkboxes
   ↓
6. User selects files, clicks "Start"
   ↓
7. POST /api/v1/magnets/download { magnet, selectedFiles, source, destination }
   ↓
8. Backend creates ONE Download record with multiple files
   ↓
9. Downloads run in parallel:
   ├── AllDebrid: aria2c downloads each direct HTTP link
   └── Raw: aria2c with --select-file=1,3,5
   ↓
10. Progress tracked cumulatively (total downloaded / total size)
   ↓
11. Task detail shows per-file progress
   ↓
12. On complete: Optional upload to cloud via rclone
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend                                 │
├─────────────────────────────────────────────────────────────────┤
│  Add Download Page                                               │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  URL Input: magnet:?xt=urn:btih:abc123...               │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  FileTree Component (React Aria Tree)                    │    │
│  │  ☑ ubuntu-24.04/                          5.3 GB        │    │
│  │    ☑ ubuntu-24.04-desktop.iso             4.5 GB        │    │
│  │    ☑ ubuntu-24.04-server.iso              800 MB        │    │
│  │    ☐ extras/                              12 MB         │    │
│  │      ☐ wallpapers.zip                     10 MB         │    │
│  │      ☐ readme.txt                         2 MB          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  [Select All] [Deselect All]    Selected: 2 files (5.3 GB)      │
│                                                                  │
│  Upload Target: [gdrive:/downloads____________]                  │
│                                                                  │
│                              [Cancel] [Start Download]           │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                      Backend API                                 │
├─────────────────────────────────────────────────────────────────┤
│  POST /api/v1/magnets/check                                      │
│  POST /api/v1/magnets/download                                   │
│  GET  /api/v1/downloads/{id}/files                               │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Magnet Resolvers                              │
├─────────────────────────────────────────────────────────────────┤
│  AllDebridResolver          │  Aria2Resolver                     │
│  - /magnet/upload           │  - aria2c --bt-metadata-only       │
│  - /magnet/files            │  - aria2c --select-file            │
│  - Direct HTTP downloads    │  - BitTorrent P2P                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Data Models

### Backend (Go)

#### Download Model Updates

**File:** `server/internal/model/download.go`

```go
type Download struct {
    ID             string         `json:"id"`
    URL            string         `json:"url"`
    ResolvedURL    string         `json:"resolvedUrl,omitempty"`
    Provider       string         `json:"provider,omitempty"`
    Status         DownloadStatus `json:"status"`
    Error          string         `json:"error,omitempty"`
    Filename       string         `json:"filename,omitempty"`
    LocalPath      string         `json:"localPath,omitempty"`
    Size           int64          `json:"size"`
    Downloaded     int64          `json:"downloaded"`
    Speed          int64          `json:"speed"`
    ETA            int            `json:"eta"`
    Destination    string         `json:"destination,omitempty"`
    UploadStatus   string         `json:"uploadStatus,omitempty"`
    UploadProgress int            `json:"uploadProgress"`
    UploadSpeed    int64          `json:"uploadSpeed"`
    Category       string         `json:"category,omitempty"`
    Tags           []string       `json:"tags,omitempty"`
    EngineID       string         `json:"-"`
    UploadJobID    string         `json:"-"`
    CreatedAt      time.Time      `json:"createdAt"`
    StartedAt      *time.Time     `json:"startedAt,omitempty"`
    CompletedAt    *time.Time     `json:"completedAt,omitempty"`
    UpdatedAt      time.Time      `json:"updatedAt"`

    // NEW: Multi-file support for magnets/torrents
    IsMagnet      bool           `json:"isMagnet,omitempty"`
    MagnetHash    string         `json:"magnetHash,omitempty"`
    MagnetSource  string         `json:"magnetSource,omitempty"`  // "alldebrid" or "aria2"
    MagnetID      string         `json:"-"`                       // AllDebrid magnet ID
    Files         []DownloadFile `json:"files,omitempty"`
    TotalFiles    int            `json:"totalFiles,omitempty"`
    FilesComplete int            `json:"filesComplete,omitempty"`
}

type DownloadFile struct {
    ID         string         `json:"id"`
    Name       string         `json:"name"`
    Path       string         `json:"path"`       // relative path: "ubuntu/iso/file.iso"
    Size       int64          `json:"size"`
    Downloaded int64          `json:"downloaded"`
    Progress   int            `json:"progress"`   // 0-100
    Status     DownloadStatus `json:"status"`     // pending, active, complete, error
    Error      string         `json:"error,omitempty"`
    EngineID   string         `json:"-"`          // aria2c GID for this file
    URL        string         `json:"-"`          // Download URL (AllDebrid direct link)
    Index      int            `json:"-"`          // File index for aria2c --select-file
}
```

#### Magnet Types

**File:** `server/internal/model/magnet.go` (new)

```go
package model

type MagnetInfo struct {
    Source   string       `json:"source"`   // "alldebrid" or "aria2"
    Cached   bool         `json:"cached"`   // true if AllDebrid cached
    MagnetID string       `json:"magnetId,omitempty"`
    Name     string       `json:"name"`
    Hash     string       `json:"hash"`
    Size     int64        `json:"size"`
    Files    []MagnetFile `json:"files"`
}

type MagnetFile struct {
    ID       string       `json:"id"`       // AllDebrid file ID or aria2c index
    Name     string       `json:"name"`
    Path     string       `json:"path"`     // full path for display
    Size     int64        `json:"size"`
    Link     string       `json:"link,omitempty"` // download link (AllDebrid only)
    IsFolder bool         `json:"isFolder"`
    Children []MagnetFile `json:"children,omitempty"`
}
```

### Frontend (TypeScript)

**File:** `frontend/src/lib/types.ts`

```typescript
export interface Download {
  id: string;
  url: string;
  resolvedUrl?: string;
  provider?: string;
  status: 'active' | 'waiting' | 'paused' | 'uploading' | 'complete' | 'error';
  error?: string;
  filename?: string;
  size: number;
  downloaded: number;
  speed: number;
  eta: number;
  destination?: string;
  uploadStatus?: string;
  uploadProgress: number;
  uploadSpeed: number;
  category?: string;
  tags?: string[];
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  updatedAt: string;

  // Multi-file support
  isMagnet?: boolean;
  magnetHash?: string;
  magnetSource?: 'alldebrid' | 'aria2';
  files?: DownloadFile[];
  totalFiles?: number;
  filesComplete?: number;
}

export interface DownloadFile {
  id: string;
  name: string;
  path: string;
  size: number;
  downloaded: number;
  progress: number;
  status: 'pending' | 'active' | 'complete' | 'error';
  error?: string;
}

export interface MagnetInfo {
  source: 'alldebrid' | 'aria2';
  cached: boolean;
  magnetId?: string;
  name: string;
  hash: string;
  size: number;
  files: MagnetFile[];
}

export interface MagnetFile {
  id: string;
  name: string;
  path: string;
  size: number;
  isFolder: boolean;
  children?: MagnetFile[];
}

export interface MagnetCheckRequest {
  magnet: string;
}

export interface MagnetDownloadRequest {
  magnet: string;
  source: 'alldebrid' | 'aria2';
  magnetId?: string;
  selectedFiles: string[];
  destination?: string;
}
```

---

## Database Schema

**File:** `server/internal/store/migrations/003_magnet_files.sql`

```sql
-- Add magnet fields to downloads table
ALTER TABLE downloads ADD COLUMN is_magnet BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE downloads ADD COLUMN magnet_hash TEXT;
ALTER TABLE downloads ADD COLUMN magnet_source TEXT;
ALTER TABLE downloads ADD COLUMN magnet_id TEXT;
ALTER TABLE downloads ADD COLUMN total_files INTEGER NOT NULL DEFAULT 0;
ALTER TABLE downloads ADD COLUMN files_complete INTEGER NOT NULL DEFAULT 0;

-- Create download_files table for individual file tracking
CREATE TABLE IF NOT EXISTS download_files (
    id TEXT PRIMARY KEY,
    download_id TEXT NOT NULL REFERENCES downloads(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    downloaded INTEGER NOT NULL DEFAULT 0,
    progress INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT,
    engine_id TEXT,
    url TEXT,
    file_index INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_download_files_download_id ON download_files(download_id);
CREATE INDEX IF NOT EXISTS idx_download_files_engine_id ON download_files(engine_id);
```

---

## API Endpoints

### POST /api/v1/magnets/check

Check if a magnet is available and get file list.

**Request:**
```json
{
  "magnet": "magnet:?xt=urn:btih:abc123..."
}
```

**Response (AllDebrid cached):**
```json
{
  "source": "alldebrid",
  "cached": true,
  "magnetId": "123456",
  "name": "Ubuntu 24.04 Desktop",
  "hash": "abc123def456...",
  "size": 5665497088,
  "files": [
    {
      "id": "f1",
      "name": "ubuntu-24.04-desktop-amd64.iso",
      "path": "Ubuntu 24.04 Desktop/ubuntu-24.04-desktop-amd64.iso",
      "size": 5600000000,
      "isFolder": false
    },
    {
      "id": "f2",
      "name": "readme.txt",
      "path": "Ubuntu 24.04 Desktop/readme.txt",
      "size": 1024,
      "isFolder": false
    }
  ]
}
```

**Response (raw magnet via aria2c):**
```json
{
  "source": "aria2",
  "cached": false,
  "name": "Ubuntu 24.04 Desktop",
  "hash": "abc123def456...",
  "size": 5665497088,
  "files": [
    {
      "id": "1",
      "name": "ubuntu-24.04-desktop-amd64.iso",
      "path": "Ubuntu 24.04 Desktop/ubuntu-24.04-desktop-amd64.iso",
      "size": 5600000000,
      "isFolder": false
    },
    {
      "id": "2",
      "name": "readme.txt",
      "path": "Ubuntu 24.04 Desktop/readme.txt",
      "size": 1024,
      "isFolder": false
    }
  ]
}
```

**Response (not cached, AllDebrid configured but magnet not available):**
```json
{
  "source": "aria2",
  "cached": false,
  "name": "...",
  "files": [...]
}
```

### POST /api/v1/magnets/download

Start download of selected files.

**Request:**
```json
{
  "magnet": "magnet:?xt=urn:btih:abc123...",
  "source": "alldebrid",
  "magnetId": "123456",
  "selectedFiles": ["f1", "f2"],
  "destination": "gdrive:/downloads"
}
```

**Response:**
```json
{
  "id": "d_abc123",
  "url": "magnet:?xt=urn:btih:abc123...",
  "filename": "Ubuntu 24.04 Desktop",
  "status": "active",
  "size": 5600001024,
  "downloaded": 0,
  "speed": 0,
  "isMagnet": true,
  "magnetSource": "alldebrid",
  "totalFiles": 2,
  "filesComplete": 0,
  "files": [
    {
      "id": "f1",
      "name": "ubuntu-24.04-desktop-amd64.iso",
      "path": "Ubuntu 24.04 Desktop/ubuntu-24.04-desktop-amd64.iso",
      "size": 5600000000,
      "downloaded": 0,
      "progress": 0,
      "status": "pending"
    },
    {
      "id": "f2",
      "name": "readme.txt",
      "path": "Ubuntu 24.04 Desktop/readme.txt",
      "size": 1024,
      "downloaded": 0,
      "progress": 0,
      "status": "pending"
    }
  ],
  "createdAt": "2026-01-20T12:00:00Z",
  "updatedAt": "2026-01-20T12:00:00Z"
}
```

### GET /api/v1/downloads/{id}/files

Get detailed file progress for a download.

**Response:**
```json
{
  "files": [
    {
      "id": "f1",
      "name": "ubuntu-24.04-desktop-amd64.iso",
      "path": "Ubuntu 24.04 Desktop/ubuntu-24.04-desktop-amd64.iso",
      "size": 5600000000,
      "downloaded": 2800000000,
      "progress": 50,
      "status": "active"
    },
    {
      "id": "f2",
      "name": "readme.txt",
      "path": "Ubuntu 24.04 Desktop/readme.txt",
      "size": 1024,
      "downloaded": 1024,
      "progress": 100,
      "status": "complete"
    }
  ]
}
```

---

## Provider Interface

**File:** `server/internal/provider/provider.go`

```go
// MagnetProvider is implemented by providers that support magnet links
type MagnetProvider interface {
    Provider
    
    // CheckMagnet checks if a magnet is available and returns file list
    // Returns nil, nil if magnet is not cached (for debrid providers)
    CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error)
    
    // GetMagnetFiles returns file tree for a magnet (by ID for debrid, by hash for aria2)
    GetMagnetFiles(ctx context.Context, magnetID string) ([]model.MagnetFile, error)
    
    // DeleteMagnet removes a magnet from user's account (debrid only)
    DeleteMagnet(ctx context.Context, magnetID string) error
}
```

---

## AllDebrid Implementation

**File:** `server/internal/provider/alldebrid/magnet.go` (new)

```go
package alldebrid

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"

    "gravity/internal/model"
)

// CheckMagnet uploads magnet to AllDebrid and checks if cached
func (p *AllDebridProvider) CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
    // POST /v4/magnet/upload
    endpoint := fmt.Sprintf("%s/magnet/upload?agent=%s&apikey=%s", baseURL, agent, p.apiKey)
    
    form := url.Values{}
    form.Set("magnets[]", magnet)
    
    req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Status string `json:"status"`
        Data   struct {
            Magnets []struct {
                Magnet string `json:"magnet"`
                Hash   string `json:"hash"`
                Name   string `json:"name"`
                Size   int64  `json:"size"`
                Ready  bool   `json:"ready"`
                ID     int64  `json:"id"`
                Error  *struct {
                    Code    string `json:"code"`
                    Message string `json:"message"`
                } `json:"error,omitempty"`
            } `json:"magnets"`
        } `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if result.Status != "success" || len(result.Data.Magnets) == 0 {
        return nil, fmt.Errorf("failed to check magnet")
    }
    
    m := result.Data.Magnets[0]
    if m.Error != nil {
        return nil, fmt.Errorf("alldebrid error: %s", m.Error.Message)
    }
    
    if !m.Ready {
        // Not cached - caller should fall back to aria2
        return nil, nil
    }
    
    // Get files
    files, err := p.GetMagnetFiles(ctx, fmt.Sprintf("%d", m.ID))
    if err != nil {
        return nil, err
    }
    
    return &model.MagnetInfo{
        Source:   "alldebrid",
        Cached:   true,
        MagnetID: fmt.Sprintf("%d", m.ID),
        Name:     m.Name,
        Hash:     m.Hash,
        Size:     m.Size,
        Files:    files,
    }, nil
}

// GetMagnetFiles retrieves file tree for a magnet
func (p *AllDebridProvider) GetMagnetFiles(ctx context.Context, magnetID string) ([]model.MagnetFile, error) {
    // POST /v4/magnet/files
    endpoint := fmt.Sprintf("%s/magnet/files?agent=%s&apikey=%s", baseURL, agent, p.apiKey)
    
    form := url.Values{}
    form.Set("id[]", magnetID)
    
    req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Status string `json:"status"`
        Data   struct {
            Magnets []struct {
                ID    string            `json:"id"`
                Files []json.RawMessage `json:"files"`
            } `json:"magnets"`
        } `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if result.Status != "success" || len(result.Data.Magnets) == 0 {
        return nil, fmt.Errorf("failed to get magnet files")
    }
    
    // Parse AllDebrid's nested file format
    return p.parseFiles(result.Data.Magnets[0].Files, "")
}

// parseFiles converts AllDebrid's nested format to flat MagnetFile array
func (p *AllDebridProvider) parseFiles(rawFiles []json.RawMessage, parentPath string) ([]model.MagnetFile, error) {
    var files []model.MagnetFile
    
    for i, raw := range rawFiles {
        var node struct {
            N string            `json:"n"` // name
            S int64             `json:"s"` // size (file only)
            L string            `json:"l"` // link (file only)
            E []json.RawMessage `json:"e"` // children (folder only)
        }
        
        if err := json.Unmarshal(raw, &node); err != nil {
            continue
        }
        
        path := node.N
        if parentPath != "" {
            path = parentPath + "/" + node.N
        }
        
        if len(node.E) > 0 {
            // Folder
            children, err := p.parseFiles(node.E, path)
            if err != nil {
                continue
            }
            
            // Calculate folder size
            var folderSize int64
            for _, child := range children {
                folderSize += child.Size
            }
            
            files = append(files, model.MagnetFile{
                ID:       fmt.Sprintf("folder_%d_%s", i, path),
                Name:     node.N,
                Path:     path,
                Size:     folderSize,
                IsFolder: true,
                Children: children,
            })
        } else {
            // File
            files = append(files, model.MagnetFile{
                ID:       node.L, // Use link as ID for easy lookup
                Name:     node.N,
                Path:     path,
                Size:     node.S,
                Link:     node.L,
                IsFolder: false,
            })
        }
    }
    
    return files, nil
}

// DeleteMagnet removes a magnet from user's AllDebrid account
func (p *AllDebridProvider) DeleteMagnet(ctx context.Context, magnetID string) error {
    endpoint := fmt.Sprintf("%s/magnet/delete?agent=%s&apikey=%s", baseURL, agent, p.apiKey)
    
    form := url.Values{}
    form.Set("id", magnetID)
    
    req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

---

## aria2c Raw Magnet Implementation

**File:** `server/internal/engine/aria2/magnet.go` (new)

```go
package aria2

import (
    "context"
    "fmt"
    "strings"
    "time"

    "gravity/internal/model"
)

// GetMagnetFiles fetches metadata for a magnet and returns file list
func (e *Engine) GetMagnetFiles(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
    // Add magnet with metadata-only mode
    gid, err := e.rpc.Call("aria2.addUri", []interface{}{
        []string{magnet},
        map[string]interface{}{
            "bt-metadata-only": "true",
            "bt-save-metadata": "true",
        },
    })
    if err != nil {
        return nil, err
    }
    
    gidStr := gid.(string)
    defer e.rpc.Call("aria2.remove", gidStr) // Clean up after getting metadata
    
    // Wait for metadata to be fetched (poll status)
    var info *model.MagnetInfo
    for i := 0; i < 60; i++ { // Max 60 seconds
        status, err := e.rpc.Call("aria2.tellStatus", gidStr)
        if err != nil {
            return nil, err
        }
        
        statusMap := status.(map[string]interface{})
        
        // Check if we have bittorrent info
        if btInfo, ok := statusMap["bittorrent"].(map[string]interface{}); ok {
            if btInfoInner, ok := btInfo["info"].(map[string]interface{}); ok {
                name := btInfoInner["name"].(string)
                
                // Get files
                filesResp, err := e.rpc.Call("aria2.getFiles", gidStr)
                if err != nil {
                    return nil, err
                }
                
                files := filesResp.([]interface{})
                var magnetFiles []model.MagnetFile
                var totalSize int64
                
                for _, f := range files {
                    fileMap := f.(map[string]interface{})
                    index := fileMap["index"].(string)
                    path := fileMap["path"].(string)
                    size := parseSize(fileMap["length"])
                    
                    // Extract relative path (remove download dir prefix)
                    relPath := path
                    if idx := strings.Index(path, name); idx >= 0 {
                        relPath = path[idx:]
                    }
                    
                    magnetFiles = append(magnetFiles, model.MagnetFile{
                        ID:       index, // 1-indexed for --select-file
                        Name:     extractFilename(relPath),
                        Path:     relPath,
                        Size:     size,
                        IsFolder: false,
                    })
                    totalSize += size
                }
                
                // Extract hash from magnet
                hash := extractHashFromMagnet(magnet)
                
                info = &model.MagnetInfo{
                    Source: "aria2",
                    Cached: false,
                    Name:   name,
                    Hash:   hash,
                    Size:   totalSize,
                    Files:  magnetFiles,
                }
                break
            }
        }
        
        // Check for errors
        if statusMap["status"] == "error" {
            return nil, fmt.Errorf("failed to fetch magnet metadata")
        }
        
        time.Sleep(1 * time.Second)
    }
    
    if info == nil {
        return nil, fmt.Errorf("timeout fetching magnet metadata")
    }
    
    return info, nil
}

// AddMagnetWithSelection starts a magnet download with selected files only
func (e *Engine) AddMagnetWithSelection(ctx context.Context, magnet string, selectedIndexes []string, opts DownloadOptions) (string, error) {
    // Build select-file string (comma-separated 1-indexed)
    selectFile := strings.Join(selectedIndexes, ",")
    
    options := map[string]interface{}{
        "select-file": selectFile,
    }
    
    if opts.Dir != "" {
        options["dir"] = opts.Dir
    }
    
    result, err := e.rpc.Call("aria2.addUri", []interface{}{
        []string{magnet},
        options,
    })
    if err != nil {
        return "", err
    }
    
    return result.(string), nil
}

func extractHashFromMagnet(magnet string) string {
    // Extract btih hash from magnet URI
    if idx := strings.Index(magnet, "btih:"); idx >= 0 {
        hash := magnet[idx+5:]
        if ampIdx := strings.Index(hash, "&"); ampIdx >= 0 {
            hash = hash[:ampIdx]
        }
        return strings.ToLower(hash)
    }
    return ""
}

func extractFilename(path string) string {
    parts := strings.Split(path, "/")
    return parts[len(parts)-1]
}

func parseSize(v interface{}) int64 {
    switch val := v.(type) {
    case string:
        var size int64
        fmt.Sscanf(val, "%d", &size)
        return size
    case float64:
        return int64(val)
    case int64:
        return val
    }
    return 0
}
```

---

## Magnet Service

**File:** `server/internal/service/magnet.go` (new)

```go
package service

import (
    "context"
    "fmt"
    "time"

    "gravity/internal/engine"
    "gravity/internal/engine/aria2"
    "gravity/internal/model"
    "gravity/internal/provider/alldebrid"
    "gravity/internal/store"

    "github.com/google/uuid"
)

type MagnetService struct {
    downloadRepo *store.DownloadRepo
    aria2Engine  *aria2.Engine
    allDebrid    *alldebrid.AllDebridProvider
    uploadEngine engine.UploadEngine
}

func NewMagnetService(
    repo *store.DownloadRepo,
    aria2 *aria2.Engine,
    allDebrid *alldebrid.AllDebridProvider,
    uploadEngine engine.UploadEngine,
) *MagnetService {
    return &MagnetService{
        downloadRepo: repo,
        aria2Engine:  aria2,
        allDebrid:    allDebrid,
        uploadEngine: uploadEngine,
    }
}

// CheckMagnet checks if a magnet is available and returns file list
func (s *MagnetService) CheckMagnet(ctx context.Context, magnet string) (*model.MagnetInfo, error) {
    // 1. Try AllDebrid first (if configured)
    if s.allDebrid != nil && s.allDebrid.IsConfigured() {
        info, err := s.allDebrid.CheckMagnet(ctx, magnet)
        if err == nil && info != nil && info.Cached {
            return info, nil
        }
        // Not cached or error - fall through to aria2
    }
    
    // 2. Fall back to raw magnet via aria2
    info, err := s.aria2Engine.GetMagnetFiles(ctx, magnet)
    if err != nil {
        return nil, fmt.Errorf("failed to get magnet files: %w", err)
    }
    
    return info, nil
}

// DownloadMagnet starts download of selected files from a magnet
func (s *MagnetService) DownloadMagnet(ctx context.Context, req MagnetDownloadRequest) (*model.Download, error) {
    // Create download record
    d := &model.Download{
        ID:           "d_" + uuid.New().String()[:8],
        URL:          req.Magnet,
        Status:       model.StatusActive,
        Destination:  req.Destination,
        IsMagnet:     true,
        MagnetSource: req.Source,
        MagnetID:     req.MagnetID,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    
    // Build file list and calculate total size
    var totalSize int64
    for _, fileID := range req.SelectedFiles {
        file := req.FindFile(fileID)
        if file == nil {
            continue
        }
        
        d.Files = append(d.Files, model.DownloadFile{
            ID:     fileID,
            Name:   file.Name,
            Path:   file.Path,
            Size:   file.Size,
            Status: model.StatusWaiting,
            URL:    file.Link, // Only for AllDebrid
            Index:  file.Index, // Only for aria2
        })
        totalSize += file.Size
    }
    
    d.Size = totalSize
    d.TotalFiles = len(d.Files)
    d.Filename = req.Name // Torrent name
    
    // Save to database
    if err := s.downloadRepo.CreateWithFiles(ctx, d); err != nil {
        return nil, err
    }
    
    // Start downloads based on source
    if req.Source == "alldebrid" {
        go s.startAllDebridDownload(ctx, d)
    } else {
        go s.startAria2Download(ctx, d, req.Magnet)
    }
    
    return d, nil
}

// startAllDebridDownload downloads files via AllDebrid direct links
func (s *MagnetService) startAllDebridDownload(ctx context.Context, d *model.Download) {
    // Download each file in parallel via aria2
    for i := range d.Files {
        file := &d.Files[i]
        if file.URL == "" {
            continue
        }
        
        // Add to aria2
        gid, err := s.aria2Engine.Add(ctx, file.URL, engine.DownloadOptions{
            Dir:      d.LocalPath, // Base directory
            Filename: file.Path,   // Preserve path structure
        })
        
        if err != nil {
            file.Status = model.StatusError
            file.Error = err.Error()
            continue
        }
        
        file.EngineID = gid
        file.Status = model.StatusActive
    }
    
    // Update database
    s.downloadRepo.UpdateFiles(ctx, d.ID, d.Files)
}

// startAria2Download downloads magnet via native aria2 BitTorrent
func (s *MagnetService) startAria2Download(ctx context.Context, d *model.Download, magnet string) {
    // Collect selected file indexes
    var indexes []string
    for _, file := range d.Files {
        if file.Index > 0 {
            indexes = append(indexes, fmt.Sprintf("%d", file.Index))
        }
    }
    
    // Add magnet with file selection
    gid, err := s.aria2Engine.AddMagnetWithSelection(ctx, magnet, indexes, engine.DownloadOptions{
        Dir: d.LocalPath,
    })
    
    if err != nil {
        d.Status = model.StatusError
        d.Error = err.Error()
        s.downloadRepo.Update(ctx, d)
        return
    }
    
    d.EngineID = gid
    s.downloadRepo.Update(ctx, d)
}

type MagnetDownloadRequest struct {
    Magnet        string
    Source        string // "alldebrid" or "aria2"
    MagnetID      string
    Name          string
    SelectedFiles []string
    Destination   string
    AllFiles      []model.MagnetFile // For lookup
}

func (r *MagnetDownloadRequest) FindFile(id string) *model.MagnetFile {
    return findFileRecursive(r.AllFiles, id)
}

func findFileRecursive(files []model.MagnetFile, id string) *model.MagnetFile {
    for i := range files {
        if files[i].ID == id {
            return &files[i]
        }
        if len(files[i].Children) > 0 {
            if found := findFileRecursive(files[i].Children, id); found != nil {
                return found
            }
        }
    }
    return nil
}
```

---

## API Handler

**File:** `server/internal/api/magnets.go` (new)

```go
package api

import (
    "encoding/json"
    "net/http"

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
    
    json.NewEncoder(w).Encode(info)
}

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
    
    // Convert files to model
    var allFiles []model.MagnetFile
    for _, f := range req.Files {
        allFiles = append(allFiles, model.MagnetFile{
            ID:   f.ID,
            Name: f.Name,
            Path: f.Path,
            Size: f.Size,
            Link: f.Link,
        })
    }
    
    download, err := h.magnetService.DownloadMagnet(r.Context(), service.MagnetDownloadRequest{
        Magnet:        req.Magnet,
        Source:        req.Source,
        MagnetID:      req.MagnetID,
        Name:          req.Name,
        SelectedFiles: req.SelectedFiles,
        Destination:   req.Destination,
        AllFiles:      allFiles,
    })
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(download)
}
```

---

## Frontend Components

### FileTree Component

**File:** `frontend/src/components/ui/FileTree.tsx`

```tsx
import { Checkbox } from "@heroui/react";
import {
  Tree,
  TreeItem,
  TreeItemContent,
  Collection,
  Button,
} from "react-aria-components";
import { useState } from "react";
import IconChevronRight from "~icons/gravity-ui/chevron-right";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconFolder from "~icons/gravity-ui/folder";
import IconFile from "~icons/gravity-ui/file";
import { formatBytes } from "../../lib/utils";
import type { MagnetFile } from "../../lib/types";

interface FileTreeProps {
  files: MagnetFile[];
  selectedKeys: Set<string>;
  onSelectionChange: (keys: Set<string>) => void;
}

export function FileTree({ files, selectedKeys, onSelectionChange }: FileTreeProps) {
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => {
    // Auto-expand first level
    const keys = new Set<string>();
    files.forEach(f => {
      if (f.isFolder) keys.add(f.id);
    });
    return keys;
  });

  const toggleSelection = (id: string, file: MagnetFile) => {
    const newSelection = new Set(selectedKeys);
    
    if (newSelection.has(id)) {
      // Deselect this and all children
      newSelection.delete(id);
      if (file.children) {
        getAllFileIds(file.children).forEach(childId => newSelection.delete(childId));
      }
    } else {
      // Select this and all children
      newSelection.add(id);
      if (file.children) {
        getAllFileIds(file.children).forEach(childId => newSelection.add(childId));
      }
    }
    
    onSelectionChange(newSelection);
  };

  const renderItem = (file: MagnetFile, level: number = 0) => {
    const isExpanded = expandedKeys.has(file.id);
    const isSelected = selectedKeys.has(file.id);
    
    // Check if all children are selected (for folder partial state)
    const childIds = file.children ? getAllFileIds(file.children) : [];
    const allChildrenSelected = childIds.length > 0 && childIds.every(id => selectedKeys.has(id));
    const someChildrenSelected = childIds.some(id => selectedKeys.has(id));
    
    return (
      <TreeItem
        key={file.id}
        id={file.id}
        textValue={file.name}
        className="outline-none"
      >
        <TreeItemContent>
          <div 
            className="flex items-center gap-2 py-2 px-3 rounded-xl hover:bg-default/10 cursor-pointer transition-colors"
            style={{ paddingLeft: `${level * 20 + 12}px` }}
          >
            {/* Expand/Collapse button for folders */}
            {file.isFolder ? (
              <Button
                className="p-1 rounded hover:bg-default/20 outline-none"
                onPress={() => {
                  const newExpanded = new Set(expandedKeys);
                  if (isExpanded) {
                    newExpanded.delete(file.id);
                  } else {
                    newExpanded.add(file.id);
                  }
                  setExpandedKeys(newExpanded);
                }}
              >
                {isExpanded ? (
                  <IconChevronDown className="w-4 h-4 text-muted" />
                ) : (
                  <IconChevronRight className="w-4 h-4 text-muted" />
                )}
              </Button>
            ) : (
              <span className="w-6" /> // Spacer for alignment
            )}
            
            {/* Checkbox */}
            <Checkbox
              isSelected={file.isFolder ? allChildrenSelected : isSelected}
              isIndeterminate={file.isFolder && someChildrenSelected && !allChildrenSelected}
              onChange={() => toggleSelection(file.id, file)}
              className="mr-2"
            />
            
            {/* Icon */}
            {file.isFolder ? (
              <IconFolder className="w-5 h-5 text-warning" />
            ) : (
              <IconFile className="w-5 h-5 text-muted" />
            )}
            
            {/* Name */}
            <span className="flex-1 text-sm font-medium truncate">
              {file.name}
            </span>
            
            {/* Size */}
            <span className="text-xs text-muted font-mono">
              {formatBytes(file.size)}
            </span>
          </div>
        </TreeItemContent>
        
        {/* Children */}
        {file.isFolder && file.children && isExpanded && (
          <Collection items={file.children}>
            {(child) => renderItem(child, level + 1)}
          </Collection>
        )}
      </TreeItem>
    );
  };

  return (
    <Tree
      aria-label="Torrent files"
      className="w-full"
    >
      <Collection items={files}>
        {(file) => renderItem(file, 0)}
      </Collection>
    </Tree>
  );
}

// Helper to get all file IDs (non-folder) recursively
export function getAllFileIds(files: MagnetFile[]): string[] {
  const ids: string[] = [];
  for (const file of files) {
    if (!file.isFolder) {
      ids.push(file.id);
    }
    if (file.children) {
      ids.push(...getAllFileIds(file.children));
    }
  }
  return ids;
}

// Helper to calculate total size of selected files
export function getSelectedSize(files: MagnetFile[], selectedKeys: Set<string>): number {
  let total = 0;
  for (const file of files) {
    if (!file.isFolder && selectedKeys.has(file.id)) {
      total += file.size;
    }
    if (file.children) {
      total += getSelectedSize(file.children, selectedKeys);
    }
  }
  return total;
}
```

### Updated Add Download Page

**File:** `frontend/src/routes/add.tsx` (modifications)

```tsx
import { useState, useEffect } from "react";
import { Button, Chip, Label, Input, TextArea } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnet from "~icons/gravity-ui/magnet";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import { useDownloadActions } from "../hooks/useDownloads";
import { api } from "../lib/api";
import { useSettingsStore } from "../store/useSettingsStore";
import { tasksLinkOptions } from "./tasks";
import { FileTree, getAllFileIds, getSelectedSize } from "../components/ui/FileTree";
import { formatBytes } from "../lib/utils";
import type { MagnetInfo, MagnetFile } from "../lib/types";

export const Route = createFileRoute("/add")({
  component: AddDownloadPage,
});

function AddDownloadPage() {
  const navigate = useNavigate();
  const { defaultRemote, setDefaultRemote } = useSettingsStore();

  const [uris, setUris] = useState("");
  const [filename, setFilename] = useState("");
  const [resolution, setResolution] = useState<{provider: string, supported: boolean} | null>(null);

  // Magnet state
  const [isMagnet, setIsMagnet] = useState(false);
  const [isCheckingMagnet, setIsCheckingMagnet] = useState(false);
  const [magnetInfo, setMagnetInfo] = useState<MagnetInfo | null>(null);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [magnetError, setMagnetError] = useState<string | null>(null);

  const { create } = useDownloadActions();

  // Detect magnet and check
  useEffect(() => {
    const url = uris.trim();
    
    if (url.startsWith("magnet:")) {
      setIsMagnet(true);
      setResolution(null);
      checkMagnet(url);
    } else {
      setIsMagnet(false);
      setMagnetInfo(null);
      setMagnetError(null);
      setSelectedFiles(new Set());
      
      // Regular URL resolution
      if (url.startsWith("http")) {
        const timer = setTimeout(async () => {
          try {
            const res = await api.resolveUrl(url);
            setResolution(res);
          } catch (err) {
            setResolution(null);
          }
        }, 500);
        return () => clearTimeout(timer);
      }
    }
  }, [uris]);

  const checkMagnet = async (magnet: string) => {
    setIsCheckingMagnet(true);
    setMagnetError(null);
    setMagnetInfo(null);
    
    try {
      const info = await api.checkMagnet(magnet);
      setMagnetInfo(info);
      
      // Pre-select all files
      const allIds = getAllFileIds(info.files);
      setSelectedFiles(new Set(allIds));
    } catch (err: any) {
      setMagnetError(err.message || "Failed to check magnet");
    } finally {
      setIsCheckingMagnet(false);
    }
  };

  const handleSubmit = async () => {
    if (isMagnet && magnetInfo) {
      // Magnet download
      try {
        await api.downloadMagnet({
          magnet: uris.trim(),
          source: magnetInfo.source,
          magnetId: magnetInfo.magnetId,
          name: magnetInfo.name,
          selectedFiles: [...selectedFiles],
          destination: defaultRemote || undefined,
          files: flattenFiles(magnetInfo.files),
        });
        toast.success("Magnet download started");
        navigate(tasksLinkOptions("active"));
      } catch (err: any) {
        toast.error(`Failed to start download: ${err.message}`);
      }
    } else {
      // Regular download
      const uriList = uris.split("\n").filter((u) => u.trim());
      if (uriList.length === 0) return;

      create.mutate(
        {
          url: uriList[0],
          filename: filename || undefined,
          destination: defaultRemote || undefined,
        },
        {
          onSuccess: () => navigate(tasksLinkOptions("active")),
        }
      );
    }
  };

  const selectAllFiles = () => {
    if (magnetInfo) {
      setSelectedFiles(new Set(getAllFileIds(magnetInfo.files)));
    }
  };

  const deselectAllFiles = () => {
    setSelectedFiles(new Set());
  };

  const selectedSize = magnetInfo ? getSelectedSize(magnetInfo.files, selectedFiles) : 0;

  return (
    <div className="max-w-5xl mx-auto space-y-6 pb-20">
      {/* Header */}
      <div className="flex items-center justify-between bg-background p-4 rounded-3xl border border-border shadow-sm">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            isIconOnly
            onPress={() => navigate(tasksLinkOptions("active"))}
            className="h-10 w-10 rounded-xl"
          >
            <IconChevronLeft className="w-5 h-5" />
          </Button>
          <h2 className="text-xl font-black uppercase tracking-tight">
            Add Download
          </h2>
        </div>
        <div className="flex gap-2">
          <Button
            variant="ghost"
            className="px-6 h-10 rounded-xl font-bold"
            onPress={() => navigate(tasksLinkOptions("active"))}
          >
            Cancel
          </Button>
          <Button
            className="px-8 h-10 rounded-xl font-black uppercase tracking-widest shadow-lg shadow-accent/20 bg-accent text-accent-foreground"
            onPress={handleSubmit}
            isDisabled={
              !uris.trim() || 
              create.isPending || 
              isCheckingMagnet || 
              (isMagnet && selectedFiles.size === 0)
            }
            isPending={create.isPending || isCheckingMagnet}
          >
            Start
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column - URL Input */}
        <div className="lg:col-span-7 space-y-6">
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
            <div className="flex flex-col gap-3">
              <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                Download URL
              </Label>
              <TextArea
                placeholder="Paste HTTP, FTP or Magnet links here..."
                value={uris}
                onChange={(e) => setUris(e.target.value)}
                className="w-full p-6 bg-default/10 rounded-3xl text-sm border border-transparent focus:bg-default/15 focus:border-accent/30 transition-all outline-none min-h-[120px] leading-relaxed font-mono"
              />
              
              {/* Magnet indicator */}
              {isMagnet && (
                <div className="flex items-center gap-2 px-1">
                  <IconMagnet className="w-4 h-4 text-accent" />
                  <span className="text-xs font-bold text-accent">Magnet link detected</span>
                </div>
              )}
              
              {/* Regular URL resolution */}
              {!isMagnet && resolution && (
                <div className={`mt-2 p-4 rounded-2xl flex items-center gap-3 border ${resolution.supported ? 'bg-success/5 border-success/20 text-success' : 'bg-warning/5 border-warning/20 text-warning'}`}>
                  <IconNodesDown className="w-5 h-5" />
                  <div className="flex-1">
                    <p className="text-xs font-bold">
                      {resolution.supported 
                        ? `Supported by ${resolution.provider}` 
                        : "No specific provider support, will try direct download"}
                    </p>
                  </div>
                </div>
              )}
              
              {/* Magnet error */}
              {magnetError && (
                <div className="mt-2 p-4 rounded-2xl bg-danger/5 border border-danger/20 text-danger">
                  <p className="text-xs font-bold">{magnetError}</p>
                </div>
              )}
            </div>
          </div>

          {/* File Selection (Magnet only) */}
          {isMagnet && magnetInfo && (
            <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h3 className="font-bold text-lg">{magnetInfo.name}</h3>
                  <p className="text-sm text-muted">
                    {formatBytes(magnetInfo.size)} • {magnetInfo.files.length} files
                  </p>
                </div>
                <Chip 
                  color={magnetInfo.cached ? "success" : "default"}
                  variant="flat"
                  size="sm"
                >
                  {magnetInfo.cached ? "Cached" : magnetInfo.source === "aria2" ? "P2P" : "Not Cached"}
                </Chip>
              </div>
              
              {/* Select/Deselect buttons */}
              <div className="flex gap-2 mb-4">
                <Button
                  size="sm"
                  variant="flat"
                  onPress={selectAllFiles}
                  className="rounded-xl"
                >
                  Select All
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onPress={deselectAllFiles}
                  className="rounded-xl"
                >
                  Deselect All
                </Button>
              </div>
              
              {/* File Tree */}
              <div className="max-h-[400px] overflow-y-auto rounded-2xl border border-border bg-default/5">
                <FileTree
                  files={magnetInfo.files}
                  selectedKeys={selectedFiles}
                  onSelectionChange={setSelectedFiles}
                />
              </div>
              
              {/* Selection summary */}
              <div className="mt-4 flex items-center justify-between text-sm">
                <span className="text-muted">
                  {selectedFiles.size} files selected
                </span>
                <span className="font-bold">
                  {formatBytes(selectedSize)}
                </span>
              </div>
            </div>
          )}

          {/* Loading state for magnet check */}
          {isMagnet && isCheckingMagnet && (
            <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center justify-center gap-3 py-8">
                <div className="animate-spin rounded-full h-6 w-6 border-2 border-accent border-t-transparent" />
                <span className="text-sm text-muted">Fetching torrent info...</span>
              </div>
            </div>
          )}
        </div>

        {/* Right Column - Options */}
        <div className="lg:col-span-5 space-y-6">
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm space-y-6">
            {/* Filename (only for non-magnet) */}
            {!isMagnet && (
              <div className="flex flex-col gap-2">
                <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                  Filename (Optional)
                </Label>
                <Input
                  placeholder="original-name.zip"
                  value={filename}
                  onChange={(e) => setFilename(e.target.value)}
                  className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
                />
              </div>
            )}

            <div className="flex flex-col gap-2">
              <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                Upload Target
              </Label>
              <Input
                placeholder="e.g. gdrive:/downloads"
                value={defaultRemote}
                onChange={(e) => setDefaultRemote(e.target.value)}
                className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
              />
              <p className="text-[10px] text-muted font-medium px-1">
                Enter a remote path to automatically offload files to the cloud.
              </p>
            </div>
          </div>

          {/* Magnet source info */}
          {isMagnet && magnetInfo && (
            <div className="bg-background p-6 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center gap-3">
                <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${
                  magnetInfo.source === "alldebrid" ? "bg-success/10 text-success" : "bg-accent/10 text-accent"
                }`}>
                  {magnetInfo.source === "alldebrid" ? "AD" : "P2P"}
                </div>
                <div>
                  <p className="font-bold text-sm">
                    {magnetInfo.source === "alldebrid" ? "AllDebrid" : "BitTorrent"}
                  </p>
                  <p className="text-xs text-muted">
                    {magnetInfo.source === "alldebrid" 
                      ? "Fast direct download from cache" 
                      : "Peer-to-peer download"}
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Helper to flatten nested files for API request
function flattenFiles(files: MagnetFile[]): any[] {
  const result: any[] = [];
  
  function traverse(items: MagnetFile[], index: { current: number }) {
    for (const file of items) {
      if (!file.isFolder) {
        result.push({
          id: file.id,
          name: file.name,
          path: file.path,
          size: file.size,
          link: (file as any).link,
          index: index.current++,
        });
      }
      if (file.children) {
        traverse(file.children, index);
      }
    }
  }
  
  traverse(files, { current: 1 });
  return result;
}
```

### API Client Updates

**File:** `frontend/src/lib/api.ts` (add methods)

```typescript
// Magnet operations
checkMagnet(magnet: string) {
  return this.request<MagnetInfo>('POST', '/magnets/check', { magnet });
}

downloadMagnet(req: {
  magnet: string;
  source: string;
  magnetId?: string;
  name: string;
  selectedFiles: string[];
  destination?: string;
  files: any[];
}) {
  return this.request<Download>('POST', '/magnets/download', req);
}

getDownloadFiles(id: string) {
  return this.request<{ files: DownloadFile[] }>('GET', `/downloads/${id}/files`);
}
```

---

## File Changes Summary

### Backend (Go) - New Files

| File | Description |
|------|-------------|
| `server/internal/model/magnet.go` | Magnet and MagnetFile types |
| `server/internal/provider/alldebrid/magnet.go` | AllDebrid magnet API implementation |
| `server/internal/engine/aria2/magnet.go` | aria2c raw magnet implementation |
| `server/internal/service/magnet.go` | Magnet business logic |
| `server/internal/api/magnets.go` | HTTP handlers for /magnets/* |
| `server/internal/store/migrations/003_magnet_files.sql` | Database schema |

### Backend (Go) - Modified Files

| File | Changes |
|------|---------|
| `server/internal/model/download.go` | Add DownloadFile, magnet fields |
| `server/internal/provider/provider.go` | Add MagnetProvider interface |
| `server/internal/api/router.go` | Register /magnets routes |
| `server/internal/store/download.go` | Add file storage methods |
| `server/internal/service/download.go` | Handle multi-file progress |

### Frontend (TypeScript/React) - New Files

| File | Description |
|------|-------------|
| `frontend/src/components/ui/FileTree.tsx` | React Aria Tree for file selection |

### Frontend (TypeScript/React) - Modified Files

| File | Changes |
|------|---------|
| `frontend/src/lib/types.ts` | Add magnet types |
| `frontend/src/lib/api.ts` | Add magnet API methods |
| `frontend/src/routes/add.tsx` | Add magnet detection and FileTree |
| `frontend/src/components/dashboard/DownloadCard.tsx` | Show file count for magnets |

---

## Implementation Order

### Phase 1: Backend Core (2-3 hours)
1. Create `model/magnet.go` with types
2. Update `model/download.go` with magnet fields
3. Create database migration
4. Update `store/download.go` for file storage

### Phase 2: AllDebrid Integration (2-3 hours)
1. Add `MagnetProvider` interface to `provider/provider.go`
2. Implement `alldebrid/magnet.go`
3. Test with real AllDebrid API

### Phase 3: aria2c Raw Magnet (2-3 hours)
1. Implement `aria2/magnet.go`
2. Test metadata fetching
3. Test `--select-file` functionality

### Phase 4: API & Service Layer (2 hours)
1. Create `service/magnet.go`
2. Create `api/magnets.go`
3. Register routes
4. Test endpoints

### Phase 5: Frontend FileTree (2-3 hours)
1. Create `FileTree.tsx` component
2. Style with project conventions
3. Test selection behavior

### Phase 6: Frontend Integration (2-3 hours)
1. Update `add.tsx` with magnet flow
2. Add API methods
3. Test full flow

### Phase 7: Progress Tracking (2 hours)
1. Update download service for multi-file progress
2. Update DownloadCard for magnet display
3. Test progress aggregation

### Phase 8: Testing & Polish (2 hours)
1. End-to-end testing
2. Error handling
3. Edge cases

---

## Testing Checklist

- [ ] AllDebrid cached magnet shows file tree
- [ ] AllDebrid non-cached magnet falls back to aria2
- [ ] Raw magnet (no AllDebrid) shows file tree via aria2
- [ ] File selection works (select/deselect individual files)
- [ ] Folder selection selects all children
- [ ] "Select All" / "Deselect All" works
- [ ] Selected size updates correctly
- [ ] Download starts with only selected files
- [ ] Progress shows cumulative (total downloaded / total size)
- [ ] Task detail shows per-file progress
- [ ] Upload to cloud works after magnet download completes
- [ ] Canceling magnet download cancels all files
- [ ] Error handling for failed individual files
