import {
	Button,
	Checkbox,
	Input,
	ListBox,
	Select,
	Tabs,
	Tooltip,
} from "@heroui/react";
import React, { useId } from "react";
import IconArchive from "~icons/gravity-ui/archive";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconChevronUp from "~icons/gravity-ui/chevron-up";
import IconMagnifier from "~icons/gravity-ui/magnifier";
import IconPause from "~icons/gravity-ui/pause";
import IconPlay from "~icons/gravity-ui/play";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useAllTasks, useAria2Actions } from "../../hooks/useAria2";
import type { Aria2Task } from "../../lib/aria2-rpc";
import { aria2 } from "../../lib/aria2-rpc";
import { DownloadCard } from "./DownloadCard";

export const TaskList: React.FC = () => {
	const { active, waiting, stopped, refetch } = useAllTasks();
	const { pause, unpause, remove } = useAria2Actions();
	const baseId = useId();
	const [selectedTab, setSelectedTab] = React.useState<React.Key>(
		`${baseId}-all`,
	);
	const [selectedGids, setSelectedGids] = React.useState<Set<string>>(
		new Set(),
	);
	const [isSelectionMode, setIsSelectionMode] = React.useState(false);
	const [searchQuery, setSearchQuery] = React.useState("");
	const [sortBy, setSortBy] = React.useState<string>("default");

	const allTasks = [...active, ...waiting, ...stopped];

	const getFilteredAndSortedTasks = () => {
		let tasks: Aria2Task[] = [];
		switch (selectedTab) {
			case `${baseId}-active`:
				tasks = active;
				break;
			case `${baseId}-waiting`:
				tasks = waiting;
				break;
			case `${baseId}-stopped`:
				tasks = stopped;
				break;
			default:
				tasks = allTasks;
		}

		// Filter by search
		if (searchQuery) {
			const query = searchQuery.toLowerCase();
			tasks = tasks.filter((t) => {
				const fileName =
					t.bittorrent?.info?.name ||
					t.files[0]?.path?.split("/").pop() ||
					t.gid;
				return fileName.toLowerCase().includes(query);
			});
		}

		// Sort
		if (sortBy !== "default") {
			const sortMode = sortBy.replace(`${baseId}-sort-`, "");
			tasks = [...tasks].sort((a, b) => {
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

		return tasks;
	};

	const tasks = getFilteredAndSortedTasks();

	const toggleSelection = (gid: string) => {
		const newSelected = new Set(selectedGids);
		if (newSelected.has(gid)) newSelected.delete(gid);
		else newSelected.add(gid);
		setSelectedGids(newSelected);
	};

	const handleBatchAction = async (action: "pause" | "unpause" | "remove") => {
		const gids = Array.from(selectedGids);
		for (const gid of gids) {
			if (action === "pause") pause.mutate(gid);
			if (action === "unpause") unpause.mutate(gid);
			if (action === "remove") remove.mutate(gid);
		}
		setSelectedGids(new Set());
		setIsSelectionMode(false);
	};

	const handleMove = async (
		gid: string,
		pos: number,
		how: "pos-set" | "pos-cur" | "pos-end",
	) => {
		await aria2.changePosition(gid, pos, how);
		refetch();
	};

	return (
		<div className="space-y-4">
			<div className="flex flex-col gap-4">
				<div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
					<Tabs
						aria-label="Filter tasks"
						selectedKey={selectedTab as string}
						onSelectionChange={(key) => {
							setSelectedTab(key);
							setSelectedGids(new Set());
						}}
					>
						<Tabs.ListContainer>
							<Tabs.List>
								<Tabs.Tab id={`${baseId}-all`}>
									All ({allTasks.length})
									<Tabs.Indicator />
								</Tabs.Tab>
								<Tabs.Tab id={`${baseId}-active`}>
									Downloading ({active.length})
									<Tabs.Indicator />
								</Tabs.Tab>
								<Tabs.Tab id={`${baseId}-waiting`}>
									Waiting ({waiting.length})
									<Tabs.Indicator />
								</Tabs.Tab>
								<Tabs.Tab id={`${baseId}-stopped`}>
									Stopped ({stopped.length})
									<Tabs.Indicator />
								</Tabs.Tab>
							</Tabs.List>
						</Tabs.ListContainer>
					</Tabs>

					<div className="flex items-center gap-2 w-full md:w-auto">
						{isSelectionMode ? (
							<div className="flex items-center gap-2 bg-default-100 p-1 rounded-xl animate-in fade-in zoom-in duration-200 w-full md:w-auto">
								<span className="text-tiny font-bold px-3 uppercase text-default-500 whitespace-nowrap">
									{selectedGids.size} Selected
								</span>
								<Button
									size="sm"
									variant="ghost"
									isIconOnly
									onPress={() => handleBatchAction("unpause")}
								>
									<IconPlay className="w-4 h-4" />
								</Button>
								<Button
									size="sm"
									variant="ghost"
									isIconOnly
									onPress={() => handleBatchAction("pause")}
								>
									<IconPause className="w-4 h-4" />
								</Button>
								<Button
									size="sm"
									variant="ghost"
									isIconOnly
									className="text-danger"
									onPress={() => handleBatchAction("remove")}
								>
									<IconTrashBin className="w-4 h-4" />
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
						<IconMagnifier className="absolute left-3 top-1/2 -translate-y-1/2 text-default-400 z-10 w-4 h-4" />
						<Input
							placeholder="Search tasks by name..."
							className="pl-10"
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
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
					<div className="flex flex-col items-center justify-center py-12 text-default-400">
						<IconArchive className="w-12 h-12 mb-4 opacity-50" />
						<p className="text-lg font-medium">No tasks found</p>
						<p className="text-sm">
							{searchQuery
								? "Try a different search query"
								: "Add a new download to get started"}
						</p>
					</div>
				) : (
					tasks.map((task: Aria2Task, index: number) => (
						<div key={task.gid} className="flex items-center gap-4 group">
							{isSelectionMode && (
								<Checkbox
									isSelected={selectedGids.has(task.gid)}
									onChange={() => toggleSelection(task.gid)}
								/>
							)}
							<div className="flex-1 min-w-0">
								<DownloadCard task={task} />
							</div>

							{!isSelectionMode &&
								(selectedTab === `${baseId}-all` ||
									selectedTab === `${baseId}-waiting` ||
									selectedTab === `${baseId}-active`) && (
									<div className="flex flex-col gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
										<Tooltip>
											<Tooltip.Trigger>
												<Button
													size="sm"
													variant="ghost"
													isIconOnly
													isDisabled={index === 0}
													onPress={() => handleMove(task.gid, 0, "pos-set")}
												>
													<IconChevronUp className="w-4 h-4" />
												</Button>
											</Tooltip.Trigger>
											<Tooltip.Content>Move to Top</Tooltip.Content>
										</Tooltip>

										<Tooltip>
											<Tooltip.Trigger>
												<Button
													size="sm"
													variant="ghost"
													isIconOnly
													isDisabled={index === tasks.length - 1}
													onPress={() => handleMove(task.gid, 0, "pos-end")}
												>
													<IconChevronDown className="w-4 h-4" />
												</Button>
											</Tooltip.Trigger>
											<Tooltip.Content>Move to Bottom</Tooltip.Content>
										</Tooltip>
									</div>
								)}
						</div>
					))
				)}
			</div>
		</div>
	);
};
