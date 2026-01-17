import {
	queryOptions,
	useMutation,
	useQuery,
	useQueryClient,
	useSuspenseQuery,
} from "@tanstack/react-query";
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
	return useSuspenseQuery(aria2VersionOptions(rpcUrl));
}

export function useGlobalStat() {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useSuspenseQuery(globalStatOptions(rpcUrl, pollingInterval));
}

export function useActiveTasks() {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useSuspenseQuery(activeTasksOptions(rpcUrl, pollingInterval));
}

export function useWaitingTasks(offset = 0, num = 100) {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useSuspenseQuery(
		waitingTasksOptions(rpcUrl, pollingInterval, offset, num),
	);
}

export function useStoppedTasks(offset = 0, num = 100) {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useSuspenseQuery(
		stoppedTasksOptions(rpcUrl, pollingInterval, offset, num),
	);
}

export function useAllTasks() {
	const active = useActiveTasks();
	const waiting = useWaitingTasks(0, 50);
	const stopped = useStoppedTasks(0, 50);

	return {
		active: active.data || [],
		waiting: waiting.data || [],
		stopped: stopped.data || [],
		refetch: () => {
			// Suspense queries handle refetching internally through options,
			// but we can expose manual refetch if needed.
		},
	};
}

export function useGlobalOption() {
	const { rpcUrl } = useSettingsStore();
	return useSuspenseQuery(globalOptionOptions(rpcUrl));
}

export function useTaskOption(gid: string) {
	const { rpcUrl } = useSettingsStore();
	return useSuspenseQuery({
		queryKey: ["aria2", "option", gid, rpcUrl],
		queryFn: () => aria2.getOption(gid),
	});
}

export function useTaskFiles(gid: string) {
	const { rpcUrl } = useSettingsStore();
	return useSuspenseQuery(taskFilesOptions(rpcUrl, gid));
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
	return useSuspenseQuery(taskStatusOptions(rpcUrl, gid));
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
		onSuccess: invalidateTasks,
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
		onSuccess: invalidateTasks,
	});

	const addMetalink = useMutation({
		mutationFn: ({
			metalink,
			options,
		}: {
			metalink: string;
			options?: Record<string, string>;
		}) => aria2.addMetalink(metalink, options),
		onSuccess: invalidateTasks,
	});

	const pause = useMutation({
		mutationFn: (gid: string) => aria2.pause(gid),
		onSuccess: invalidateTasks,
	});

	const unpause = useMutation({
		mutationFn: (gid: string) => aria2.unpause(gid),
		onSuccess: invalidateTasks,
	});

	const remove = useMutation({
		mutationFn: (gid: string) => aria2.remove(gid),
		onSuccess: invalidateTasks,
	});

	const forceRemove = useMutation({
		mutationFn: (gid: string) => aria2.forceRemove(gid),
		onSuccess: invalidateTasks,
	});

	const purgeDownloadResult = useMutation({
		mutationFn: () => aria2.purgeDownloadResult(),
		onSuccess: invalidateTasks,
	});

	const saveSession = useMutation({
		mutationFn: () => aria2.saveSession(),
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
		purgeDownloadResult,
		saveSession,
		changeGlobalOption,
		changeOption,
	};
}
