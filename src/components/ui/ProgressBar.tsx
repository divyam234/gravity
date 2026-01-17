import { cn } from "@heroui/react";
import type React from "react";

interface ProgressBarProps {
	value: number;
	max?: number;
	label?: string;
	showValueLabel?: boolean;
	color?: "primary" | "success" | "warning" | "danger" | "default";
	size?: "sm" | "md" | "lg";
	className?: string;
}

export const ProgressBar: React.FC<ProgressBarProps> = ({
	value,
	max = 100,
	label,
	showValueLabel = false,
	color = "primary",
	size = "md",
	className,
}) => {
	const percentage = Math.min(Math.max((value / max) * 100, 0), 100);

	const heightMap = {
		sm: "h-1",
		md: "h-2.5",
		lg: "h-4",
	};

	const colorMap = {
		default: "bg-default-400",
		primary: "bg-primary",
		success: "bg-success",
		warning: "bg-warning",
		danger: "bg-danger",
	};

	return (
		<div className={cn("w-full flex flex-col gap-1", className)}>
			{(label || showValueLabel) && (
				<div className="flex justify-between text-small text-default-500">
					{label && <span>{label}</span>}
					{showValueLabel && <span>{percentage.toFixed(1)}%</span>}
				</div>
			)}
			<div
				className={cn(
					"w-full bg-default-100 rounded-full overflow-hidden",
					heightMap[size],
				)}
			>
				<div
					className={cn(
						"h-full transition-all duration-500 ease-out rounded-full",
						colorMap[color],
					)}
					style={{ width: `${percentage}%` }}
				/>
			</div>
		</div>
	);
};
