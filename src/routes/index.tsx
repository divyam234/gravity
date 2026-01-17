import { Button, Kbd, Tooltip } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconGear from "~icons/gravity-ui/gear";
import IconPlus from "~icons/gravity-ui/plus";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { StatsOverview } from "../components/dashboard/StatsOverview";
import { TaskList } from "../components/dashboard/TaskList";
import {
	activeTasksOptions,
	globalStatOptions,
	stoppedTasksOptions,
	useAria2Actions,
	waitingTasksOptions,
} from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/")({
	component: Dashboard,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl, pollingInterval } = useSettingsStore.getState();

		// Prefetch essential data for the dashboard
		await Promise.all([
			queryClient.ensureQueryData(globalStatOptions(rpcUrl, pollingInterval)),
			queryClient.ensureQueryData(activeTasksOptions(rpcUrl, pollingInterval)),
			queryClient.ensureQueryData(
				waitingTasksOptions(rpcUrl, pollingInterval, 0, 50),
			),
			queryClient.ensureQueryData(
				stoppedTasksOptions(rpcUrl, pollingInterval, 0, 50),
			),
		]);
	},
});

function Dashboard() {
	useNotifications();
	const navigate = useNavigate();
	const { purgeDownloadResult } = useAria2Actions();

	return (
		<div className="space-y-6">
			{/* Toolbar */}
			<div className="flex justify-between items-center">
				<h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
				<div className="flex gap-2">
					<Tooltip>
						<Tooltip.Trigger>
							<Button
								variant="tertiary"
								onPress={() => purgeDownloadResult.mutate()}
								isDisabled={purgeDownloadResult.isPending}
							>
								<IconTrashBin className="w-4.5 h-4.5" />
								Purge Finished
							</Button>
						</Tooltip.Trigger>
						<Tooltip.Content className="p-2 text-tiny flex items-center gap-2">
							Clear finished tasks <Kbd>Shift + C</Kbd>
						</Tooltip.Content>
					</Tooltip>

					<Tooltip>
						<Tooltip.Trigger>
							<Button
								onPress={() => navigate({ to: "/settings" })}
								variant="secondary"
							>
								<IconGear className="w-4.5 h-4.5" />
								Settings
							</Button>
						</Tooltip.Trigger>
						<Tooltip.Content className="p-2 text-tiny flex items-center gap-2">
							Configure app <Kbd>Shift + S</Kbd>
						</Tooltip.Content>
					</Tooltip>

					<Tooltip>
						<Tooltip.Trigger>
							<Button onPress={() => navigate({ to: "/add" })}>
								<IconPlus className="w-5 h-5" />
								Add Task
							</Button>
						</Tooltip.Trigger>
						<Tooltip.Content className="p-2 text-tiny flex items-center gap-2">
							New download <Kbd>Shift + A</Kbd>
						</Tooltip.Content>
					</Tooltip>
				</div>
			</div>

			{/* Stats */}
			<StatsOverview />

			{/* List */}
			<TaskList />
		</div>
	);
}
