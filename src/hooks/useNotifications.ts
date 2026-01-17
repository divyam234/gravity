import { useEffect, useRef } from "react";
import { useActiveTasks, useStoppedTasks } from "./useAria2";

export function useNotifications() {
	const { data: activeTasks } = useActiveTasks();
	const { data: stoppedTasks } = useStoppedTasks();
	const previousActiveGids = useRef<Set<string>>(new Set());

	useEffect(() => {
		if ("Notification" in window && Notification.permission === "default") {
			Notification.requestPermission();
		}
	}, []);

	useEffect(() => {
		if (!activeTasks || !stoppedTasks) return;

		const currentActiveGids = new Set(activeTasks.map((t) => t.gid));

		// Check for tasks that were active and are now stopped
		for (const gid of previousActiveGids.current) {
			if (!currentActiveGids.has(gid)) {
				const task = stoppedTasks.find((t) => t.gid === gid);
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
			}
		}

		previousActiveGids.current = currentActiveGids;
	}, [activeTasks, stoppedTasks]);
}
