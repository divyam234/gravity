import { Button, Card, Kbd, Tooltip } from "@heroui/react";
import { Link } from "@tanstack/react-router";
import type React from "react";
import IconListUl from "~icons/gravity-ui/list-ul";
import IconPause from "~icons/gravity-ui/pause";
import IconPlay from "~icons/gravity-ui/play";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useAria2Actions } from "../../hooks/useAria2";
import type { Aria2Task } from "../../lib/aria2-rpc";
import { formatBytes, formatTime } from "../../lib/utils";
import { ProgressBar } from "../ui/ProgressBar";
import { StatusChip } from "../ui/StatusChip";

interface DownloadCardProps {
	task: Aria2Task;
}

export const DownloadCard: React.FC<DownloadCardProps> = ({ task }) => {
	const { pause, unpause, remove } = useAria2Actions();

	const totalLength = Number(task.totalLength);
	const completedLength = Number(task.completedLength);
	const downloadSpeed = Number(task.downloadSpeed);
	const uploadSpeed = Number(task.uploadSpeed);

	const progress = totalLength > 0 ? (completedLength / totalLength) * 100 : 0;
	const eta =
		downloadSpeed > 0 ? (totalLength - completedLength) / downloadSpeed : 0;

	const fileName =
		task.bittorrent?.info?.name ||
		task.files[0]?.path?.split("/").pop() ||
		task.files[0]?.uris[0]?.uri?.split("/").pop() ||
		task.gid;

	const isPaused = task.status === "paused";
	const isActive = task.status === "active";
	const isError = task.status === "error";
	const isComplete = task.status === "complete";

	return (
		<Card className="w-full shadow-sm border-default-100 hover:shadow-md transition-shadow">
			<Card.Content className="p-4 space-y-4">
				<div className="flex justify-between items-start gap-4">
					<Link
						to="/task/$gid"
						params={{ gid: task.gid }}
						className="flex-1 min-w-0 cursor-pointer group outline-none"
					>
						<h3
							className="text-lg font-bold truncate group-hover:text-primary transition-colors leading-tight"
							title={fileName}
						>
							{fileName}
						</h3>
						<div className="flex items-center gap-2 mt-1.5">
							<StatusChip status={task.status} />
							<span className="text-small text-default-500 font-medium">
								{formatBytes(completedLength)} / {formatBytes(totalLength)}
							</span>
						</div>
					</Link>

					<div className="flex items-center gap-1">
						<Tooltip>
							<Tooltip.Trigger>
								<Link to="/task/$gid" params={{ gid: task.gid }}>
									<Button isIconOnly size="sm" variant="ghost">
										<IconListUl className="w-4.5 h-4.5" />
									</Button>
								</Link>
							</Tooltip.Trigger>
							<Tooltip.Content className="p-2 text-tiny">
								Files & Details
							</Tooltip.Content>
						</Tooltip>

						{isActive && (
							<Tooltip>
								<Tooltip.Trigger>
									<Button
										isIconOnly
										size="sm"
										variant="ghost"
										onPress={() => pause.mutate(task.gid)}
									>
										<IconPause className="w-4.5 h-4.5 text-warning" />
									</Button>
								</Tooltip.Trigger>
								<Tooltip.Content className="p-2 text-tiny flex items-center gap-2">
									Pause <Kbd>P</Kbd>
								</Tooltip.Content>
							</Tooltip>
						)}

						{isPaused && (
							<Tooltip>
								<Tooltip.Trigger>
									<Button
										isIconOnly
										size="sm"
										variant="ghost"
										onPress={() => unpause.mutate(task.gid)}
									>
										<IconPlay className="w-4.5 h-4.5 text-success" />
									</Button>
								</Tooltip.Trigger>
								<Tooltip.Content className="p-2 text-tiny flex items-center gap-2">
									Resume <Kbd>R</Kbd>
								</Tooltip.Content>
							</Tooltip>
						)}

						<Tooltip>
							<Tooltip.Trigger>
								<Button
									isIconOnly
									size="sm"
									variant="ghost"
									className="text-danger"
									onPress={() => remove.mutate(task.gid)}
								>
									<IconTrashBin className="w-4.5 h-4.5" />
								</Button>
							</Tooltip.Trigger>
							<Tooltip.Content className="p-2 text-tiny">
								Remove
							</Tooltip.Content>
						</Tooltip>
					</div>
				</div>

				<ProgressBar
					value={progress}
					size="md"
					color={isError ? "danger" : isComplete ? "success" : "primary"}
					showValueLabel
				/>

				<div className="flex justify-between items-center text-small text-default-500 bg-default-50/50 p-2 rounded-xl border border-default-50">
					<div className="flex gap-4">
						<span className="flex items-center gap-1.5 font-bold text-success/80">
							<div className="w-1.5 h-1.5 rounded-full bg-success" />
							{formatBytes(downloadSpeed)}/s
						</span>
						<span className="flex items-center gap-1.5 font-bold text-primary/80">
							<div className="w-1.5 h-1.5 rounded-full bg-primary" />
							{formatBytes(uploadSpeed)}/s
						</span>
					</div>

					<div className="flex gap-4 font-medium">
						{isActive && <span>ETA: {formatTime(eta)}</span>}
						{task.numSeeders && (
							<span className="bg-default-100 px-2 py-0.5 rounded text-[10px] uppercase font-black">
								Peers: {task.connections} (S:{task.numSeeders})
							</span>
						)}
						{!task.numSeeders && (
							<span className="bg-default-100 px-2 py-0.5 rounded text-[10px] uppercase font-black">
								Conn: {task.connections}
							</span>
						)}
					</div>
				</div>
			</Card.Content>
		</Card>
	);
};
