import { useEffect, useRef, useState } from "react";
import { useGlobalStat } from "./useAria2";

const MAX_HISTORY = 60; // 60 seconds

export function useSpeedHistory() {
	const { data: stats } = useGlobalStat();
	const [downloadHistory, setDownloadHistory] = useState<number[]>([]);
	const [uploadHistory, setUploadHistory] = useState<number[]>([]);
	const lastUpdate = useRef<number>(0);

	useEffect(() => {
		if (stats) {
			const now = Date.now();
			// Only update once per second approx
			if (now - lastUpdate.current >= 900) {
				setDownloadHistory((prev) => {
					const next = [...prev, Number(stats.downloadSpeed)];
					return next.slice(-MAX_HISTORY);
				});
				setUploadHistory((prev) => {
					const next = [...prev, Number(stats.uploadSpeed)];
					return next.slice(-MAX_HISTORY);
				});
				lastUpdate.current = now;
			}
		}
	}, [stats]);

	return { downloadHistory, uploadHistory };
}
