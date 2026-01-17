import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export function formatBytes(bytes: number | string, decimals = 2) {
	if (!bytes) return "0 B";
	const k = 1024;
	const dm = decimals < 0 ? 0 : decimals;
	const sizes = ["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

	const i = Math.floor(Math.log(Number(bytes)) / Math.log(k));
	return `${parseFloat((Number(bytes) / k ** i).toFixed(dm))} ${sizes[i]}`;
}

export function formatTime(seconds: number | string) {
	if (!seconds || Number(seconds) === 0) return "--";
	const sec = Number(seconds);
	if (sec > 86400) return "> 1d";
	const h = Math.floor(sec / 3600);
	const m = Math.floor((sec % 3600) / 60);
	const s = Math.floor(sec % 60);
	return `${h}h ${m}m ${s}s`.replace(/^0h /, "").replace(/^0m /, "");
}
