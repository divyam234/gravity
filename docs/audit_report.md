# Codebase Audit Report

## 1. Task Updation & Concurrency

### Race Conditions in Queue Processing
In `DownloadService.ProcessQueue`:
- **Issue:** The check for `activeCount < maxConcurrent` happens inside a lock, but the lock is released *before* calling `engine.Add` and updating the task status to `Active`.
- **Impact:** If `ProcessQueue` is triggered concurrently (e.g., via HTTP request and Event Bus simultaneously), multiple routines could read the same `activeCount`, see available slots, and submit tasks, effectively exceeding `maxConcurrent` limits.
- **Recommendation:** Use a "pending/allocating" state or keep the tasks locked (or semaphored) until `engine.Add` returns or is committed.

### State Synchronization
In `DownloadService.Sync`:
- **Issue:** `Sync` iterates active tasks in DB and checks against Engine. If missing in Engine, it moves them to `Waiting`.
- **Risk:** If the Engine is slow to list tasks (e.g. Aria2 RPC latency), `Sync` might incorrectly assume tasks are missing if the list is partial or times out, causing a loop of restart attempts.
- **Mitigation:** The current logic seems robust enough by checking `gidMap` presence, but relies on `engine.List` returning a *complete* list.

## 2. Global Progress Calculation

### Speed Aggregation (`StatsService`)
- **Implementation:** Uses `sync.Map` (`taskSpeeds`) to store the last reported speed of active tasks. Sums them up for global speed.
- **Inconsistency:** When a download completes, it is immediately removed from `taskSpeeds`. This causes a sudden drop in global speed to 0 (or lower) instantly, even if the transfer was saturating bandwidth milliseconds ago.
- **UX Impact:** Global speed graphs will look jagged/spiky rather than smooth.
- **Recommendation:** Implement a sliding window or decay for speed reporting, or retain the last speed value for a few seconds after completion.

### Session Stats Accuracy
- **Issue:** `TotalDownloaded` in `StatsRepo` appears to be incremented only on `DownloadCompleted` event (lines 90-93 in `stats.go`).
- **Impact:** Data downloaded by *active* (large/slow) torrents is not counted towards total stats until they finish. If a user downloads 50GB of a 100GB torrent and restarts the server, that 50GB might not be recorded in "Total Downloaded" history if persistence is only on completion.
- **Recommendation:** Periodically persist incremental `Downloaded` bytes for active tasks to the `StatsRepo`, or sum `d.Downloaded` of active tasks when querying current stats.

## 3. Single Progress Calculation

### Progress Precision
- **File Progress:** Calculated as `int((downloaded * 100) / size)`. This floor integer division loses precision (e.g., 99.9% shows as 99%).
- **Task Progress:** The `Download` model relies on `Downloaded` vs `Size` bytes.
- **Bug in ETA:** In `updateAggregateProgress`, if `d.Speed` is 0 (momentary drop), `ETA` is set to 0. This is misleading (looks like "done"). It should probably be -1 or "infinity".

### Metadata Fetching
- **Native Engine:** Uses a dedicated goroutine to wait for `GotInfo()` channel. If it times out, it calls `Remove`.
- **Bug:** If `Remove` is called, the task in `DownloadService` might remain in `Active` (metadata fetching) state because `onError` callback might not trigger a state transition to `Error` if the engine just silently removes it (though `NativeEngine` calls `onError` before remove).

## 4. Context & Resource Management (Recently Fixed)

- **Previous State:** Widespread use of `context.Background()` in services (`Download`, `Upload`, `Stats`) and Engines (`Native`, `Aria2`) caused goroutine leaks and inability to gracefully shut down background workers.
- **Current State:** Refactored `App`, `Service`, and `Engine` layers to propagate a lifecycle `context.Context`.
    - `Start(ctx)` pattern implemented across services.
    - Providers now configure synchronously with timeouts (`Configure(ctx, ...)`).
    - `DirectProvider` reuses HTTP client to fix connection churn.

## 5. Provider Inconsistencies

- **Direct Provider:** Now mimics `rclone` logic strictly (HEAD/GET range 0-0). This is good for compatibility but might fail on servers that don't support Range requests (fallback to full download was removed/optimized out).
- **MegaDebrid:** Relies on a mutex-protected token. If multiple concurrent requests trigger token refresh, they might serialize efficiently, but error handling for invalid tokens triggers a recursive `Resolve` call which could theoretically loop if the token remains invalid.

## 6. Recommendations

1.  **Fix Race in ProcessQueue:** Introduce a robust reservation system for slots.
2.  **Real-time Stats Persistence:** Flush active download bytes to stats DB periodically, not just on completion.
3.  **Smooth Speed calculation:** Implement a moving average for global speed.
4.  **Integration Tests:** Add tests specifically for the `Sync` logic to ensure it recovers correctly from engine crashes.
