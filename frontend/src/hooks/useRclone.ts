import { useQuery } from "@tanstack/react-query";
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
