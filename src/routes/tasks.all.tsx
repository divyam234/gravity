import { createFileRoute } from "@tanstack/react-router";
import React from "react";
import { TaskList } from "../components/dashboard/TaskList";
import { TaskPageHeader } from "../components/dashboard/TaskPageHeader";
import {
	activeTasksOptions,
	stoppedTasksOptions,
	waitingTasksOptions,
} from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/tasks/all")({
	component: AllTasksPage,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl, pollingInterval } = useSettingsStore.getState();
		if (!rpcUrl) return;
		await Promise.all([
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

function AllTasksPage() {
	useNotifications();

	return (
		<div className="space-y-6 px-1">
			<TaskPageHeader title="All Tasks" />
			<TaskList status="all" />
		</div>
	);
}
