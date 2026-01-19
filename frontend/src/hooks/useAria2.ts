import {
	queryOptions,
	useMutation,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";
import { aria2 } from "../lib/aria2-rpc";
import { useSettingsStore } from "../store/useSettingsStore";

// --- Query Options ---

export const aria2VersionOptions = (rpcUrl: string) =>
	queryOptions({
		queryKey: ["aria2", "version", rpcUrl],
		queryFn: () => aria2.getVersion(),
		retry: false,
	});

export const globalStatOptions = (rpcUrl: string, pollingInterval: number) =>
	queryOptions({
		queryKey: ["aria2", "globalStat", rpcUrl],
		queryFn: () => aria2.getGlobalStat(),
		refetchInterval: pollingInterval,
		retry: false,
	});

export const activeTasksOptions = (rpcUrl: string, pollingInterval: number) =>
	queryOptions({
		queryKey: ["aria2", "tasks", "active", rpcUrl],
		queryFn: () => aria2.tellActive(),
		refetchInterval: pollingInterval,
		retry: false,
	});

export const waitingTasksOptions = (
	rpcUrl: string,
	pollingInterval: number,
	offset = 0,
	num = 100,
) =>
	queryOptions({
		queryKey: ["aria2", "tasks", "waiting", offset, num, rpcUrl],
		queryFn: () => aria2.tellWaiting(offset, num),
		refetchInterval: pollingInterval,
		retry: false,
	});

export const stoppedTasksOptions = (
	rpcUrl: string,
	pollingInterval: number,
	offset = 0,
	num = 100,
) =>
	queryOptions({
		queryKey: ["aria2", "tasks", "stopped", offset, num, rpcUrl],
		queryFn: () => aria2.tellStopped(offset, num),
		refetchInterval: pollingInterval,
		retry: false,
	});

export const uploadingTasksOptions = (
	rpcUrl: string,
	pollingInterval: number,
) =>
	queryOptions({
		queryKey: ["aria2", "tasks", "uploading", rpcUrl],
		queryFn: () => aria2.tellUploading(),
		refetchInterval: pollingInterval,
		retry: false,
	});

export const globalOptionOptions = (rpcUrl: string) =>
	queryOptions({
		queryKey: ["aria2", "globalOption", rpcUrl],
		queryFn: () => aria2.getGlobalOption(),
	});

export const taskStatusOptions = (rpcUrl: string, gid: string) =>
	queryOptions({
		queryKey: ["aria2", "status", gid, rpcUrl],
		queryFn: () => aria2.tellStatus(gid),
		refetchInterval: 2000,
	});

export const taskFilesOptions = (rpcUrl: string, gid: string) =>
	queryOptions({
		queryKey: ["aria2", "files", gid, rpcUrl],
		queryFn: () => aria2.getFiles(gid),
		refetchInterval: 2000,
	});

export const taskPeersOptions = (rpcUrl: string, gid: string) =>
	queryOptions({
		queryKey: ["aria2", "peers", gid, rpcUrl],
		queryFn: () => aria2.getPeers(gid),
		refetchInterval: 3000,
	});

export const taskServersOptions = (rpcUrl: string, gid: string) =>
	queryOptions({
		queryKey: ["aria2", "servers", gid, rpcUrl],
		queryFn: () => aria2.getServers(gid),
		refetchInterval: 5000,
	});

// --- Hooks ---

export function useAria2Version() {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		...aria2VersionOptions(rpcUrl),
		enabled: !!rpcUrl,
	});
}

export function useGlobalStat() {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		...globalStatOptions(rpcUrl, pollingInterval),
		enabled: !!rpcUrl,
	});
}

export function useActiveTasks(options?: { enabled?: boolean }) {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		...activeTasksOptions(rpcUrl, pollingInterval),
		enabled: !!rpcUrl && (options?.enabled ?? true),
	});
}

export function useWaitingTasks(
	offset = 0,
	num = 100,
	options?: { enabled?: boolean },
) {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		...waitingTasksOptions(rpcUrl, pollingInterval, offset, num),
		enabled: !!rpcUrl && (options?.enabled ?? true),
	});
}

export function useStoppedTasks(
	offset = 0,
	num = 100,
	options?: { enabled?: boolean },
) {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		...stoppedTasksOptions(rpcUrl, pollingInterval, offset, num),
		enabled: !!rpcUrl && (options?.enabled ?? true),
	});
}

export function useUploadingTasks(options?: { enabled?: boolean }) {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		...uploadingTasksOptions(rpcUrl, pollingInterval),
		enabled: !!rpcUrl && (options?.enabled ?? true),
	});
}

export function useAllTasks(
	status: "active" | "waiting" | "stopped" | "uploading" = "active",
) {
	const active = useActiveTasks({ enabled: status === "active" });
	const uploading = useUploadingTasks({ enabled: status === "uploading" });
	const waiting = useWaitingTasks(0, 50, { enabled: status === "waiting" });
	const stopped = useStoppedTasks(0, 50, { enabled: status === "stopped" });

	return {
		active: active.data || [],
		uploading: uploading.data || [],
		waiting: waiting.data || [],
		stopped: stopped.data || [],
		isLoading:
			active.isLoading ||
			waiting.isLoading ||
			stopped.isLoading ||
			uploading.isLoading,
		refetch: () => {
			if (status === "active") active.refetch();
			if (status === "uploading") uploading.refetch();
			if (status === "waiting") waiting.refetch();
			if (status === "stopped") stopped.refetch();
		},
	};
}

export function useGlobalOption() {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		...globalOptionOptions(rpcUrl),
		enabled: !!rpcUrl,
	});
}

export function useTaskOption(gid: string) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "option", gid, rpcUrl],
		queryFn: () => aria2.getOption(gid),
		enabled: !!rpcUrl && !!gid,
	});
}

export function useTaskFiles(gid: string) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		...taskFilesOptions(rpcUrl, gid),
		enabled: !!rpcUrl && !!gid,
	});
}

export function useTaskPeers(gid: string, enabled = false) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		...taskPeersOptions(rpcUrl, gid),
		enabled: !!gid && enabled,
	});
}

export function useTaskServers(gid: string, enabled = false) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		...taskServersOptions(rpcUrl, gid),
		enabled: !!gid && enabled,
	});
}

export function useTaskStatus(gid: string) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		...taskStatusOptions(rpcUrl, gid),
		enabled: !!rpcUrl && !!gid,
	});
}

// Mutations

export function useAria2Actions() {
	const queryClient = useQueryClient();

	const invalidateTasks = () => {
		queryClient.invalidateQueries({ queryKey: ["aria2", "tasks"] });
		queryClient.invalidateQueries({ queryKey: ["aria2", "globalStat"] });
	};

	const addUri = useMutation({
		mutationFn: ({
			uris,
			options,
		}: {
			uris: string[];
			options?: Record<string, string>;
		}) => aria2.addUri(uris, options),
		onSuccess: () => {
			invalidateTasks();
			toast.success("Download started");
		},
		onError: (err: Error) => toast.error(`Failed to add download: ${err.message}`),
	});

	const addTorrent = useMutation({
		mutationFn: ({
			torrent,
			uris,
			options,
		}: {
			torrent: string;
			uris?: string[];
			options?: Record<string, string>;
		}) => aria2.addTorrent(torrent, uris, options),
		onSuccess: () => {
			invalidateTasks();
			toast.success("Torrent added");
		},
		onError: (err: Error) => toast.error(`Failed to add torrent: ${err.message}`),
	});

	const addMetalink = useMutation({
		mutationFn: ({
			metalink,
			options,
		}: {
			metalink: string;
			options?: Record<string, string>;
		}) => aria2.addMetalink(metalink, options),
		onSuccess: () => {
			invalidateTasks();
			toast.success("Metalink added");
		},
		onError: (err: Error) => toast.error(`Failed to add metalink: ${err.message}`),
	});

	const pause = useMutation({
		mutationFn: (gid: string) => aria2.pause(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.info("Download paused");
		},
		onError: (err: Error) => toast.error(`Failed to pause: ${err.message}`),
	});

	const unpause = useMutation({
		mutationFn: (gid: string) => aria2.unpause(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.success("Download resumed");
		},
		onError: (err: Error) => toast.error(`Failed to resume: ${err.message}`),
	});

	const remove = useMutation({
		mutationFn: (gid: string) => aria2.remove(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.info("Download removed");
		},
		onError: (err: Error) => toast.error(`Failed to remove: ${err.message}`),
	});

	const forceRemove = useMutation({
		mutationFn: (gid: string) => aria2.forceRemove(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.info("Download force removed");
		},
		onError: (err: Error) => toast.error(`Failed to force remove: ${err.message}`),
	});

	const removeDownloadResult = useMutation({
		mutationFn: (gid: string) => aria2.removeDownloadResult(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.info("Task removed from list");
		},
		onError: (err: Error) => toast.error(`Failed to remove task: ${err.message}`),
	});

	const purgeDownloadResult = useMutation({
		mutationFn: () => aria2.purgeDownloadResult(),
		onSuccess: () => {
			invalidateTasks();
			toast.info("Finished tasks purged");
		},
		onError: (err: Error) => toast.error(`Failed to purge tasks: ${err.message}`),
	});

	const saveSession = useMutation({
		mutationFn: () => aria2.saveSession(),
	});

	const retryTask = useMutation({
		mutationFn: (gid: string) => aria2.retryTask(gid),
		onSuccess: () => {
			invalidateTasks();
			toast.success("Retry triggered");
		},
		onError: (err: Error) => toast.error(`Failed to retry: ${err.message}`),
	});

	const changeGlobalOption = useMutation({
		mutationFn: (options: Record<string, string>) =>
			aria2.changeGlobalOption(options),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["aria2", "globalOption"] });
		},
	});

	const changeOption = useMutation({
		mutationFn: ({
			gid,
			options,
		}: {
			gid: string;
			options: Record<string, string>;
		}) => aria2.changeOption(gid, options),
		onSuccess: (_, { gid }) => {
			queryClient.invalidateQueries({ queryKey: ["aria2", "status", gid] });
		},
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
