import { Button, Checkbox, Dropdown, ListBox, Tooltip } from "@heroui/react";
import React, { useId } from "react";
import { useDragAndDrop } from "react-aria-components";
import IconArchive from "~icons/gravity-ui/archive";
import IconCopy from "~icons/gravity-ui/copy";
import IconLayoutCellsLarge from "~icons/gravity-ui/layout-cells-large";
import IconListUl from "~icons/gravity-ui/list-ul";
import IconPause from "~icons/gravity-ui/pause";
import IconPlay from "~icons/gravity-ui/play";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useAllTasks, useAria2Actions } from "../../hooks/useAria2";
import type { Aria2Task } from "../../lib/aria2-rpc";
import { aria2 } from "../../lib/aria2-rpc";
import { cn } from "../../lib/utils";
import { useSettingsStore } from "../../store/useSettingsStore";
import { DownloadCard } from "./DownloadCard";

interface TaskListProps {
	status: string;
}

export const TaskList: React.FC<TaskListProps> = ({ status }) => {
	const { active, waiting, stopped, refetch } = useAllTasks();
	const { pause, unpause, remove } = useAria2Actions();
	const baseId = useId();
	const {
		viewMode,
		setViewMode,
		searchQuery,
		isSelectionMode,
		setIsSelectionMode,
		selectedGids,
		toggleGidSelection,
		setSelectedGids,
	} = useSettingsStore();

	const allTasks = React.useMemo(
		() => [...active, ...waiting, ...stopped],
		[active, waiting, stopped],
	);

	const isAllSelected =
		allTasks.length > 0 && selectedGids.size === allTasks.length;
	const isIndeterminate =
		selectedGids.size > 0 && selectedGids.size < allTasks.length;

	const handleBatchAction = async (action: "pause" | "unpause") => {
		const gids = Array.from(selectedGids);
		for (const gid of gids) {
			if (action === "pause") pause.mutate(gid);
			if (action === "unpause") unpause.mutate(gid);
		}
		setIsSelectionMode(false);
	};

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

		return t;
	}, [status, active, waiting, stopped, allTasks, searchQuery]);

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

	return (
		<div className="space-y-4">
			<div className="flex flex-col md:flex-row justify-between items-center gap-4 bg-muted-background/20 p-2 rounded-2xl border border-border">
				<div className="flex items-center gap-2">
					{isSelectionMode ? (
						<div className="flex items-center gap-2 animate-in fade-in zoom-in duration-200">
							<div className="flex items-center px-2 gap-3">
								<Checkbox
									isSelected={isAllSelected}
									isIndeterminate={isIndeterminate}
									onChange={(selected) => {
										if (selected)
											setSelectedGids(new Set(allTasks.map((t) => t.gid)));
										else setSelectedGids(new Set());
									}}
								/>
								<span className="text-xs font-black uppercase text-muted whitespace-nowrap tracking-widest">
									{selectedGids.size} Selected
								</span>
							</div>
							<div className="w-px h-4 bg-border mx-1" />
							<Button
								size="sm"
								variant="ghost"
								onPress={() => handleBatchAction("unpause")}
								className="h-8 text-xs font-bold"
							>
								Start
							</Button>
							<Button
								size="sm"
								variant="ghost"
								onPress={() => handleBatchAction("pause")}
								className="h-8 text-xs font-bold"
							>
								Pause
							</Button>
							<Button
								size="sm"
								variant="secondary"
								onPress={() => {
									setIsSelectionMode(false);
								}}
								className="h-8 text-xs font-bold px-4"
							>
								Done
							</Button>
						</div>
					) : (
						<Button
							size="sm"
							variant="ghost"
							onPress={() => setIsSelectionMode(true)}
							className="h-9 px-4 text-xs font-bold rounded-xl"
						>
							Select Mode
						</Button>
					)}
				</div>

				<div className="flex items-center gap-1.5 bg-default/10 p-1 rounded-xl">
					<Tooltip>
						<Tooltip.Trigger>
							<Button
								isIconOnly
								size="sm"
								variant={viewMode === "list" ? "secondary" : "ghost"}
								onPress={() => setViewMode("list")}
								className="h-8 w-8 min-w-0"
							>
								<IconListUl className="w-4 h-4" />
							</Button>
						</Tooltip.Trigger>
						<Tooltip.Content>List View</Tooltip.Content>
					</Tooltip>
					<Tooltip>
						<Tooltip.Trigger>
							<Button
								isIconOnly
								size="sm"
								variant={viewMode === "grid" ? "secondary" : "ghost"}
								onPress={() => setViewMode("grid")}
								className="h-8 w-8 min-w-0"
							>
								<IconLayoutCellsLarge className="w-4 h-4" />
							</Button>
						</Tooltip.Trigger>
						<Tooltip.Content>Grid View</Tooltip.Content>
					</Tooltip>
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
						className={cn(
							"outline-none border-none p-0 bg-transparent",
							viewMode === "grid"
								? "grid grid-cols-1 lg:grid-cols-2 gap-4"
								: "flex flex-col border border-border rounded-[32px] bg-muted-background/20 overflow-hidden divide-y divide-border",
						)}
					>
						{tasks.map((task) => (
							<ListBox.Item
								key={task.gid}
								id={task.gid}
								textValue={task.bittorrent?.info?.name || task.files[0]?.path}
								className="outline-none focus:outline-none bg-transparent p-0 w-full"
							>
								{/* @ts-ignore - contextMenu is valid at runtime for RAC triggers */}
								<Dropdown trigger="contextMenu">
									<div className="flex items-center gap-4 group w-full">
										{isSelectionMode && (
											<div className="pl-6">
												<Checkbox
													isSelected={selectedGids.has(task.gid)}
													onChange={() => toggleGidSelection(task.gid)}
												/>
											</div>
										)}
										<div className="flex-1 min-w-0 w-full">
											<DownloadCard task={task} variant={viewMode} />
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
