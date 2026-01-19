import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { aria2 } from "../lib/aria2-rpc";

export function useRcloneStats(enabled = true) {
	return useQuery({
		queryKey: ["rclone", "stats"],
		queryFn: () => aria2.rcloneGetStats(),
		refetchInterval: 1000,
		enabled,
	});
}

export function useRcloneVersion() {
	return useQuery({
		queryKey: ["rclone", "version"],
		queryFn: () => aria2.rcloneGetVersion(),
	});
}

export function useRcloneRemotes() {
	return useQuery({
		queryKey: ["rclone", "remotes"],
		queryFn: () => aria2.rcloneListRemotes(),
	});
}

export function useRcloneActions() {
	const queryClient = useQueryClient();

	const createRemote = useMutation({
		mutationFn: (params: { name: string; type: string; parameters: Record<string, string> }) =>
			aria2.rcloneCreateRemote(params.name, params.type, params.parameters),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["rclone", "remotes"] });
			toast.success("Remote created successfully");
		},
		onError: (err: Error) => toast.error(`Failed to create remote: ${err.message}`),
	});

	const deleteRemote = useMutation({
		mutationFn: (name: string) =>
			aria2.rcloneDeleteRemote(name),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["rclone", "remotes"] });
			toast.success("Remote deleted");
		},
		onError: (err: Error) => toast.error(`Failed to delete remote: ${err.message}`),
	});

	return { createRemote, deleteRemote };
}
