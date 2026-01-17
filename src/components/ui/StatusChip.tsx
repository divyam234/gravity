import { Chip } from "@heroui/react";
import type React from "react";

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
		error: "danger",
		complete: "accent",
		removed: "default",
	};

	const labelMap: Record<string, string> = {
		active: "Downloading",
		waiting: "Queued",
		paused: "Paused",
		error: "Error",
		complete: "Finished",
		removed: "Removed",
	};

	return (
		<Chip
			color={colorMap[status] || "default"}
			variant="soft"
			size="sm"
			className={className}
		>
			{labelMap[status] || status}
		</Chip>
	);
};
