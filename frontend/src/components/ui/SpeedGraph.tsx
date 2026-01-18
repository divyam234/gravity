import type React from "react";
import { useMemo } from "react";
import { Area, AreaChart, ResponsiveContainer, Tooltip } from "recharts";
import { cn, formatBytes } from "../../lib/utils";

interface SpeedGraphProps {
	data: number[];
	color?: string;
	className?: string;
	height?: number;
}

const CustomTooltip = ({ active, payload }: any) => {
	if (active && payload && payload.length) {
		return (
			<div className="bg-content1/80 backdrop-blur-md border border-border px-2 py-1 rounded-lg shadow-xl text-[10px] font-bold text-foreground/90">
				{formatBytes(payload[0].value ?? 0)}/s
			</div>
		);
	}
	return null;
};

export const SpeedGraph: React.FC<SpeedGraphProps> = ({
	data,
	color = "var(--accent)",
	className,
	height = 40,
}) => {
	const chartData = useMemo(
		() => data.map((value, i) => ({ value, i })),
		[data],
	);
	const gradientId = useMemo(
		() => `gradient-${color.replace(/[^a-zA-Z0-9]/g, "")}`,
		[color],
	);

	if (data.length < 2) {
		return <div className={cn("w-full", className)} style={{ height }} />;
	}

	return (
		<div className={cn("w-full overflow-hidden", className)} style={{ height }}>
			<ResponsiveContainer width="100%" height="100%">
				<AreaChart
					data={chartData}
					margin={{ top: 5, right: 0, left: 0, bottom: 0 }}
				>
					<defs>
						<linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
							<stop offset="5%" stopColor={color} stopOpacity={0.3} />
							<stop offset="95%" stopColor={color} stopOpacity={0} />
						</linearGradient>
					</defs>
					<Tooltip
						content={<CustomTooltip />}
						cursor={{ stroke: color, strokeWidth: 1, strokeDasharray: "3 3" }}
						isAnimationActive={false}
					/>
					<Area
						type="monotone"
						dataKey="value"
						stroke={color}
						strokeWidth={2}
						fillOpacity={1}
						fill={`url(#${gradientId})`}
						isAnimationActive={true}
						animationDuration={300}
					/>
				</AreaChart>
			</ResponsiveContainer>
		</div>
	);
};
