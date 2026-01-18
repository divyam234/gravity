import { createFileRoute } from "@tanstack/react-router";
import { TaskList } from "../components/dashboard/TaskList";
import { TaskPageHeader } from "../components/dashboard/TaskPageHeader";
import { stoppedTasksOptions } from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/tasks/stopped")({
	component: StoppedTasksPage,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl, pollingInterval } = useSettingsStore.getState();
		if (!rpcUrl) return;
		await queryClient.ensureQueryData(
			stoppedTasksOptions(rpcUrl, pollingInterval, 0, 50),
		);
	},
});

function StoppedTasksPage() {
	useNotifications();

	return (
		<div className="space-y-6">
			<TaskPageHeader title="Stopped / Finished" titleColor="text-danger" />
			<TaskList status="stopped" />
		</div>
	);
}
