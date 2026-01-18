import { useEffect, useRef } from "react";
import { aria2 } from "../lib/aria2-rpc";
import { useSettingsStore } from "../store/useSettingsStore";
import { useActiveTasks, useGlobalStat } from "./useAria2";

export function useNotifications() {
	const { enableNotifications } = useSettingsStore();
	const { data: stats } = useGlobalStat();
	// Only poll active tasks if notifications are enabled AND there are active tasks
	const hasActive = enableNotifications && Number(stats?.numActive) > 0;

	const { data: activeTasks } = useActiveTasks({ enabled: hasActive });
	const previousActiveGids = useRef<Set<string>>(new Set());

	useEffect(() => {
		if (
			enableNotifications &&
			"Notification" in window &&
			Notification.permission === "default"
		) {
			Notification.requestPermission();
		}
	}, [enableNotifications]);

	useEffect(() => {
		if (!enableNotifications) return;

		// If we have no active tasks and no history, nothing to do
		if (!activeTasks && previousActiveGids.current.size === 0) return;

		const currentActiveGids = new Set(activeTasks?.map((t) => t.gid) || []);

		// Check for tasks that were active and are now missing
		const checkFinishedTasks = async () => {
			for (const gid of previousActiveGids.current) {
				if (!currentActiveGids.has(gid)) {
					try {
						// Fetch status of the missing task
						const task = await aria2.tellStatus(gid);
						if (task) {
							if (task.status === "complete") {
								new Notification("Download Complete", {
									body:
										task.bittorrent?.info?.name ||
										task.files[0]?.path?.split("/").pop() ||
										"File downloaded",
									icon: "/logo.svg",
								});
							} else if (task.status === "error") {
								new Notification("Download Error", {
									body: `Error downloading ${task.files[0]?.path?.split("/").pop() || gid}`,
									icon: "/logo.svg",
								});
							}
						}
					} catch (err) {
						// Task might be removed or other error
						console.error("Failed to check status of finished task", err);
					}
				}
			}
			previousActiveGids.current = currentActiveGids;
		};

		checkFinishedTasks();
	}, [activeTasks, enableNotifications]);
}
