# Gravity

**Gravity** is a modern, self-hosted download manager designed to abstract away the complexity of underlying engines like Aria2 and Rclone. It provides a unified, beautiful dashboard for managing downloads and automatic cloud uploads.

## Features

*   **Unified Dashboard:** Manage all your downloads in one place.
*   **Modern UI:** Built with React, TanStack Router, and HeroUI.
*   **Engine Abstraction:** Works with Aria2 for downloads and Rclone for cloud transfers, but exposes a clean, generic API.
*   **Auto-Upload:** Automatically upload finished downloads to configured cloud remotes.
*   **Real-time Stats:** Live speed and progress updates via WebSockets.
*   **Provider Integration:** Built-in support for Debrid services (Real-Debrid, AllDebrid).

## Architecture

*   **Backend:** Go (Golang) - Handles engine management, database (SQLite), and API.
*   **Frontend:** React (Vite) - Interacts with the backend via a REST API and WebSockets.
*   **Engines:** 
    *   **Aria2:** High-performance download utility.
    *   **Rclone:** Cloud storage management and transfer.

## Prerequisites

*   **Go** (1.21 or later)
*   **Node.js** (20+) or **Bun**
*   **Aria2c** (installed and available in PATH)
*   **Rclone** (installed and available in PATH)

## Development Setup

### 1. Frontend

Navigate to the frontend directory and install dependencies:

```bash
cd frontend
bun install
```

Start the development server:

```bash
bun run dev
```

### 2. Backend

Navigate to the server directory:

```bash
cd server
```

Run the server (it will look for the frontend build in `dist` by default, but for dev you can rely on the Vite proxy):

```bash
go run cmd/gravity/main.go
```

**Note:** The backend expects an `aria2c` process or manages one. By default, it will try to start one.

## Building for Production

1.  **Build Frontend:**

    ```bash
    cd frontend
    bun run build
    ```
    This will output static files to `server/dist` (configured in `vite.config.ts`).

2.  **Build Backend:**

    ```bash
    cd server
    go build -o gravity cmd/gravity/main.go
    ```

3.  **Run:**

    ```bash
    ./gravity
    ```
    Ensure `aria2c` and `rclone` are in your PATH.

## Configuration

Gravity uses a SQLite database located in the `data` directory (created on first run). You can configure settings via the Web UI.

## License

MIT