import { useEffect, useRef, useState } from "react";
import { useGlobalStat } from "./useEngine";

const MAX_HISTORY = 60; // 60 seconds

export function useSpeedHistory() {
	const statsQuery = useGlobalStat();
	const stats = statsQuery.data;
	const [downloadHistory, setDownloadHistory] = useState<number[]>(
		new Array(MAX_HISTORY).fill(0),
	);
	const [uploadHistory, setUploadHistory] = useState<number[]>(
		new Array(MAX_HISTORY).fill(0),
	);
	const lastUpdate = useRef<number>(0);

	useEffect(() => {
		if (stats) {
			const now = Date.now();
			if (now - lastUpdate.current >= 950) {
				setDownloadHistory((prev) => {
					const next = [...prev, Number(stats.speeds.download)];
					return next.slice(-MAX_HISTORY);
				});
				setUploadHistory((prev) => {
					const next = [...prev, Number(stats.speeds.upload)];
					return next.slice(-MAX_HISTORY);
				});
				lastUpdate.current = now;
			}
		}
	}, [stats]);

	return { downloadHistory, uploadHistory };
}
