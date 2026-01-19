import { createFileRoute } from "@tanstack/react-router";
import { StatsOverview } from "../components/dashboard/StatsOverview";
import { RecentDownloads } from "../components/dashboard/RecentDownloads";
import { globalStatOptions } from "../hooks/useEngine";
import { useNotifications } from "../hooks/useNotifications";

export const Route = createFileRoute("/")({
	component: Dashboard,
	loader: async ({ context: { queryClient } }) => {
		queryClient.prefetchQuery(globalStatOptions());
	},
});

function Dashboard() {
	useNotifications();

	return (
		<div className="space-y-8">
			{/* Toolbar */}
			<div className="flex justify-between items-center">
				<h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
			</div>

			<StatsOverview />

            <RecentDownloads />
		</div>
	);
}
