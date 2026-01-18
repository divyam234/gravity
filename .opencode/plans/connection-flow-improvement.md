# Plan: Improve Connection Flow & UI Non-Blocking Shell

The current application blocks the entire UI with a "Welcome" or "Connection Failed" overlay if aria2c is not reachable. This prevents users from accessing settings or browsing the UI until a connection is established. This plan details how to transition to an "Always-Visible Shell" pattern.

## Goals
- Show the full application shell (Sidebar, Header, Layout) immediately on startup.
- Prevent route loaders from blocking the UI while waiting for RPC responses.
- Provide real-time, non-intrusive feedback about connection status in the Header.
- Guide new users to configuration via Alerts rather than full-page overlays.

## Proposed Changes

### 1. `src/routes/__root.tsx`
- **Remove `ConnectionErrorOverlay`**: This component currently covers the entire screen.
- **Update `RootComponent`**: Remove the `showOverlay` logic. Always render `<Outlet />` inside `<Layout />`.
- **Cleanup**: Remove unused imports related to the overlay (IconGear, IconLink, IconShieldKeyhole, etc.).

### 2. Route Loaders (`src/routes/index.tsx`, `src/routes/tasks.tsx`)
- **Remove `await` from `ensureQueryData`**: Currently, the router waits for these calls to resolve/fail before showing the page.
- **Switch to `prefetchQuery`**: Use `queryClient.prefetchQuery` instead of `ensureQueryData` and do **not** await it. This allows the component to mount immediately while fetching data in the background.

### 3. `src/components/Layout.tsx`
- **Add "No Server" Alert**: If `rpcUrl` is missing (new user), show a prominent but non-blocking Alert directing them to "Connection Settings".
- **Improve Connection Status Chip**: (Already partially done) Ensure the pulse/loader state is clear and consistent.
- **Sidebar Integration**: The sidebar is already always visible (added in previous turn), but ensure it stays interactive even during errors.

### 4. `src/components/dashboard/settings/ConnectionSettings.tsx`
- (Optional) Ensure the "Test Connection" and "Save" flows are robust, as this becomes the primary landing spot for new users.

## Verification Plan
- Run `bun run typecheck` to ensure no broken imports or props.
- Manual check: Clear localStorage/settings to simulate a new user and verify the shell appears with a configuration alert.
- Manual check: Point to a fake RPC URL and verify the UI stays interactive while showing the "Offline" status.
