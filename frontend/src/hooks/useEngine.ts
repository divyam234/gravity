import {
	queryOptions,
	useMutation,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "../lib/api";
import { useSettingsStore } from "../store/useSettingsStore";

// --- Query Options Helpers ---

export const globalOptionOptions = () =>
	queryOptions({
		queryKey: ["gravity", "globalOption"],
		queryFn: () => api.getSettings(),
	});

export const globalStatOptions = () =>
	queryOptions({
		queryKey: ["gravity", "stats"],
		queryFn: async () => {
			const stats = await api.getStats();
			return {
				numActive: stats.active.downloads + stats.active.uploads,
				numWaiting: stats.queue.pending + stats.queue.paused,
				numStopped: stats.totals.completed + stats.totals.failed,
				downloadSpeed: stats.active.downloadSpeed,
				uploadSpeed: stats.active.uploadSpeed,
			};
		},
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
	});
}

export function useGlobalStat() {
	const { pollingInterval } = useSettingsStore();
	return useQuery({
		...globalStatOptions(),
		refetchInterval: pollingInterval,
	});
}

// Helper to fetch downloads
const fetchDownloads = async (status: string[], limit?: number, offset?: number) => {
	const res = await api.getDownloads({ status: status.length > 0 ? status : undefined, limit, offset });
	return res.data;
};

export function useActiveTasks(options?: { enabled?: boolean }) {
	const { pollingInterval } = useSettingsStore();
	return useQuery({
		queryKey: ["gravity", "downloads", "active"],
		queryFn: () => fetchDownloads(['downloading', 'uploading', 'pending']),
		refetchInterval: pollingInterval,
		enabled: options?.enabled ?? true,
	});
}

export function useWaitingTasks(
	offset = 0,
	num = 100,
	options?: { enabled?: boolean },
) {
	const { pollingInterval } = useSettingsStore();
	return useQuery({
		queryKey: ["gravity", "downloads", "waiting", offset, num],
		queryFn: () => fetchDownloads(['paused'], num, offset),
		refetchInterval: pollingInterval,
		enabled: options?.enabled ?? true,
	});
}

export function useStoppedTasks(
	offset = 0,
	num = 100,
	options?: { enabled?: boolean },
) {
	const { pollingInterval } = useSettingsStore();
	return useQuery({
		queryKey: ["gravity", "downloads", "stopped", offset, num],
		queryFn: () => fetchDownloads(['complete'], num, offset),
		refetchInterval: pollingInterval,
		enabled: options?.enabled ?? true,
	});
}

export function useUploadingTasks(options?: { enabled?: boolean }) {
	const { pollingInterval } = useSettingsStore();
	return useQuery({
		queryKey: ["gravity", "downloads", "uploading"],
		queryFn: () => fetchDownloads(['uploading']),
		refetchInterval: pollingInterval,
		enabled: options?.enabled ?? true,
	});
}

export function useFailedTasks(
	offset = 0,
	num = 100,
	options?: { enabled?: boolean },
) {
	const { pollingInterval } = useSettingsStore();
	return useQuery({
		queryKey: ["gravity", "downloads", "failed", offset, num],
		queryFn: () => fetchDownloads(['error'], num, offset),
		refetchInterval: pollingInterval,
		enabled: options?.enabled ?? true,
	});
}

export function useRecentDownloads(limit = 5) {
	const { pollingInterval } = useSettingsStore();
	return useQuery({
		queryKey: ["gravity", "downloads", "recent", limit],
		queryFn: () => fetchDownloads([], limit),
		refetchInterval: pollingInterval,
	});
}

export function useAllTasks(
	status: "active" | "waiting" | "stopped" | "uploading" | "failed" = "active",
) {
	const active = useActiveTasks({ enabled: status === "active" });
	const uploading = useUploadingTasks({ enabled: status === "uploading" });
	const waiting = useWaitingTasks(0, 50, { enabled: status === "waiting" });
	const stopped = useStoppedTasks(0, 50, { enabled: status === "stopped" });
	const failed = useFailedTasks(0, 50, { enabled: status === "failed" });

	return {
		active: active.data || [],
		uploading: uploading.data || [],
		waiting: waiting.data || [],
		stopped: stopped.data || [],
		failed: failed.data || [],
		isLoading:
			active.isLoading ||
			waiting.isLoading ||
			stopped.isLoading ||
			uploading.isLoading ||
			failed.isLoading,
		refetch: () => {
			if (status === "active") active.refetch();
			if (status === "uploading") uploading.refetch();
			if (status === "waiting") waiting.refetch();
			if (status === "stopped") stopped.refetch();
			if (status === "failed") failed.refetch();
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
		refetchInterval: 2000,
	});
}

// Mutations

export function useEngineActions() {
	const queryClient = useQueryClient();

	const invalidateTasks = () => {
		queryClient.invalidateQueries({ queryKey: ["gravity", "downloads"] });
		queryClient.invalidateQueries({ queryKey: ["gravity", "stats"] });
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
	};
}