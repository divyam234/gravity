import type React from "react";
import { ProgressBar as AriaProgressBar } from "react-aria-components";
import { cn } from "../../lib/utils";

interface ProgressBarProps {
	value: number;
	max?: number;
	label?: string;
	showValueLabel?: boolean;
	color?: "accent" | "success" | "warning" | "danger" | "default";
	size?: "sm" | "md" | "lg";
	className?: string;
	isIndeterminate?: boolean;
}

export const ProgressBar: React.FC<ProgressBarProps> = ({
	value,
	max = 100,
	label,
	showValueLabel = false,
	color = "accent",
	size = "md",
	className,
	isIndeterminate = false,
}) => {
	const heightMap = {
		sm: "h-1",
		md: "h-2.5",
		lg: "h-4",
	};

	const colorMap = {
		default: "bg-default/50",
		accent: "bg-accent",
		success: "bg-success",
		warning: "bg-warning",
		danger: "bg-danger",
	};

	return (
		<AriaProgressBar
			value={value}
			maxValue={max}
			isIndeterminate={isIndeterminate}
			className={cn("w-full flex flex-col gap-1", className)}
		>
			{({ percentage, valueText }) => (
				<>
					<div className="flex justify-between text-sm text-muted font-medium">
						{label && <span>{label}</span>}
						{showValueLabel && !isIndeterminate && <span>{valueText}</span>}
						{isIndeterminate && <span>Uploading...</span>}
					</div>
					<div
						className={cn(
							"w-full bg-default/30 rounded-full overflow-hidden relative",
							heightMap[size],
						)}
					>
						<div
							className={cn(
								"h-full rounded-full shadow-sm",
								colorMap[color],
								isIndeterminate
									? "absolute left-0 top-0 w-1/3 animate-[shimmer_1.5s_infinite_linear]"
									: "transition-all duration-500 ease-out",
							)}
							style={!isIndeterminate ? { width: `${percentage}%` } : {}}
						/>
					</div>
				</>
			)}
		</AriaProgressBar>
	);
};
