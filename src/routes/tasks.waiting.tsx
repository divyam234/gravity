import { createFileRoute } from "@tanstack/react-router";
import { TaskList } from "../components/dashboard/TaskList";
import { TaskPageHeader } from "../components/dashboard/TaskPageHeader";
import { waitingTasksOptions } from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/tasks/waiting")({
	component: WaitingTasksPage,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl, pollingInterval } = useSettingsStore.getState();
		if (!rpcUrl) return;
		await queryClient.ensureQueryData(
			waitingTasksOptions(rpcUrl, pollingInterval, 0, 50),
		);
	},
});

function WaitingTasksPage() {
	useNotifications();

	return (
		<div className="space-y-6">
			<TaskPageHeader title="Waiting" titleColor="text-warning" />
			<TaskList status="waiting" />
		</div>
	);
}
