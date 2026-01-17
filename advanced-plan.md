# Aria2c RPC Control Panel - Advanced Features Plan

## 1. Selective Download (File Browser)
- **Goal**: Allow users to select which files to download within a multi-file torrent/metalink.
- **Components**: 
  - `TaskDetailsDrawer`: A side panel for detailed file view.
  - `FileList`: A list with checkboxes for each file.
- **Logic**: Use `aria2.getFiles(gid)` and `aria2.changeOption(gid, { 'select-file': '1,3,5' })`.

## 2. Torrent File Support
- **Goal**: Full support for `.torrent` and `.metalink` file uploads.
- **Implementation**: 
  - Add `FileReader` logic to `AddDownloadModal`.
  - Convert file to Base64.
  - Call `aria2.addTorrent` or `aria2.addMetalink`.

## 3. Real-time Speed Visualization
- **Goal**: Visual speed history sparklines.
- **Implementation**:
  - `useSpeedHistory` hook to buffer last 60 seconds of speed data.
  - Simple SVG sparkline component in `StatsOverview`.

## 4. System Notifications
- **Goal**: Desktop notifications for download completion or errors.
- **Implementation**:
  - `useNotifications` hook to monitor task status changes.
  - Browser `Notification` API integration.

## 5. WebSocket Integration
- **Goal**: Lower latency and real-time push notifications.
- **Implementation**: 
  - Upgrade `Aria2Client` to support `WebSocket` protocol.
  - Use `aria2.onDownloadStart`, `aria2.onDownloadComplete` events.

## 6. UI/UX Polishing
- **Goal**: Refined interactions and mobile responsiveness.
- **Tasks**:
  - Add skeleton screens for loading states.
  - Implement "Swipe to Remove" for mobile.
  - Add "Purge Completed" button to toolbar.
