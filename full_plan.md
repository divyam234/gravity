# Gravity - Download & Cloud Upload Manager

## Overview

**Gravity** is a modern download manager with automatic cloud upload capabilities. It provides a clean REST API and real-time WebSocket updates, abstracting away the complexity of underlying engines (Aria2, Rclone).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                    FRONTEND                                         │
│                              (React + TanStack Query)                               │
└───────────────────────────────────────┬─────────────────────────────────────────────┘
                                        │
                    ┌───────────────────┴───────────────────┐
                    ▼                                       ▼
        ┌───────────────────┐                   ┌───────────────────┐
        │   REST API        │                   │   WebSocket       │
        │   /api/v1/*       │                   │   /ws             │
        └─────────┬─────────┘                   └─────────┬─────────┘
                  │                                       │
                  └───────────────────┬───────────────────┘
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              SERVICE LAYER                                          │
│                                                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌───────────────┐  │
│  │ DownloadService │  │ ProviderService │  │  UploadService  │  │ StatsService  │  │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘  └───────┬───────┘  │
└───────────┼────────────────────┼────────────────────┼───────────────────┼──────────┘
            │                    │                    │                   │
            ▼                    ▼                    ▼                   ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              ENGINE LAYER (Hidden)                                  │
│                                                                                     │
│  ┌──────────────────────────────┐       ┌──────────────────────────────┐           │
│  │   DownloadEngine Interface   │       │   UploadEngine Interface     │           │
│  │                              │       │                              │           │
│  │   Implementation: Aria2      │       │   Implementation: Rclone     │           │
│  └──────────────────────────────┘       └──────────────────────────────┘           │
│                                                                                     │
│  ┌──────────────────────────────────────────────────────────────────────────────┐  │
│  │                         Provider System                                       │  │
│  │                                                                               │  │
│  │   ┌─────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────┐                 │  │
│  │   │ Direct  │  │  AllDebrid  │  │ Real-Debrid │  │  More  │                 │  │
│  │   └─────────┘  └─────────────┘  └─────────────┘  └────────┘                 │  │
│  └──────────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              DATA LAYER                                             │
│                                                                                     │
│                            SQLite Database                                          │
│   ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐       │
│   │ downloads │  │ providers │  │  remotes  │  │   stats   │  │ settings  │       │
│   └───────────┘  └───────────┘  └───────────┘  └───────────┘  └───────────┘       │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
gravity/
├── cmd/
│   └── gravity/
│       └── main.go                 # Entry point (~80 lines)
│
├── internal/
│   ├── api/                        # HTTP layer
│   │   ├── router.go               # Chi router setup
│   │   ├── middleware.go           # Auth, logging, CORS
│   │   ├── downloads.go            # Download endpoints
│   │   ├── providers.go            # Provider endpoints
│   │   ├── remotes.go              # Remote endpoints
│   │   ├── stats.go                # Stats endpoints
│   │   ├── settings.go             # Settings endpoints
│   │   └── websocket.go            # WebSocket handler
│   │
│   ├── service/                    # Business logic
│   │   ├── download.go             # Download orchestration
│   │   ├── upload.go               # Upload orchestration
│   │   ├── provider.go             # Provider management
│   │   └── stats.go                # Statistics
│   │
│   ├── engine/                     # External tool abstractions
│   │   ├── download.go             # DownloadEngine interface
│   │   ├── upload.go               # UploadEngine interface
│   │   ├── aria2/                  # Aria2 implementation
│   │   │   ├── engine.go
│   │   │   ├── client.go
│   │   │   └── runner.go
│   │   └── rclone/                 # Rclone implementation
│   │       ├── engine.go
│   │       ├── client.go
│   │       └── runner.go
│   │
│   ├── provider/                   # Link resolution providers
│   │   ├── provider.go             # Interfaces
│   │   ├── registry.go             # Provider registry
│   │   ├── resolver.go             # URL resolution logic
│   │   ├── direct/                 # Direct downloads
│   │   │   └── direct.go
│   │   ├── alldebrid/              # AllDebrid
│   │   │   └── alldebrid.go
│   │   └── realdebrid/             # Real-Debrid
│   │       └── realdebrid.go
│   │
│   ├── store/                      # Data layer
│   │   ├── sqlite.go               # SQLite connection
│   │   ├── migrations.go           # Migration runner
│   │   ├── migrations/             # SQL files
│   │   │   └── 001_initial.sql
│   │   ├── download.go             # Download repository
│   │   ├── provider.go             # Provider repository
│   │   ├── remote.go               # Remote repository
│   │   ├── stats.go                # Stats repository
│   │   └── settings.go             # Settings repository
│   │
│   ├── model/                      # Domain models
│   │   ├── download.go
│   │   ├── provider.go
│   │   ├── remote.go
│   │   └── stats.go
│   │
│   ├── event/                      # Event system
│   │   ├── bus.go                  # Event bus
│   │   └── types.go                # Event types
│   │
│   └── config/                     # Configuration
│       └── config.go
│
├── frontend/                       # React frontend (existing, to be updated)
│   └── src/
│
├── go.mod
├── go.sum
└── README.md
```

---

## REST API Specification

### Base URL
```
/api/v1
```

### Authentication
```
Header: X-API-Key: <api-key>
```

---

### Downloads

#### List Downloads
```http
GET /api/v1/downloads
GET /api/v1/downloads?status=downloading,paused
GET /api/v1/downloads?limit=50&offset=0
GET /api/v1/downloads?sort=-createdAt
```

**Response:**
```json
{
  "data": [
    {
      "id": "d_abc123",
      "status": "downloading",
      "url": "https://example.com/file.zip",
      "filename": "file.zip",
      "size": 1073741824,
      "downloaded": 536870912,
      "progress": 50.0,
      "speed": 10485760,
      "eta": 51,
      "provider": "alldebrid",
      "destination": "gdrive:/downloads",
      "createdAt": "2024-01-15T10:30:00Z",
      "error": null
    }
  ],
  "meta": {
    "total": 150,
    "limit": 50,
    "offset": 0
  }
}
```

#### Create Download
```http
POST /api/v1/downloads
Content-Type: application/json

{
  "url": "https://example.com/file.zip",
  "filename": "custom-name.zip",
  "destination": "gdrive:/backups",
  "category": "movies",
  "tags": ["4k", "remux"]
}
```

**Response:**
```json
{
  "id": "d_abc123",
  "status": "pending",
  "url": "https://example.com/file.zip",
  "filename": "custom-name.zip",
  "destination": "gdrive:/backups",
  "category": "movies",
  "tags": ["4k", "remux"],
  "createdAt": "2024-01-15T10:30:00Z"
}
```

#### Create Batch Downloads
```http
POST /api/v1/downloads/batch
Content-Type: application/json

{
  "urls": [
    "https://example.com/file1.zip",
    "https://example.com/file2.zip"
  ],
  "destination": "gdrive:/backups"
}
```

#### Get Download
```http
GET /api/v1/downloads/:id
```

#### Delete Download
```http
DELETE /api/v1/downloads/:id
DELETE /api/v1/downloads/:id?deleteFiles=true
```

#### Pause Download
```http
POST /api/v1/downloads/:id/pause
```

#### Resume Download
```http
POST /api/v1/downloads/:id/resume
```

#### Retry Download
```http
POST /api/v1/downloads/:id/retry
```

#### Batch Operations
```http
POST /api/v1/downloads/batch/pause
Content-Type: application/json
{ "ids": ["d_abc123", "d_def456"] }

POST /api/v1/downloads/batch/resume
POST /api/v1/downloads/batch/delete
```

---

### Providers

#### List Providers
```http
GET /api/v1/providers
```

**Response:**
```json
{
  "data": [
    {
      "name": "direct",
      "displayName": "Direct Download",
      "type": "direct",
      "enabled": true,
      "configured": true,
      "priority": 0
    },
    {
      "name": "alldebrid",
      "displayName": "AllDebrid",
      "type": "debrid",
      "enabled": true,
      "configured": true,
      "priority": 100,
      "account": {
        "username": "user@example.com",
        "isPremium": true,
        "expiresAt": "2025-12-31T23:59:59Z",
        "supportedHosts": 150
      }
    }
  ]
}
```

#### Get Provider
```http
GET /api/v1/providers/:name
```

**Response includes config schema:**
```json
{
  "name": "alldebrid",
  "displayName": "AllDebrid",
  "type": "debrid",
  "enabled": true,
  "configured": true,
  "priority": 100,
  "configSchema": [
    {
      "key": "api_key",
      "label": "API Key",
      "type": "password",
      "required": true,
      "description": "Get from alldebrid.com/apikeys"
    }
  ],
  "account": {
    "username": "user@example.com",
    "isPremium": true,
    "expiresAt": "2025-12-31T23:59:59Z"
  }
}
```

#### Configure Provider
```http
PUT /api/v1/providers/:name
Content-Type: application/json

{
  "enabled": true,
  "priority": 100,
  "config": {
    "api_key": "xxxxxxxxxxxx"
  }
}
```

#### Delete Provider Config
```http
DELETE /api/v1/providers/:name
```

#### Test Provider
```http
POST /api/v1/providers/:name/test
```

**Response:**
```json
{
  "success": true,
  "message": "Connected successfully",
  "account": {
    "username": "user@example.com",
    "isPremium": true
  }
}
```

#### Get Supported Hosts
```http
GET /api/v1/providers/:name/hosts
```

#### Resolve URL (Preview)
```http
POST /api/v1/providers/resolve
Content-Type: application/json

{
  "url": "https://rapidgator.net/file/abc123"
}
```

**Response:**
```json
{
  "url": "https://rapidgator.net/file/abc123",
  "provider": "alldebrid",
  "supported": true,
  "hostName": "Rapidgator"
}
```

---

### Remotes (Cloud Destinations)

#### List Remotes
```http
GET /api/v1/remotes
```

**Response:**
```json
{
  "data": [
    {
      "name": "gdrive",
      "type": "drive",
      "connected": true
    },
    {
      "name": "s3-backup",
      "type": "s3",
      "connected": true
    }
  ]
}
```

#### Create Remote
```http
POST /api/v1/remotes
Content-Type: application/json

{
  "name": "my-gdrive",
  "type": "drive",
  "config": {
    "client_id": "xxx",
    "client_secret": "xxx",
    "token": "xxx"
  }
}
```

#### Delete Remote
```http
DELETE /api/v1/remotes/:name
```

#### Test Remote
```http
POST /api/v1/remotes/:name/test
```

---

### Stats

#### Get Current Stats
```http
GET /api/v1/stats
```

**Response:**
```json
{
  "active": {
    "downloads": 3,
    "uploads": 1,
    "downloadSpeed": 52428800,
    "uploadSpeed": 10485760
  },
  "queue": {
    "pending": 5,
    "paused": 2
  },
  "totals": {
    "downloaded": 1099511627776,
    "uploaded": 549755813888,
    "completed": 1250,
    "failed": 12
  },
  "byProvider": {
    "alldebrid": {
      "resolved": 500,
      "bytes": 549755813888
    },
    "direct": {
      "resolved": 750,
      "bytes": 549755813888
    }
  }
}
```

---

### Settings

#### Get Settings
```http
GET /api/v1/settings
```

**Response:**
```json
{
  "downloadDir": "/downloads",
  "maxConcurrentDownloads": 5,
  "maxConcurrentUploads": 2,
  "defaultDestination": "gdrive:/downloads",
  "autoUpload": true,
  "deleteAfterUpload": false
}
```

#### Update Settings
```http
PATCH /api/v1/settings
Content-Type: application/json

{
  "maxConcurrentDownloads": 10
}
```

---

## WebSocket API

### Connection
```
ws://localhost:8080/ws?token=<api-key>
```

### Message Format
```json
{
  "type": "event_type",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "data": { ... }
}
```

### Events

| Event | Description | Data |
|-------|-------------|------|
| `download.created` | New download added | Download object |
| `download.started` | Download began | `{ id, filename }` |
| `download.progress` | Progress update (1/sec) | `{ id, downloaded, speed, eta }` |
| `download.paused` | Download paused | `{ id }` |
| `download.resumed` | Download resumed | `{ id }` |
| `download.completed` | Download finished | `{ id, filename, size }` |
| `download.error` | Download failed | `{ id, error }` |
| `upload.started` | Upload began | `{ id, destination }` |
| `upload.progress` | Upload progress (1/sec) | `{ id, uploaded, speed }` |
| `upload.completed` | Upload finished | `{ id, destination }` |
| `upload.error` | Upload failed | `{ id, error }` |
| `stats` | Stats update (5/sec) | Stats object |

---

## Database Schema

```sql
-- 001_initial.sql

-- Downloads
CREATE TABLE downloads (
    id TEXT PRIMARY KEY,
    
    -- Source
    url TEXT NOT NULL,
    resolved_url TEXT,
    provider TEXT,
    
    -- Status
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT,
    
    -- File info
    filename TEXT,
    size INTEGER DEFAULT 0,
    downloaded INTEGER DEFAULT 0,
    
    -- Progress
    speed INTEGER DEFAULT 0,
    eta INTEGER DEFAULT 0,
    
    -- Destination
    destination TEXT,
    upload_status TEXT,
    upload_progress INTEGER DEFAULT 0,
    upload_speed INTEGER DEFAULT 0,
    
    -- Organization
    category TEXT,
    tags TEXT,  -- JSON array
    
    -- Engine references (internal)
    engine_id TEXT,
    upload_job_id TEXT,
    
    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_downloads_status ON downloads(status);
CREATE INDEX idx_downloads_created ON downloads(created_at DESC);

-- Providers
CREATE TABLE providers (
    name TEXT PRIMARY KEY,
    enabled INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 0,
    config TEXT,  -- JSON, encrypted
    cached_hosts TEXT,  -- JSON array
    cached_account TEXT,  -- JSON
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Remotes
CREATE TABLE remotes (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    config TEXT,  -- JSON, managed by rclone
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Stats
CREATE TABLE stats (
    key TEXT PRIMARY KEY,
    value INTEGER DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO stats (key, value) VALUES
    ('total_downloaded', 0),
    ('total_uploaded', 0),
    ('downloads_completed', 0),
    ('uploads_completed', 0),
    ('downloads_failed', 0);

-- Provider stats
CREATE TABLE provider_stats (
    provider TEXT NOT NULL,
    date TEXT NOT NULL,  -- YYYY-MM-DD
    resolved INTEGER DEFAULT 0,
    bytes INTEGER DEFAULT 0,
    PRIMARY KEY (provider, date)
);

-- Settings
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO settings (key, value) VALUES
    ('download_dir', '/downloads'),
    ('max_concurrent_downloads', '5'),
    ('max_concurrent_uploads', '2'),
    ('default_destination', ''),
    ('auto_upload', 'true'),
    ('delete_after_upload', 'false'),
    ('api_key', NULL);
```

---

## Implementation Phases

### Phase 1: Foundation (Days 1-3)
> Core infrastructure, can't do anything without this

| Task | Description | Files |
|------|-------------|-------|
| P1.1 | Project setup, go.mod, directory structure | `go.mod`, dirs |
| P1.2 | Configuration loading | `config/config.go` |
| P1.3 | SQLite setup + migrations | `store/sqlite.go`, `store/migrations.go` |
| P1.4 | Domain models | `model/*.go` |
| P1.5 | Event bus for real-time updates | `event/bus.go`, `event/types.go` |
| P1.6 | HTTP router + middleware | `api/router.go`, `api/middleware.go` |
| P1.7 | WebSocket handler | `api/websocket.go` |

### Phase 2: Engine Layer (Days 4-6)
> Abstract Aria2 and Rclone behind clean interfaces

| Task | Description | Files |
|------|-------------|-------|
| P2.1 | DownloadEngine interface | `engine/download.go` |
| P2.2 | Aria2 runner (process management) | `engine/aria2/runner.go` |
| P2.3 | Aria2 client (RPC calls) | `engine/aria2/client.go` |
| P2.4 | Aria2 engine implementation | `engine/aria2/engine.go` |
| P2.5 | UploadEngine interface | `engine/upload.go` |
| P2.6 | Rclone runner | `engine/rclone/runner.go` |
| P2.7 | Rclone client | `engine/rclone/client.go` |
| P2.8 | Rclone engine implementation | `engine/rclone/engine.go` |

### Phase 3: Core Services (Days 7-9)
> Business logic layer

| Task | Description | Files |
|------|-------------|-------|
| P3.1 | Download repository | `store/download.go` |
| P3.2 | Download service | `service/download.go` |
| P3.3 | Download API endpoints | `api/downloads.go` |
| P3.4 | Upload service | `service/upload.go` |
| P3.5 | Stats repository + service | `store/stats.go`, `service/stats.go` |
| P3.6 | Stats API endpoints | `api/stats.go` |
| P3.7 | Settings repository + API | `store/settings.go`, `api/settings.go` |

### Phase 4: Provider System (Days 10-12)
> Debrid and file host integrations

| Task | Description | Files |
|------|-------------|-------|
| P4.1 | Provider interfaces | `provider/provider.go` |
| P4.2 | Provider registry | `provider/registry.go` |
| P4.3 | URL resolver | `provider/resolver.go` |
| P4.4 | Direct provider (passthrough) | `provider/direct/direct.go` |
| P4.5 | AllDebrid provider | `provider/alldebrid/alldebrid.go` |
| P4.6 | Real-Debrid provider | `provider/realdebrid/realdebrid.go` |
| P4.7 | Provider repository | `store/provider.go` |
| P4.8 | Provider service | `service/provider.go` |
| P4.9 | Provider API endpoints | `api/providers.go` |

### Phase 5: Remotes & Polish (Days 13-14)
> Cloud destinations and finishing touches

| Task | Description | Files |
|------|-------------|-------|
| P5.1 | Remote repository | `store/remote.go` |
| P5.2 | Remote API endpoints | `api/remotes.go` |
| P5.3 | Main entry point | `cmd/gravity/main.go` |
| P5.4 | Integration testing | `*_test.go` |
| P5.5 | Error handling polish | All files |

### Phase 6: Frontend Updates (Days 15-17)
> Update React frontend for new API

| Task | Description | Files |
|------|-------------|-------|
| P6.1 | New API client | `frontend/src/lib/api.ts` |
| P6.2 | WebSocket hook | `frontend/src/hooks/useWebSocket.ts` |
| P6.3 | Update useDownloads hook | `frontend/src/hooks/useDownloads.ts` |
| P6.4 | Provider settings page | `frontend/src/routes/settings.providers.tsx` |
| P6.5 | Update task list | `frontend/src/components/` |
| P6.6 | Update add download dialog | `frontend/src/routes/add.tsx` |

---

## Core Interfaces

### DownloadEngine
```go
// engine/download.go

type DownloadEngine interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop() error
    
    // Operations
    Add(ctx context.Context, url string, opts DownloadOptions) (string, error)
    Pause(ctx context.Context, id string) error
    Resume(ctx context.Context, id string) error
    Cancel(ctx context.Context, id string) error
    Remove(ctx context.Context, id string) error
    
    // Status
    Status(ctx context.Context, id string) (*DownloadStatus, error)
    List(ctx context.Context) ([]*DownloadStatus, error)
    
    // Events
    OnProgress(handler func(id string, progress Progress))
    OnComplete(handler func(id string, filePath string))
    OnError(handler func(id string, err error))
}

type DownloadOptions struct {
    Filename    string
    Dir         string
    Headers     map[string]string
    MaxSpeed    int64
    Connections int
}

type DownloadStatus struct {
    ID          string
    Status      string  // active, paused, complete, error
    URL         string
    Filename    string
    Dir         string
    Size        int64
    Downloaded  int64
    Speed       int64
    Connections int
    Error       string
}

type Progress struct {
    Downloaded int64
    Size       int64
    Speed      int64
    ETA        int
}
```

### UploadEngine
```go
// engine/upload.go

type UploadEngine interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop() error
    
    // Operations
    Upload(ctx context.Context, src, dst string, opts UploadOptions) (string, error)
    Cancel(ctx context.Context, jobID string) error
    
    // Status
    Status(ctx context.Context, jobID string) (*UploadStatus, error)
    
    // Events
    OnProgress(handler func(jobID string, progress UploadProgress))
    OnComplete(handler func(jobID string))
    OnError(handler func(jobID string, err error))
    
    // Remotes
    ListRemotes(ctx context.Context) ([]Remote, error)
    CreateRemote(ctx context.Context, name, rtype string, config map[string]string) error
    DeleteRemote(ctx context.Context, name string) error
    TestRemote(ctx context.Context, name string) error
}

type UploadOptions struct {
    DeleteAfter bool
}

type UploadStatus struct {
    JobID    string
    Status   string  // running, complete, error
    Src      string
    Dst      string
    Size     int64
    Uploaded int64
    Speed    int64
    Error    string
}
```

### Provider
```go
// provider/provider.go

type Provider interface {
    // Metadata
    Name() string
    DisplayName() string
    Type() ProviderType  // direct, debrid, filehost
    
    // Configuration
    ConfigSchema() []ConfigField
    Configure(config map[string]string) error
    IsConfigured() bool
    
    // URL handling
    Supports(url string) bool
    Priority() int
    Resolve(ctx context.Context, url string) (*ResolveResult, error)
    
    // Health
    Test(ctx context.Context) (*AccountInfo, error)
}

type DebridProvider interface {
    Provider
    GetHosts(ctx context.Context) ([]string, error)
}

type ConfigField struct {
    Key         string `json:"key"`
    Label       string `json:"label"`
    Type        string `json:"type"`  // text, password, select
    Required    bool   `json:"required"`
    Description string `json:"description"`
}

type ResolveResult struct {
    URL      string
    Filename string
    Size     int64
    Headers  map[string]string
    Error    string
}

type AccountInfo struct {
    Username  string
    IsPremium bool
    ExpiresAt *time.Time
    Hosts     []string
}
```

---

## Event System

```go
// event/types.go

type EventType string

const (
    EventDownloadCreated   EventType = "download.created"
    EventDownloadStarted   EventType = "download.started"
    EventDownloadProgress  EventType = "download.progress"
    EventDownloadPaused    EventType = "download.paused"
    EventDownloadResumed   EventType = "download.resumed"
    EventDownloadCompleted EventType = "download.completed"
    EventDownloadError     EventType = "download.error"
    
    EventUploadStarted     EventType = "upload.started"
    EventUploadProgress    EventType = "upload.progress"
    EventUploadCompleted   EventType = "upload.completed"
    EventUploadError       EventType = "upload.error"
    
    EventStats             EventType = "stats"
)

type Event struct {
    Type      EventType   `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data"`
}
```

```go
// event/bus.go

type Bus struct {
    subscribers map[chan Event]struct{}
    mu          sync.RWMutex
}

func (b *Bus) Subscribe() <-chan Event
func (b *Bus) Unsubscribe(ch <-chan Event)
func (b *Bus) Publish(event Event)
```

---

## Success Criteria

1. ✅ Clean REST API (no Aria2 JSON-RPC exposed)
2. ✅ Real-time WebSocket updates
3. ✅ SQLite database (fresh, no migration)
4. ✅ Provider plugin system (AllDebrid + Real-Debrid)
5. ✅ Download + auto-upload to cloud
6. ✅ Frontend updated for new API
7. ✅ All builds pass

---

## Timeline

| Phase | Duration | Total |
|-------|----------|-------|
| Phase 1: Foundation | 3 days | Day 3 |
| Phase 2: Engine Layer | 3 days | Day 6 |
| Phase 3: Core Services | 3 days | Day 9 |
| Phase 4: Provider System | 3 days | Day 12 |
| Phase 5: Remotes & Polish | 2 days | Day 14 |
| Phase 6: Frontend | 3 days | Day 17 |

**Total: ~17 days**

---

## Next Steps

1. Review and approve this plan
2. Create new `gravity/` directory (or rename existing)
3. Begin Phase 1 implementation
