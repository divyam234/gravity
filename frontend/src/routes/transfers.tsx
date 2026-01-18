import { Card, Chip } from "@heroui/react";
import { createFileRoute } from "@tanstack/react-router";
import IconArrowUp from "~icons/gravity-ui/arrow-up";
import IconCloud from "~icons/gravity-ui/cloud";
import IconPulse from "~icons/gravity-ui/pulse";
import { useRcloneStats } from "../hooks/useRclone";
import { formatBytes } from "../lib/utils";
import { ProgressBar } from "../components/ui/ProgressBar";

export const Route = createFileRoute("/transfers")({
	component: TransfersPage,
});

function TransfersPage() {
	const { data: stats } = useRcloneStats();

	const activeTransfers = stats?.transferring || [];

	return (
		<div className="space-y-6">
			<div className="flex justify-between items-center">
				<h2 className="text-2xl font-bold tracking-tight">Cloud Transfers</h2>
			</div>

			<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
				<Card className="shadow-sm border-border">
					<Card.Content className="p-4 flex items-center gap-4">
						<div className="p-3 rounded-full bg-accent/10 text-accent">
							<IconArrowUp className="w-6 h-6" />
						</div>
						<div>
							<p className="text-sm text-muted font-medium">Upload Speed</p>
							<h4 className="text-2xl font-bold">
								{formatBytes(stats?.speed || 0)}/s
							</h4>
						</div>
					</Card.Content>
				</Card>

				<Card className="shadow-sm border-border">
					<Card.Content className="p-4 flex items-center gap-4">
						<div className="p-3 rounded-full bg-success/10 text-success">
							<IconCloud className="w-6 h-6" />
						</div>
						<div>
							<p className="text-sm text-muted font-medium">Transferred</p>
							<h4 className="text-2xl font-bold">
								{formatBytes(stats?.bytes || 0)}
							</h4>
						</div>
					</Card.Content>
				</Card>

				<Card className="shadow-sm border-border">
					<Card.Content className="p-4 flex items-center gap-4">
						<div className="p-3 rounded-full bg-warning/10 text-warning">
							<IconPulse className="w-6 h-6" />
						</div>
						<div>
							<p className="text-sm text-muted font-medium">Active Transfers</p>
							<h4 className="text-2xl font-bold">
								{stats?.transfers || 0}
							</h4>
						</div>
					</Card.Content>
				</Card>
			</div>

			<div className="space-y-4">
				<h3 className="text-lg font-bold">Active Uploads</h3>
				{activeTransfers.length === 0 ? (
					<div className="flex flex-col items-center justify-center py-12 text-muted bg-default/5 rounded-3xl border border-border">
						<IconCloud className="w-12 h-12 mb-4 opacity-50" />
						<p className="text-lg font-medium">No active transfers</p>
					</div>
				) : (
					<div className="grid gap-4">
						{activeTransfers.map((transfer, i) => (
							<Card
								key={transfer.name + i}
								className="shadow-sm border-border hover:shadow-md transition-shadow"
							>
								<Card.Content className="p-4 space-y-3">
									<div className="flex justify-between items-start">
										<div className="flex-1 min-w-0">
											<h4 className="font-bold truncate" title={transfer.name}>
												{transfer.name}
											</h4>
											<p className="text-xs text-muted font-mono mt-1">
												{transfer.group}
											</p>
										</div>
										<Chip size="sm" variant="soft" color="success">
											{formatBytes(transfer.speed)}/s
										</Chip>
									</div>
									<div className="space-y-1">
										<div className="flex justify-between text-xs text-muted">
											<span>{transfer.percentage}%</span>
											<span>
												{formatBytes(transfer.bytes)} /{" "}
												{formatBytes(transfer.size)}
											</span>
										</div>
										<ProgressBar
											value={transfer.percentage}
											size="sm"
											color="accent"
										/>
									</div>
								</Card.Content>
							</Card>
						))}
					</div>
				)}
			</div>
		</div>
	);
}
