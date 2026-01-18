# Plan: Final UI Polish & Bug Sweep

This plan focuses on a final comprehensive review of the UI to ensure consistency, remove regressions, and polish the user experience before the "final check" is considered complete.

## Objectives
1.  **Consistency Check**: Ensure the "solid, high-contrast" design language (no glassiness) is applied to all remaining components, including modals, popovers, and secondary pages.
2.  **Error State Review**: Verify that error states (e.g., failed downloads, network errors) are handled gracefully and consistently with the new `Alert` styles.
3.  **Loading State Review**: Check if loading skeletons or spinners are present and consistent where `Suspense` was removed (e.g., inside `TaskList`).
4.  **Code Cleanup**: Remove any `console.log` statements, unused imports, or legacy comments.

## Execution Steps

### 1. UI Consistency Audit
-   **Files**: `src/components/dashboard/DownloadCard.tsx`, `src/components/dashboard/settings/*.tsx`, `src/routes/settings.$category.tsx`.
-   **Action**: Scan for `backdrop-blur`, `bg-opacity`, or `glass` classes and replace them with solid `bg-background` or `bg-default/5` as per the new design system.
-   **Action**: Ensure border radii and padding are consistent (`rounded-2xl`, `rounded-3xl` vs `rounded-md`).

### 2. Loading State Verification
-   **Files**: `src/components/dashboard/TaskList.tsx`.
-   **Action**: Since `useSuspenseQuery` was replaced with `useQuery`, the `TaskList` might render empty or `null` while loading. I need to verify if it shows a spinner or skeleton.
-   **Refinement**: If missing, add a simple skeleton loader to `TaskList` to prevent layout shifts.

### 3. Settings Polish
-   **Files**: `src/components/dashboard/settings/SettingField.tsx`.
-   **Action**: Ensure input fields in settings match the new `TextField` style used in `AddDownloadPage`.

### 4. Final Code Cleanup
-   **Action**: Run `grep` for `console.log` and unused imports.
-   **Action**: Run `bun run typecheck` one last time.

## Verification
-   Manual review of component code.
-   Type check confirmation.
