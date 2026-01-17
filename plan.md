# Aria2c RPC Control Panel - Implementation Plan

## 1. Project Overview
A modern, high-performance web interface for `aria2c` download manager.
- **Goal**: Provide a "Best in Class" UI with real-time updates, comprehensive settings, and a beautiful polished design.
- **Package Manager**: `bun`

## 2. Architecture & Tech Stack
- **Frontend Framework**: React 19 (Vite)
- **Language**: TypeScript
- **Styling**: Tailwind CSS v4
- **UI Component Library**: HeroUI v3 (Beta)
- **State Management**: `@tanstack/react-query` (Polling RPC state)
- **Routing**: `@tanstack/react-router`
- **RPC Client**: Custom lightweight TypeScript wrapper for JSON-RPC 2.0.

## 3. UI Design System
- **Theme**: Dark/Light mode support (HeroUI built-in).
- **Layout**: 
  - **Sidebar/Navbar**: Navigation & Global Stats.
  - **Main Content**: `Card`-based grid/list for tasks.
  - **Modals**: For complex interactions (Add Task, Settings).
- **Key Components**:
  - `Card`: Task items.
  - `ProgressBar`: Custom Tailwind component (Gradient fill).
  - `Chip`: Status indicators.
  - `Tabs`: Filtering and Settings navigation.

## 4. Comprehensive Settings Map
Settings will be divided into categories. "Read-only" settings are startup flags that cannot be changed via RPC.

### A. General
| Setting | Key | Type | Modifiable? |
| :--- | :--- | :--- | :--- |
| Download Directory | `dir` | Path | ✅ |
| Max Concurrent Downloads | `max-concurrent-downloads` | Number | ✅ |
| Check Integrity | `check-integrity` | Boolean | ✅ |
| Continue | `continue` | Boolean | ✅ |
| Log File | `log` | Path | ❌ (Startup only) |
| Log Level | `log-level` | Enum | ✅ |

### B. Connection (HTTP/FTP)
| Setting | Key | Type | Modifiable? |
| :--- | :--- | :--- | :--- |
| Max Connections/Server | `max-connection-per-server` | Number | ✅ |
| Split | `split` | Number | ✅ |
| Connect Timeout | `connect-timeout` | Seconds | ✅ |
| Lowest Speed Limit | `lowest-speed-limit` | Speed | ✅ |
| Proxy | `all-proxy` | URL | ✅ |
| Proxy User | `all-proxy-user` | String | ✅ |
| Proxy Password | `all-proxy-passwd` | String | ✅ |
| User Agent | `user-agent` | String | ✅ |

### C. BitTorrent
| Setting | Key | Type | Modifiable? |
| :--- | :--- | :--- | :--- |
| Enable DHT | `enable-dht` | Boolean | ✅ |
| Seed Ratio | `seed-ratio` | Decimal | ✅ |
| Max Upload Limit | `max-upload-limit` | Speed | ✅ |
| Listen Port | `listen-port` | Number | ❌ (Startup only) |
| Peer Exchange | `enable-peer-exchange` | Boolean | ✅ |
| Follow Torrent | `follow-torrent` | Boolean | ✅ |

### D. RPC & Security (Read-Only Info)
| Setting | Key | Type | Modifiable? |
| :--- | :--- | :--- | :--- |
| RPC Listen Port | `rpc-listen-port` | Number | ❌ |
| RPC Secret | `rpc-secret` | String | ❌ |
| Secure | `rpc-secure` | Boolean | ❌ |

### E. Advanced
| Setting | Key | Type | Modifiable? |
| :--- | :--- | :--- | :--- |
| File Allocation | `file-allocation` | Enum | ❌ |
| Disk Cache | `disk-cache` | Size | ❌ |
| Save Session | `save-session` | Path | ❌ |
| Auto Save Interval | `auto-save-interval` | Seconds | ✅ |

### F. UI Controls (App Preferences)
These are stored in `localStorage`, not `aria2c`.
- **Theme**: Light / Dark / System.
- **Refresh Rate**: 1s / 2s / 5s / Manual.
- **Notifications**: Enable/Disable toast notifications on complete.
- **Language**: English (expandable).

## 5. Implementation Phases

### Phase 1: Setup & Dependencies
- [ ] Install HeroUI v3 Beta packages using `bun`.
- [ ] Install `tailwind-variants`, `clsx`, `tailwind-merge`.
- [ ] Configure `tailwind.css` for HeroUI v3 compatibility.
- [ ] Verify build environment.

### Phase 2: Core Infrastructure
- [ ] Create `src/lib/aria2-rpc.ts`: Strongly typed JSON-RPC client.
- [ ] Create `src/hooks/useAria2.ts`: React Query hooks (`useGlobalStat`, `useTasks`, `useOption`).
- [ ] Create `src/store/ui-store.ts`: For UI preferences (Theme, Refresh Rate).

### Phase 3: Base UI Components
- [ ] `ProgressBar`: Custom component with animation.
- [ ] `SpeedGraph`: Simple sparkline for speed history.
- [ ] `StatusChip`: Mapping `aria2` status to HeroUI colors.
- [ ] `Layout`: App shell with Navbar.

### Phase 4: Dashboard & Task List
- [ ] `StatsOverview`: Cards for Global Download/Upload speed.
- [ ] `TaskList`: Virtualized or paginated list of downloads.
- [ ] `DownloadRow`: Detailed card for a single task (Progress, Speed, Actions).
- [ ] Implement Pause/Resume/Remove actions.

### Phase 5: Features & Modals
- [ ] `AddDownloadModal`:
  - URI Input (Textarea for multiple).
  - File/Torrent upload.
  - Advanced options accordion.
- [ ] `SettingsModal`:
  - Tabs for categories.
  - Forms for all settings defined in Section 4.
  - Read-only states for immutable settings.

### Phase 6: Polish & Verification
- [ ] Error handling (RPC connection failure).
- [ ] Empty states (No downloads).
- [ ] Responsive check (Mobile view).
- [ ] Build & Type check.

## 6. Commands
```bash
# Install
bun add @heroui/react@beta @heroui/styles@beta tailwind-variants clsx tailwind-merge framer-motion

# Dev
bun run dev
```
