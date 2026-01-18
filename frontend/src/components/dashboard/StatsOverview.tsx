import { Card } from "@heroui/react";
import type React from "react";
import IconArrowDown from "~icons/gravity-ui/arrow-down";
import IconArrowUp from "~icons/gravity-ui/arrow-up";
import IconPulse from "~icons/gravity-ui/pulse";
import { useGlobalStat } from "../../hooks/useAria2";
import { useSpeedHistory } from "../../hooks/useSpeedHistory";
import { formatBytes } from "../../lib/utils";
import { SpeedGraph } from "../ui/SpeedGraph";

export const StatsOverview: React.FC = () => {
	const { data: stats } = useGlobalStat();
	const { downloadHistory, uploadHistory } = useSpeedHistory();

	return (
		<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
			<Card className="overflow-hidden shadow-sm border-border">
				<Card.Content className="p-4 flex flex-col gap-2">
					<div className="flex items-center gap-4">
						<div className="p-3 rounded-full bg-success/10 text-success">
							<IconArrowDown className="w-6 h-6" />
						</div>
						<div>
							<p className="text-sm text-muted font-medium">Download Speed</p>
							<h4 className="text-2xl font-bold">
								{formatBytes(stats?.downloadSpeed ?? 0)}/s
							</h4>
						</div>
					</div>
					<SpeedGraph
						data={downloadHistory}
						color="var(--success)"
						height={40}
						className="mt-2"
					/>
				</Card.Content>
			</Card>

			<Card className="overflow-hidden shadow-sm border-border">
				<Card.Content className="p-4 flex flex-col gap-2">
					<div className="flex items-center gap-4">
						<div className="p-3 rounded-full bg-accent/10 text-accent">
							<IconArrowUp className="w-6 h-6" />
						</div>
						<div>
							<p className="text-sm text-muted font-medium">Upload Speed</p>
							<h4 className="text-2xl font-bold">
								{formatBytes(stats?.uploadSpeed ?? 0)}/s
							</h4>
						</div>
					</div>
					<SpeedGraph
						data={uploadHistory}
						color="var(--accent)"
						height={40}
						className="mt-2"
					/>
				</Card.Content>
			</Card>

			<Card className="shadow-sm border-border">
				<Card.Content className="p-4 flex items-center h-full gap-4">
					<div className="p-3 rounded-full bg-warning/10 text-warning">
						<IconPulse className="w-6 h-6" />
					</div>
					<div>
						<p className="text-sm text-muted font-medium">Active Tasks</p>
						<div className="flex gap-2 items-baseline">
							<h4 className="text-2xl font-bold">{stats?.numActive ?? 0}</h4>
							<span className="text-sm text-muted">
								({stats?.numWaiting ?? 0} waiting)
							</span>
						</div>
					</div>
				</Card.Content>
			</Card>
		</div>
	);
};
