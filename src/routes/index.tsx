import { Button } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconGear from "~icons/gravity-ui/gear";
import IconPlus from "~icons/gravity-ui/plus";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { StatsOverview } from "../components/dashboard/StatsOverview";
import { TaskList } from "../components/dashboard/TaskList";
import { useAria2Actions } from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";

export const Route = createFileRoute("/")({
	component: Dashboard,
});

function Dashboard() {
	useNotifications();
	const navigate = useNavigate();
	const { purgeDownloadResult } = useAria2Actions();

	return (
		<div className="space-y-6">
			{/* Toolbar */}
			<div className="flex justify-between items-center">
				<h2 className="text-2xl font-bold">Dashboard</h2>
				<div className="flex gap-2">
					<Button
						variant="tertiary"
						onPress={() => purgeDownloadResult.mutate()}
						isDisabled={purgeDownloadResult.isPending}
					>
						<IconTrashBin className="w-4.5 h-4.5" />
						Purge Finished
					</Button>
					<Button
						onPress={() => navigate({ to: "/settings" })}
						variant="secondary"
					>
						<IconGear className="w-4.5 h-4.5" />
						Settings
					</Button>
					<Button onPress={() => navigate({ to: "/add" })}>
						<IconPlus className="w-5 h-5" />
						Add Task
					</Button>
				</div>
			</div>

			{/* Stats */}
			<StatsOverview />

			{/* List */}
			<TaskList />
		</div>
	);
}
