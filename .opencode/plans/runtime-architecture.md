# Runtime Architecture & Execution Flow

This document explains how the `aria2-rclone-ui` application functions at runtime, detailing the role of the Go backend as a "Process Supervisor" and "Automation Orchestrator".

## 1. The Supervisor Architecture
Instead of managing three separate components (Frontend, Aria2, Rclone), you run a single Go binary. This binary acts as the central command center.

### **The Stack**
-   **Go Server (Port 8080)**: The main entry point.
-   **Aria2c (Child Process, Port 6800)**: Managed download engine.
-   **Rclone RCD (Child Process, Port 5572)**: Managed upload engine.
-   **React Frontend**: Static files served by the Go server.

## 2. Execution Lifecycle

When you run the Go binary (`./server`), the following sequence occurs:

### **Phase 1: Startup & Process Orchestration**
1.  **Process Spawning**:
    *   The Go app launches `aria2c` in the background with RPC enabled (`--enable-rpc`) and a generated secret.
    *   The Go app launches `rclone rcd` in the background to expose the Remote Control API.
2.  **Output Piping**:
    *   Standard Output (logs) from Aria2 and Rclone are piped to the Go app's console, giving you a unified log stream.

### **Phase 2: API Gateway Initialization**
1.  **HTTP Server Starts**: The Go app listens on port `8080`.
2.  **Reverse Proxies**:
    *   `/jsonrpc` -> Proxies traffic to internal Aria2 (localhost:6800).
    *   `/rclone/*` -> Proxies traffic to internal Rclone (localhost:5572).
    *   `/*` -> Serves the React frontend (SPA).
    *   **Benefit**: This solves CORS issues and hides the complexity of multiple ports from the frontend.

### **Phase 3: The "Glue" (Automation Event Loop)**
1.  **WebSocket Connection**:
    *   The Go app connects to Aria2's WebSocket (`ws://localhost:6800/jsonrpc`) as an internal client.
2.  **Event Listening**:
    *   It sits idle, waiting for the `aria2.onDownloadComplete` event.

### **Phase 4: The Auto-Upload Workflow**
When a download finishes:
1.  **Trigger**: Aria2 sends `onDownloadComplete` with the Task GID.
2.  **Lookup**: The Go app calls `aria2.tellStatus(gid)` to find the local file path of the download.
3.  **Action**: The Go app sends a command to Rclone (`operations/copyfile`) to upload that file to your configured cloud remote (e.g., `gdrive:`).
4.  **Visualization**: The Frontend polls `/rclone/core/stats` to show the upload progress in real-time.

### **Phase 5: Graceful Shutdown**
1.  **Signal Trap**: When you press `Ctrl+C`, the Go app catches the signal.
2.  **Cleanup**: It sends kill signals to the `aria2c` and `rclone` processes, ensuring they shut down cleanly and don't remain as zombie processes.

## 3. User Experience
From the user's perspective:
1.  Run `./server`.
2.  Open `http://localhost:8080`.
3.  Add a download in the UI.
4.  Watch it download (Aria2 stats).
5.  Watch it automatically upload (Rclone stats).
6.  Close the app when done.
