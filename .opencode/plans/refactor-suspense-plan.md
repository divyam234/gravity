# Refactor to Suspense & Loaders Plan

## Goal
Replace client-side loading states (Skeletons) with `useSuspenseQuery` and TanStack Router `loader` prefetching. This ensures data is ready before the route transitions, eliminating layout shift.

## 1. Refactor Hooks (`src/hooks/useAria2.ts`)
Currently, hooks hide the `queryKey` and `queryFn`. We need to expose **Query Options** so the Router can use them in `loader` functions.

-   **Action**: Refactor `useGlobalStat`, `useAllTasks`, etc., to export `queryOptions` objects.
-   **Example**:
    ```typescript
    export const globalStatOptions = (rpcUrl: string) => queryOptions({
      queryKey: ["aria2", "globalStat", rpcUrl],
      queryFn: () => aria2.getGlobalStat(),
      // ...
    });
    ```

## 2. Implement Route Loaders
The `loader` function runs *before* the component renders. We will use `queryClient.ensureQueryData` to fetch/cache data.

### A. Dashboard (`src/routes/index.tsx`)
-   **Loader**: Await `globalStat` and `allTasks`.
-   **Component**: Use `useSuspenseQuery(globalStatOptions)`. Remove `<Skeleton>` checks.

### B. Task Details (`src/routes/task.$gid.tsx`)
-   **Loader**: Await `taskStatus(gid)` and `taskFiles(gid)`.
-   **Component**: Use `useSuspenseQuery`. Remove `isStatusLoading` checks and Skeletons.

### C. Settings (`src/routes/settings.tsx`)
-   **Loader**: Await `globalOption`.

## 3. Global Suspense & Pending State
-   **Root (`src/routes/__root.tsx`)**:
    -   Since we are awaiting in loaders, navigation will "freeze" until data is ready.
    -   **Requirement**: Add a global navigation indicator (e.g., top progress bar) so the user knows something is happening.
    -   **Action**: Use `useRouter` state to show a progress bar when `isLoading` is true.

## 4. Cleanup
-   Remove `Skeleton` imports and usage from `StatsOverview`, `TaskList`, and `TaskDetailsPage`.
-   Remove `isLoading` props/checks.

## 5. Execution Order
1.  Modify `useAria2.ts` to export options.
2.  Update `__root.tsx` with a global spinner/bar.
3.  Refactor `index.tsx` (Dashboard).
4.  Refactor `task.$gid.tsx` (Details).
5.  Verify no Skeletons remain.
