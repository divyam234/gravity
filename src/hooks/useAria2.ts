import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { aria2 } from "../lib/aria2-rpc";
import { useSettingsStore } from "../store/useSettingsStore";

export function useSyncAria2() {
	const { rpcUrl, rpcSecret } = useSettingsStore();

	useEffect(() => {
		aria2.updateConfig(rpcUrl, rpcSecret);
	}, [rpcUrl, rpcSecret]);
}

export function useAria2Version() {
	const { rpcUrl } = useSettingsStore(); // Trigger re-fetch on URL change
	return useQuery({
		queryKey: ["aria2", "version", rpcUrl],
		queryFn: () => aria2.getVersion(),
		retry: false,
	});
}

export function useGlobalStat() {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "globalStat", rpcUrl],
		queryFn: () => aria2.getGlobalStat(),
		refetchInterval: pollingInterval,
		retry: false,
	});
}

export function useActiveTasks() {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "tasks", "active", rpcUrl],
		queryFn: () => aria2.tellActive(),
		refetchInterval: pollingInterval,
		retry: false,
	});
}

export function useWaitingTasks(offset = 0, num = 100) {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "tasks", "waiting", offset, num, rpcUrl],
		queryFn: () => aria2.tellWaiting(offset, num),
		refetchInterval: pollingInterval,
		retry: false,
	});
}

export function useStoppedTasks(offset = 0, num = 100) {
	const { pollingInterval, rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "tasks", "stopped", offset, num, rpcUrl],
		queryFn: () => aria2.tellStopped(offset, num),
		refetchInterval: pollingInterval,
		retry: false,
	});
}

export function useAllTasks() {
	const active = useActiveTasks();
	const waiting = useWaitingTasks(0, 50);
	const stopped = useStoppedTasks(0, 50);

	return {
		isLoading: active.isLoading || waiting.isLoading || stopped.isLoading,
		isError: active.isError || waiting.isError || stopped.isError,
		active: active.data || [],
		waiting: waiting.data || [],
		stopped: stopped.data || [],
		refetch: () => {
			active.refetch();
			waiting.refetch();
			stopped.refetch();
		},
	};
}

export function useGlobalOption() {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "globalOption", rpcUrl],
		queryFn: () => aria2.getGlobalOption(),
	});
}

export function useTaskFiles(gid: string) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "files", gid, rpcUrl],
		queryFn: () => aria2.getFiles(gid),
		enabled: !!gid,
		refetchInterval: 2000,
	});
}

export function useTaskPeers(gid: string, enabled: boolean = false) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "peers", gid, rpcUrl],
		queryFn: () => aria2.getPeers(gid),
		enabled: !!gid && enabled,
		refetchInterval: 3000,
	});
}

export function useTaskServers(gid: string, enabled: boolean = false) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "servers", gid, rpcUrl],
		queryFn: () => aria2.getServers(gid),
		enabled: !!gid && enabled,
		refetchInterval: 5000,
	});
}

export function useTaskStatus(gid: string) {
	const { rpcUrl } = useSettingsStore();
	return useQuery({
		queryKey: ["aria2", "status", gid, rpcUrl],
		queryFn: () => aria2.tellStatus(gid),
		enabled: !!gid,
		refetchInterval: 2000,
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

	return {
		addUri,
		addTorrent,
		addMetalink,
		pause,
		unpause,
		remove,
		forceRemove,
		purgeDownloadResult,
	};
}
