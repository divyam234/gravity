import {
	Button,
	Checkbox,
	Dropdown,
	Input,
	ListBox,
	Select,
} from "@heroui/react";
import React, { useId } from "react";
import { type Selection, useDragAndDrop } from "react-aria-components";
import IconArchive from "~icons/gravity-ui/archive";
import IconCopy from "~icons/gravity-ui/copy";
import IconMagnifier from "~icons/gravity-ui/magnifier";
import IconPause from "~icons/gravity-ui/pause";
import IconPlay from "~icons/gravity-ui/play";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useAllTasks, useAria2Actions } from "../../hooks/useAria2";
import type { Aria2Task } from "../../lib/aria2-rpc";
import { aria2 } from "../../lib/aria2-rpc";
import { DownloadCard } from "./DownloadCard";

interface TaskListProps {
	status: string;
}

export const TaskList: React.FC<TaskListProps> = ({ status }) => {
	const { active, waiting, stopped, refetch } = useAllTasks();
	const { pause, unpause, remove } = useAria2Actions();
	const baseId = useId();
	const [selectedGids, setSelectedGids] = React.useState<Selection>(new Set());
	const [isSelectionMode, setIsSelectionMode] = React.useState(false);
	const [searchQuery, setSearchQuery] = React.useState("");
	const [sortBy, setSortBy] = React.useState<string>("default");

	const allTasks = React.useMemo(
		() => [...active, ...waiting, ...stopped],
		[active, waiting, stopped],
	);

	const tasks = React.useMemo(() => {
		let t: Aria2Task[] = [];
		switch (status) {
			case "active":
				t = active;
				break;
			case "waiting":
				t = waiting;
				break;
			case "stopped":
				t = stopped;
				break;
			default:
				t = allTasks;
		}

		if (searchQuery) {
			const query = searchQuery.toLowerCase();
			t = t.filter((task) => {
				const fileName =
					task.bittorrent?.info?.name ||
					task.files[0]?.path?.split("/").pop() ||
					task.gid;
				return fileName.toLowerCase().includes(query);
			});
		}

		if (sortBy !== "default") {
			const sortMode = sortBy.replace(`${baseId}-sort-`, "");
			t = [...t].sort((a, b) => {
				if (sortMode === "name") {
					const nameA = a.bittorrent?.info?.name || a.files[0]?.path || "";
					const nameB = b.bittorrent?.info?.name || b.files[0]?.path || "";
					return nameA.localeCompare(nameB);
				}
				if (sortMode === "size") {
					return Number(b.totalLength) - Number(a.totalLength);
				}
				if (sortMode === "speed") {
					return Number(b.downloadSpeed) - Number(a.downloadSpeed);
				}
				if (sortMode === "progress") {
					const progA =
						Number(a.totalLength) > 0
							? Number(a.completedLength) / Number(a.totalLength)
							: 0;
					const progB =
						Number(b.totalLength) > 0
							? Number(b.completedLength) / Number(b.totalLength)
							: 0;
					return progB - progA;
				}
				return 0;
			});
		}

		return t;
	}, [status, active, waiting, stopped, allTasks, searchQuery, sortBy, baseId]);

	const handleBatchAction = async (action: "pause" | "unpause" | "remove") => {
		const gids =
			selectedGids === "all"
				? tasks.map((t) => t.gid)
				: Array.from(selectedGids as Set<string>);

		for (const gid of gids) {
			if (action === "pause") pause.mutate(gid);
			if (action === "unpause") unpause.mutate(gid);
			if (action === "remove") remove.mutate(gid);
		}
		setSelectedGids(new Set());
		setIsSelectionMode(false);
	};

	const { dragAndDropHooks } = useDragAndDrop({
		getItems: (keys) => [...keys].map((key) => ({ "text/plain": String(key) })),
		onReorder: async (e) => {
			const targetGid = String(e.target.key);
			const sourceGid = String([...e.keys][0]);

			const targetIndex = tasks.findIndex((t) => t.gid === targetGid);
			if (targetIndex === -1) return;

			let finalPos = targetIndex;
			if (e.target.dropPosition === "after") {
				finalPos += 1;
			}

			await aria2.changePosition(sourceGid, finalPos, "pos-set");
			refetch();
		},
	});

	const isAllSelected =
		selectedGids === "all" ||
		(selectedGids instanceof Set &&
			selectedGids.size === tasks.length &&
			tasks.length > 0);

	const isIndeterminate =
		selectedGids instanceof Set &&
		selectedGids.size > 0 &&
		selectedGids.size < tasks.length;

	const selectionCount =
		selectedGids === "all"
			? tasks.length
			: (selectedGids as Set<string>).size || 0;

	return (
		<div className="space-y-4">
			<div className="flex flex-col gap-4">
				<div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
					<div className="flex items-center gap-2 w-full md:w-auto ml-auto">
						{isSelectionMode ? (
							<div className="flex items-center gap-2 bg-default/30 p-1 rounded-xl animate-in fade-in zoom-in duration-200 w-full md:w-auto">
								<div className="flex items-center px-3 gap-2">
									<Checkbox
										isSelected={isAllSelected}
										isIndeterminate={isIndeterminate}
										onChange={(selected) => {
											if (selected) setSelectedGids("all");
											else setSelectedGids(new Set());
										}}
									/>
									<span className="text-xs font-bold uppercase text-muted whitespace-nowrap">
										{selectionCount} Selected
									</span>
								</div>
								<Button
									size="sm"
									variant="ghost"
									onPress={() => handleBatchAction("unpause")}
								>
									Unpause
								</Button>
								<Button
									size="sm"
									variant="ghost"
									onPress={() => handleBatchAction("pause")}
								>
									Pause
								</Button>
								<Button
									size="sm"
									variant="secondary"
									onPress={() => {
										setIsSelectionMode(false);
										setSelectedGids(new Set());
									}}
								>
									Done
								</Button>
							</div>
						) : (
							<Button
								size="sm"
								variant="secondary"
								onPress={() => setIsSelectionMode(true)}
							>
								Select Mode
							</Button>
						)}
					</div>
				</div>

				<div className="flex flex-col md:flex-row gap-3">
					<div className="relative flex-1">
						<IconMagnifier className="absolute left-3 top-1/2 -translate-y-1/2 text-muted z-10 w-4 h-4" />
						<Input
							placeholder="Search tasks by name..."
							className="pl-10 h-11 border-border"
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							fullWidth
						/>
					</div>
					<Select
						className="md:w-48"
						aria-label="Sort by"
						selectedKey={sortBy}
						onSelectionChange={(key) => {
							if (key) setSortBy(key as string);
						}}
					>
						<Select.Trigger>
							<Select.Value>
								Sort by: {sortBy.replace(`${baseId}-sort-`, "")}
							</Select.Value>
							<Select.Indicator />
						</Select.Trigger>
						<Select.Popover>
							<ListBox>
								<ListBox.Item id={`${baseId}-sort-default`}>
									Default
								</ListBox.Item>
								<ListBox.Item id={`${baseId}-sort-name`}>Name</ListBox.Item>
								<ListBox.Item id={`${baseId}-sort-size`}>Size</ListBox.Item>
								<ListBox.Item id={`${baseId}-sort-speed`}>Speed</ListBox.Item>
								<ListBox.Item id={`${baseId}-sort-progress`}>
									Progress
								</ListBox.Item>
							</ListBox>
						</Select.Popover>
					</Select>
				</div>
			</div>

			<div className="flex flex-col gap-4">
				{tasks.length === 0 ? (
					<div className="flex flex-col items-center justify-center py-12 text-muted">
						<IconArchive className="w-12 h-12 mb-4 opacity-50" />
						<p className="text-lg font-medium">No tasks found</p>
						<p className="text-sm">
							{searchQuery
								? "Try a different search query"
								: "Add a new download to get started"}
						</p>
					</div>
				) : (
					<ListBox
						aria-label="Tasks"
						dragAndDropHooks={dragAndDropHooks}
						selectionMode={isSelectionMode ? "multiple" : "none"}
						selectedKeys={selectedGids}
						onSelectionChange={setSelectedGids}
						className="space-y-4 outline-none border-none p-0 bg-transparent"
					>
						{tasks.map((task) => (
							<ListBox.Item
								key={task.gid}
								id={task.gid}
								textValue={task.bittorrent?.info?.name || task.files[0]?.path}
								className="outline-none focus:outline-none bg-transparent"
							>
								{/* @ts-ignore - contextMenu is valid at runtime for RAC triggers */}
								<Dropdown trigger="contextMenu">
									<div className="flex items-center gap-4 group">
										{isSelectionMode && (
											<Checkbox
												isSelected={
													selectedGids === "all" ||
													(selectedGids instanceof Set &&
														selectedGids.has(task.gid))
												}
											/>
										)}
										<div className="flex-1 min-w-0">
											<DownloadCard task={task} />
										</div>
									</div>

									<Dropdown.Popover className="min-w-[200px] p-1 bg-background border border-border rounded-xl shadow-xl">
										<Dropdown.Menu
											onAction={(key) => {
												const action = String(key).replace(`${baseId}-`, "");
												if (action === "pause") pause.mutate(task.gid);
												if (action === "unpause") unpause.mutate(task.gid);
												if (action === "remove") remove.mutate(task.gid);
												if (action === "copy") {
													const uri = task.files[0]?.uris[0]?.uri;
													if (uri) navigator.clipboard.writeText(uri);
												}
											}}
										>
											<Dropdown.Item
												id={
													task.status === "paused"
														? `${baseId}-unpause`
														: `${baseId}-pause`
												}
												textValue={
													task.status === "paused" ? "Resume" : "Pause"
												}
												className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 cursor-pointer outline-none flex items-center gap-2"
											>
												{task.status === "paused" ? (
													<IconPlay className="w-4 h-4 text-success" />
												) : (
													<IconPause className="w-4 h-4 text-warning" />
												)}
												<span className="text-sm font-medium">
													{task.status === "paused" ? "Resume" : "Pause"}
												</span>
											</Dropdown.Item>

											<Dropdown.Item
												id={`${baseId}-copy`}
												textValue="Copy Link"
												className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 cursor-pointer outline-none flex items-center gap-2"
											>
												<IconCopy className="w-4 h-4 text-accent" />
												<span className="text-sm font-medium">Copy Link</span>
											</Dropdown.Item>

											<Dropdown.Item
												id={`${baseId}-remove`}
												textValue="Remove"
												variant="danger"
												className="px-3 py-2 rounded-lg data-[hover=true]:bg-danger/10 text-danger cursor-pointer outline-none flex items-center gap-2"
											>
												<IconTrashBin className="w-4 h-4" />
												<span className="text-sm font-medium">Remove</span>
											</Dropdown.Item>
										</Dropdown.Menu>
									</Dropdown.Popover>
								</Dropdown>
							</ListBox.Item>
						))}
					</ListBox>
				)}
			</div>
		</div>
	);
};
