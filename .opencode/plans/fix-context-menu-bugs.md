# Plan: Fix Context Menu & Drag-Drop Bugs

I have identified similar bugs in the Task List's context menu and drag-and-drop interactions where state-dependent actions are not properly validated.

## Objectives
1.  **Fix Context Menu Removal**: Ensure "Remove" in the right-click menu works for finished/error tasks by using `removeDownloadResult` instead of `remove`.
2.  **Fix Context Menu State**: Disable or hide "Pause/Resume" options for tasks that are completed or in error.
3.  **Restrict Drag & Drop**: Disable reordering for "Stopped" (finished) tasks, as `changePosition` is not applicable to them.

## Execution Steps

### 1. `src/components/dashboard/TaskList.tsx`
-   **Import**: Add `removeDownloadResult` to `useAria2Actions` destructuring.
-   **Context Menu (Remove)**: Update the `onAction` handler for "remove" to check `task.status`.
    -   If `complete`, `error`, or `removed`: Call `removeDownloadResult.mutate(gid)`.
    -   Else: Call `remove.mutate(gid)`.
-   **Context Menu (Pause/Resume)**:
    -   Only render the `Dropdown.Item` for Pause/Resume if `task.status` is "active" or "paused".
    -   Alternatively, disable it for other states.
-   **Drag & Drop**:
    -   Conditionally pass `dragAndDropHooks` to `ListBox`.
    -   Only pass hooks if `status === "waiting"`. Active tasks *can* be reordered in some Aria2 configurations, but usually it's for the queue. Stopped tasks definitely cannot.
    -   Allow reordering for "active" and "waiting", but strictly disable for "stopped".

## Verification
-   Manual review of the code changes.
-   Type check to ensure `task.status` checks are valid.
