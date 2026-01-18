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
	variant?: "list" | "grid";
}

export const DownloadCard: React.FC<DownloadCardProps> = ({
	task,
	variant = "list",
}) => {
	const { pause, unpause, remove, removeDownloadResult } = useAria2Actions();

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

	const handleRemove = () => {
		if (isComplete || isError || task.status === "removed") {
			removeDownloadResult.mutate(task.gid);
		} else {
			remove.mutate(task.gid);
		}
	};

	if (variant === "list") {
		return (
			<div className="w-full flex items-center gap-6 py-4 px-6 hover:bg-default/5 transition-colors rounded-xl group/item min-h-[72px]">
				<Link
					to="/task/$gid"
					params={{ gid: task.gid }}
					className="flex-1 min-w-0 cursor-pointer outline-none flex items-center gap-8"
				>
					<div className="flex-1 min-w-0">
						<h3
							className="text-base font-bold truncate group-hover/item:text-accent transition-colors leading-tight"
							title={fileName}
						>
							{fileName}
						</h3>
						<div className="flex items-center gap-3 mt-1.5">
							<StatusChip
								status={task.status}
								className="h-5 text-[10px] px-2"
							/>
							<span className="text-xs text-muted font-bold">
								{formatBytes(completedLength)} / {formatBytes(totalLength)}
							</span>
						</div>
					</div>

					<div className="w-48 shrink-0 hidden sm:block">
						<ProgressBar
							value={progress}
							size="sm"
							color={isError ? "danger" : isComplete ? "success" : "accent"}
							className="h-2"
						/>
					</div>

					<div className="w-32 shrink-0 hidden md:flex flex-col items-end gap-0.5">
						<span className="text-xs font-black text-success/80">
							↓ {formatBytes(downloadSpeed)}/s
						</span>
						{isActive && (
							<span className="text-[10px] text-muted font-bold uppercase tracking-wider">
								{formatTime(eta)}
							</span>
						)}
					</div>
				</Link>

				<div className="flex items-center gap-2 shrink-0 opacity-0 group-hover/item:opacity-100 transition-opacity ml-auto">
					{isActive && (
						<Button
							isIconOnly
							size="sm"
							variant="ghost"
							onPress={() => pause.mutate(task.gid)}
							className="h-8 w-8 min-w-0"
						>
							<IconPause className="w-4 h-4 text-warning" />
						</Button>
					)}
					{isPaused && (
						<Button
							isIconOnly
							size="sm"
							variant="ghost"
							onPress={() => unpause.mutate(task.gid)}
							className="h-8 w-8 min-w-0"
						>
							<IconPlay className="w-4 h-4 text-success" />
						</Button>
					)}
					<Button
						isIconOnly
						size="sm"
						variant="ghost"
						className="text-danger h-8 w-8 min-w-0"
						onPress={handleRemove}
					>
						<IconTrashBin className="w-4 h-4" />
					</Button>
				</div>
			</div>
		);
	}

	return (
		<Card className="w-full shadow-sm border-border hover:shadow-md transition-shadow">
			<Card.Content className="p-4 space-y-4">
				<div className="flex justify-between items-start gap-4">
					<Link
						to="/task/$gid"
						params={{ gid: task.gid }}
						className="flex-1 min-w-0 cursor-pointer group outline-none"
					>
						<h3
							className="text-lg font-bold truncate group-hover:text-accent transition-colors leading-tight"
							title={fileName}
						>
							{fileName}
						</h3>
						<div className="flex items-center gap-2 mt-1.5">
							<StatusChip status={task.status} />
							<span className="text-sm text-muted font-medium">
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
							<Tooltip.Content className="p-2 text-xs">
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
								<Tooltip.Content className="p-2 text-xs flex items-center gap-2">
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
								<Tooltip.Content className="p-2 text-xs flex items-center gap-2">
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
									onPress={handleRemove}
								>
									<IconTrashBin className="w-4.5 h-4.5" />
								</Button>
							</Tooltip.Trigger>
							<Tooltip.Content className="p-2 text-xs">Remove</Tooltip.Content>
						</Tooltip>
					</div>
				</div>

				<ProgressBar
					value={progress}
					size="md"
					color={isError ? "danger" : isComplete ? "success" : "accent"}
					showValueLabel
				/>

				<div className="flex justify-between items-center text-sm text-muted bg-muted-background/50 p-2 rounded-xl border border-border/50">
					<div className="flex gap-4">
						<span className="flex items-center gap-1.5 font-bold text-success/80">
							<div className="w-1.5 h-1.5 rounded-full bg-success" />
							{formatBytes(downloadSpeed)}/s
						</span>
						<span className="flex items-center gap-1.5 font-bold text-accent/80">
							<div className="w-1.5 h-1.5 rounded-full bg-accent" />
							{formatBytes(uploadSpeed)}/s
						</span>
					</div>

					<div className="flex gap-4 font-medium">
						{isActive && <span>ETA: {formatTime(eta)}</span>}
						{task.numSeeders && (
							<span className="bg-default/30 px-2 py-0.5 rounded text-[10px] uppercase font-black">
								Peers: {task.connections} (S:{task.numSeeders})
							</span>
						)}
						{!task.numSeeders && (
							<span className="bg-default/30 px-2 py-0.5 rounded text-[10px] uppercase font-black">
								Conn: {task.connections}
							</span>
						)}
					</div>
				</div>
			</Card.Content>
		</Card>
	);
};
