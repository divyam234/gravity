import { createFileRoute } from "@tanstack/react-router";
import { StatsOverview } from "../components/dashboard/StatsOverview";
import { globalStatOptions } from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/")({
	component: Dashboard,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl, pollingInterval } = useSettingsStore.getState();

		if (!rpcUrl) return;

		// Prefetch essential data for the dashboard without blocking
		queryClient.prefetchQuery(globalStatOptions(rpcUrl, pollingInterval));
	},
});

function Dashboard() {
	useNotifications();

	return (
		<div className="space-y-6">
			{/* Toolbar */}
			<div className="flex justify-between items-center">
				<h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
			</div>

			<StatsOverview />

			<div className="grid grid-cols-1 md:grid-cols-2 gap-6">
				{/* You could add more overview cards here later */}
			</div>
		</div>
	);
}
