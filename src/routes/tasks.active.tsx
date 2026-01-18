import { createFileRoute } from "@tanstack/react-router";
import { TaskList } from "../components/dashboard/TaskList";
import { TaskPageHeader } from "../components/dashboard/TaskPageHeader";
import { activeTasksOptions } from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/tasks/active")({
	component: ActiveTasksPage,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl, pollingInterval } = useSettingsStore.getState();
		if (!rpcUrl) return;
		await queryClient.ensureQueryData(
			activeTasksOptions(rpcUrl, pollingInterval),
		);
	},
});

function ActiveTasksPage() {
	useNotifications();

	return (
		<div className="space-y-6">
			<TaskPageHeader title="Downloading" titleColor="text-success" />
			<TaskList status="active" />
		</div>
	);
}
