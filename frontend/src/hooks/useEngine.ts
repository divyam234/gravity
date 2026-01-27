import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";
import { openapi } from "../lib/openapi";
import type { TaskStatus } from "../routes/tasks";

export const globalOptionOptions = () =>
  openapi.queryOptions("get", "/settings");

export const globalStatOptions = () => openapi.queryOptions("get", "/stats");

export const taskStatusOptions = (gid: string) =>
  openapi.queryOptions("get", "/downloads/{id}", {
    params: { path: { id: gid } },
  });

export const taskFilesOptions = (gid: string) =>
  queryOptions({
    queryKey: ["gravity", "files", gid],
    queryFn: () => Promise.resolve([]),
  });

// --- Hooks ---

export function useGravityVersion() {
  return useQuery({
    ...openapi.queryOptions("get", "/system/version"),
    staleTime: 0,
    select: (res) => res.data,
  });
}

export function useGlobalStat() {
  return useQuery({
    ...globalStatOptions(),
    select: (res) => res.data,
    refetchInterval: (query) =>
      query.state.status === "error" ? false : 30000, // Heartbeat only
  });
}

export function useTasksByStatus(
  status: TaskStatus,
  options?: { enabled?: boolean; limit?: number; offset?: number },
) {
  const limit = options?.limit ?? 50;
  const offset = options?.offset ?? 0;

  return useQuery({
    ...openapi.queryOptions("get", "/downloads", {
      params: {
        query: {
          status,
          limit,
          offset,
        },
      },
    }),
    select: (res) => res.data || [],
    refetchInterval: (query) =>
      query.state.status === "error" ? false : 30000, // Heartbeat only
    enabled: options?.enabled ?? true,
  });
}

export function useActiveTasks(options?: { enabled?: boolean }) {
  return useTasksByStatus("active", options);
}

export function useWaitingTasks(options?: { enabled?: boolean }) {
  return useTasksByStatus("waiting", options);
}

export function usePausedTasks(options?: { enabled?: boolean }) {
  return useTasksByStatus("paused", options);
}

export function useUploadingTasks(options?: { enabled?: boolean }) {
  return useTasksByStatus("uploading", options);
}

export function useCompletedTasks(options?: { enabled?: boolean }) {
  return useTasksByStatus("complete", options);
}

export function useErrorTasks(options?: { enabled?: boolean }) {
  return useTasksByStatus("error", options);
}

export function useAllTasks(status: TaskStatus = "active") {
  const active = useActiveTasks({ enabled: status === "active" });
  const waiting = useWaitingTasks({ enabled: status === "waiting" });
  const uploading = useUploadingTasks({ enabled: status === "uploading" });
  const paused = usePausedTasks({ enabled: status === "paused" });
  const complete = useCompletedTasks({ enabled: status === "complete" });
  const error = useErrorTasks({ enabled: status === "error" });

  return {
    active: active.data || [],
    waiting: waiting.data || [],
    uploading: uploading.data || [],
    paused: paused.data || [],
    complete: complete.data || [],
    error: error.data || [],
    isLoading:
      active.isLoading ||
      waiting.isLoading ||
      uploading.isLoading ||
      paused.isLoading ||
      complete.isLoading ||
      error.isLoading,
    refetch: () => {
      if (status === "active") active.refetch();
      if (status === "waiting") waiting.refetch();
      if (status === "uploading") uploading.refetch();
      if (status === "paused") paused.refetch();
      if (status === "complete") complete.refetch();
      if (status === "error") error.refetch();
    },
  };
}

export function useGlobalOption() {
  return useQuery({
    ...globalOptionOptions(),
    select: (res) => res.data,
  });
}

export function useTaskFiles(gid: string) {
  return useQuery({
    ...taskFilesOptions(gid),
    enabled: !!gid,
  });
}

export function useTaskStatus(gid: string) {
  return useQuery({
    ...taskStatusOptions(gid),
    enabled: !!gid,
    select: (res) => res.data,
    refetchInterval: (query) => (query.state.status === "error" ? false : 2000),
  });
}

// Mutations

export function useEngineActions() {
  const queryClient = useQueryClient();

  const invalidateTasks = () => {
    queryClient.refetchQueries({
      queryKey: openapi.queryOptions("get", "/downloads").queryKey,
    });
    queryClient.refetchQueries({
      queryKey: openapi.queryOptions("get", "/stats").queryKey,
    });
  };

  const addUri = openapi.useMutation("post", "/downloads", {
    onSuccess: () => {
      invalidateTasks();
      toast.success("Download started");
    },
    onError: (err) => toast.error(`Failed to add download: ${err.error}`),
  });

  const pause = openapi.useMutation("post", "/downloads/{id}/pause", {
    onSuccess: () => {
      invalidateTasks();
      toast.info("Download paused");
    },
    onError: (err) => toast.error(`Failed to pause: ${err.error}`),
  });

  const unpause = openapi.useMutation("post", "/downloads/{id}/resume", {
    onSuccess: () => {
      invalidateTasks();
      toast.success("Download resumed");
    },
    onError: (err) => toast.error(`Failed to resume: ${err.error}`),
  });

  const remove = openapi.useMutation("delete", "/downloads/{id}", {
    onSuccess: () => {
      invalidateTasks();
      toast.info("Download removed");
    },
    onError: (err) => toast.error(`Failed to remove: ${err.error}`),
  });

  const forceRemove = remove;
  const removeDownloadResult = remove;

  const purgeDownloadResult = useMutation({
    mutationFn: () => Promise.resolve(),
    onSuccess: () => {
      invalidateTasks();
      toast.info("Finished tasks purged (Not implemented)");
    },
  });

  const saveSession = useMutation({
    mutationFn: () => Promise.resolve(),
  });

  const retryTask = openapi.useMutation("post", "/downloads/{id}/retry", {
    onSuccess: () => {
      invalidateTasks();
      toast.success("Retry triggered");
    },
    onError: (err) => toast.error(`Failed to retry: ${err.error}`),
  });

  const changeGlobalOption = openapi.useMutation("patch", "/settings", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/settings").queryKey,
      });
      toast.success("Settings saved");
    },
    onError: (err) => toast.error(`Failed to save settings: ${err.error}`),
  });

  const changeOption = useMutation({
    mutationFn: () => Promise.resolve(),
  });

  const restartAria2 = openapi.useMutation("post", "/system/restart/aria2", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/system/version").queryKey,
      });
      toast.success("Aria2 engine restarted");
    },
    onError: (err) => toast.error(`Failed to restart Aria2: ${err.error}`),
  });

  const restartRclone = openapi.useMutation("post", "/system/restart/rclone", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/system/version").queryKey,
      });
      toast.success("Rclone engine restarted");
    },
    onError: (err) => toast.error(`Failed to restart Rclone: ${err.error}`),
  });

  const restartServer = openapi.useMutation("post", "/system/restart/server", {
    onSuccess: () => {
      toast.success(
        "Server restart triggered. Gravity will be back in a few seconds.",
      );
    },
  });

  return {
    addUri,
    pause,
    unpause,
    remove,
    forceRemove,
    removeDownloadResult,
    purgeDownloadResult,
    saveSession,
    retryTask,
    changeGlobalOption,
    changeOption,
    restartAria2,
    restartRclone,
    restartServer,
  };
}
