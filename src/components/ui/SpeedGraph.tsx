import type React from "react";
import { cn } from "../../lib/utils";

interface SpeedGraphProps {
	data: number[];
	color?: string;
	className?: string;
	height?: number;
}

export const SpeedGraph: React.FC<SpeedGraphProps> = ({
	data,
	color = "currentColor",
	className,
	height = 40,
}) => {
	if (data.length < 2)
		return <div className={cn("w-full", className)} style={{ height }} />;

	const max = Math.max(...data, 1024 * 1024); // Min 1MB/s scale
	const min = 0;
	const range = max - min;
	const width = 100; // SVG viewBox width

	const points = data
		.map((val, i) => {
			const x = (i / (data.length - 1)) * width;
			const y = height - ((val - min) / range) * height;
			return `${x},${y}`;
		})
		.join(" ");

	const areaPoints = `0,${height} ${points} ${width},${height}`;

	return (
		<div className={cn("w-full overflow-hidden", className)}>
			<svg
				viewBox={`0 0 ${width} ${height}`}
				preserveAspectRatio="none"
				className="w-full h-full"
			>
				<defs>
					<linearGradient id={`gradient-${color}`} x1="0" y1="0" x2="0" y2="1">
						<stop offset="0%" stopColor={color} stopOpacity="0.3" />
						<stop offset="100%" stopColor={color} stopOpacity="0" />
					</linearGradient>
				</defs>
				<polyline
					fill="none"
					stroke={color}
					strokeWidth="1.5"
					strokeLinecap="round"
					strokeLinejoin="round"
					points={points}
				/>
				<polygon fill={`url(#gradient-${color})`} points={areaPoints} />
			</svg>
		</div>
	);
};
