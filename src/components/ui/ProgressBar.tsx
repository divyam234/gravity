import type React from "react";
import { ProgressBar as AriaProgressBar } from "react-aria-components";
import { cn } from "../../lib/utils";

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
		<AriaProgressBar
			value={value}
			maxValue={max}
			className={cn("w-full flex flex-col gap-1", className)}
		>
			{({ percentage, valueText }) => (
				<>
					<div className="flex justify-between text-small text-default-500 font-medium">
						{label && <span>{label}</span>}
						{showValueLabel && <span>{valueText}</span>}
					</div>
					<div
						className={cn(
							"w-full bg-default-100 rounded-full overflow-hidden",
							heightMap[size],
						)}
					>
						<div
							className={cn(
								"h-full transition-all duration-500 ease-out rounded-full shadow-sm",
								colorMap[color],
							)}
							style={{ width: `${percentage}%` }}
						/>
					</div>
				</>
			)}
		</AriaProgressBar>
	);
};
