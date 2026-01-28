import { Chip } from "@heroui/react";
import type React from "react";
import { cn } from "../../lib/utils";

interface StatusChipProps {
	status: string;
	className?: string;
}

export const StatusChip: React.FC<StatusChipProps> = ({
	status,
	className,
}) => {
	const colorMap: Record<
		string,
		"success" | "warning" | "default" | "danger" | "accent"
	> = {
		active: "success",
		waiting: "warning",
		paused: "default",
		allocating: "warning",
		error: "danger",
		complete: "accent",
		removed: "default",
		uploading: "default", // Use default and override with class
	};

	const labelMap: Record<string, string> = {
		active: "Downloading",
		waiting: "Queued",
		paused: "Paused",
		allocating: "Allocating",
		error: "Error",
		complete: "Finished",
		removed: "Removed",
		uploading: "Uploading to Cloud",
	};

	return (
		<Chip
			color={colorMap[status] || "default"}
			variant="soft"
			size="sm"
			className={cn(
				className,
				status === "uploading" && "bg-cyan-500/10 text-cyan-600 font-bold",
			)}
		>
			{labelMap[status] || status}
		</Chip>
	);
};
