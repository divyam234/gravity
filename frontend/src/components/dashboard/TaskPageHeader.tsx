import { Button, Checkbox, Tooltip } from "@heroui/react";
import React from "react";
import IconLayoutCellsLarge from "~icons/gravity-ui/layout-cells-large";
import IconListUl from "~icons/gravity-ui/list-ul";
import { useAllTasks, useEngineActions } from "../../hooks/useEngine";
import { cn } from "../../lib/utils";
import { useSettingsStore } from "../../store/useSettingsStore";

interface TaskPageHeaderProps {
	title: string;
	titleColor?: string;
}

export const TaskPageHeader: React.FC<TaskPageHeaderProps> = ({
	title,
	titleColor,
}) => {
	const {
		viewMode,
		setViewMode,
		isSelectionMode,
		setIsSelectionMode,
		selectedGids,
		setSelectedGids,
	} = useSettingsStore();
	const { active, waiting, stopped } = useAllTasks();
	const { pause, unpause } = useEngineActions();

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

	return (
		<div className="flex justify-between items-center px-1">
			<h2 className={cn("text-2xl font-bold tracking-tight", titleColor)}>
				{title}
			</h2>
			<div className="flex items-center gap-3">
				{isSelectionMode ? (
					<div className="flex items-center gap-2 bg-default/10 p-1 rounded-xl animate-in fade-in zoom-in duration-200">
						<div className="flex items-center px-2 gap-3">
							<Checkbox
								isSelected={isAllSelected}
								isIndeterminate={isIndeterminate}
								onChange={(selected) => {
									if (selected)
										setSelectedGids(new Set(allTasks.map((t) => t.id)));
									else setSelectedGids(new Set());
								}}
							/>
							<span className="text-[10px] font-black uppercase text-muted whitespace-nowrap tracking-widest">
								{selectedGids.size} Selected
							</span>
						</div>
						<div className="w-px h-4 bg-border mx-1" />
						<Button
							size="sm"
							variant="ghost"
							onPress={() => handleBatchAction("unpause")}
							className="h-8 text-[10px] font-black uppercase tracking-widest"
						>
							Start
						</Button>
						<Button
							size="sm"
							variant="ghost"
							onPress={() => handleBatchAction("pause")}
							className="h-8 text-[10px] font-black uppercase tracking-widest"
						>
							Pause
						</Button>
						<Button
							size="sm"
							variant="secondary"
							onPress={() => setIsSelectionMode(false)}
							className="h-8 text-[10px] font-black uppercase tracking-widest px-4"
						>
							Done
						</Button>
					</div>
				) : (
					<Button
						size="sm"
						variant="ghost"
						onPress={() => setIsSelectionMode(true)}
						className="font-bold text-xs h-9 px-4 rounded-xl"
					>
						Select
					</Button>
				)}

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
		</div>
	);
};
