import {
	queryOptions,
	useMutation,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "../lib/api";
import type { TaskStatus } from "../routes/tasks";

// --- Query Options Helpers ---

export const globalOptionOptions = () =>
	queryOptions({
		queryKey: ["gravity", "globalOption"],
		queryFn: () => api.getSettings(),
	});

export const globalStatOptions = () =>
	queryOptions({
		queryKey: ["gravity", "stats"],
		queryFn: () => api.getStats(),
	});

export const taskStatusOptions = (gid: string) =>
	queryOptions({
		queryKey: ["gravity", "status", gid],
		queryFn: () => api.getDownload(gid),
	});

export const taskFilesOptions = (gid: string) =>
	queryOptions({
		queryKey: ["gravity", "files", gid],
		queryFn: () => Promise.resolve([]),
	});

// --- Hooks ---

export function useGravityVersion() {
	return useQuery({
		queryKey: ["gravity", "version"],
		queryFn: () => api.getVersion(),
		staleTime: 0,
	});
}

export function useGlobalStat() {
	return useQuery({
		...globalStatOptions(),
		refetchInterval: (query) => (query.state.status === 'error' ? false : 30000), // Heartbeat only
	});
}

// Helper to fetch downloads
const fetchDownloads = async (status: string[], limit?: number, offset?: number) => {
	const res = await api.getDownloads({ status: status.length > 0 ? status : undefined, limit, offset });
	return res.data;
};

export function useTasksByStatus(status: TaskStatus, options?: { enabled?: boolean, limit?: number, offset?: number }) {
    const limit = options?.limit ?? 50;
    const offset = options?.offset ?? 0;
    
    return useQuery({
        queryKey: ["gravity", "downloads", status, limit, offset],
        queryFn: () => fetchDownloads([status], limit, offset),
        refetchInterval: (query) => (query.state.status === 'error' ? false : 30000), // Heartbeat only
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

export function useAllTasks(
	status: TaskStatus = "active",
) {
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
	});
}

export function useTaskOption(gid: string) {
	return useQuery({
		queryKey: ["gravity", "option", gid],
		queryFn: () => Promise.resolve({}), // Not supported yet
		enabled: !!gid,
	});
}

export function useTaskFiles(gid: string) {
	return useQuery({
		...taskFilesOptions(gid),
		enabled: !!gid,
	});
}

export function useTaskPeers(gid: string, enabled = false) {
	return useQuery({
		queryKey: ["gravity", "peers", gid],
		queryFn: () => Promise.resolve([]),
		enabled: !!gid && enabled,
	});
}

export function useTaskServers(gid: string, enabled = false) {
	return useQuery({
		queryKey: ["gravity", "servers", gid],
		queryFn: () => Promise.resolve([]),
		enabled: !!gid && enabled,
	});
}

export function useTaskStatus(gid: string) {
	return useQuery({
		...taskStatusOptions(gid),
		enabled: !!gid,
		refetchInterval: (query) => (query.state.status === 'error' ? false : 2000),
	});
}

// Mutations

export function useEngineActions() {
	const queryClient = useQueryClient();

	const invalidateTasks = () => {
		queryClient.refetchQueries({ queryKey: ["gravity", "downloads"] });
		queryClient.refetchQueries({ queryKey: ["gravity", "stats"] });
	};

	const addUri = useMutation({
		mutationFn: ({
			uris,
			options,
		}: {
			uris: string[];
			options?: Record<string, string>;
		}) => {
			return api.createDownload(uris[0], undefined, options?.out);
		},
		onSuccess: () => {
			invalidateTasks();
			toast.success("Download started");
		},
		onError: (err: Error) =>
			toast.error(`Failed to add download: ${err.message}`),
	});

	const addTorrent = useMutation({
		mutationFn: () => Promise.reject(new Error("Torrent upload not supported via API yet")),
	});

	const addMetalink = useMutation({
		mutationFn: () => Promise.reject(new Error("Metalink upload not supported via API yet")),
	});

	const pause = useMutation({
		mutationFn: (gid: string) => api.pauseDownload(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.info("Download paused");
		},
		onError: (err: Error) => toast.error(`Failed to pause: ${err.message}`),
	});

	const unpause = useMutation({
		mutationFn: (gid: string) => api.resumeDownload(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.success("Download resumed");
		},
		onError: (err: Error) => toast.error(`Failed to resume: ${err.message}`),
	});

	const remove = useMutation({
		mutationFn: (gid: string) => api.deleteDownload(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.info("Download removed");
		},
		onError: (err: Error) => toast.error(`Failed to remove: ${err.message}`),
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

	const retryTask = useMutation({
		mutationFn: (gid: string) => api.request('POST', `/downloads/${gid}/retry`),
		onSuccess: () => {
			invalidateTasks();
			toast.success("Retry triggered");
		},
		onError: (err: Error) => toast.error(`Failed to retry: ${err.message}`),
	});

	const changeGlobalOption = useMutation({
		mutationFn: (options: Record<string, string>) => {
            // Need to implement settings update in API if we want this to work.
            // Current API has /settings PATCH.
            // api.ts has updateSettings.
            return api.updateSettings(options);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["gravity", "globalOption"] });
            toast.success("Settings saved");
        },
        onError: (err: Error) => toast.error(`Failed to save settings: ${err.message}`),
	});

	const changeOption = useMutation({
		mutationFn: () => Promise.resolve(),
	});

	const restartAria2 = useMutation({
		mutationFn: () => api.restartAria2(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["gravity", "version"] });
			toast.success("Aria2 engine restarted");
		},
		onError: (err: Error) => toast.error(`Failed to restart Aria2: ${err.message}`),
	});

	const restartRclone = useMutation({
		mutationFn: () => api.restartRclone(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["gravity", "version"] });
			toast.success("Rclone engine restarted");
		},
		onError: (err: Error) => toast.error(`Failed to restart Rclone: ${err.message}`),
	});

	const restartServer = useMutation({
		mutationFn: () => api.restartServer(),
		onSuccess: () => {
			toast.success("Server restart triggered. Gravity will be back in a few seconds.");
		},
		onError: (err: Error) => toast.error(`Failed to restart server: ${err.message}`),
	});

	return {
		addUri,
		addTorrent,
		addMetalink,
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
