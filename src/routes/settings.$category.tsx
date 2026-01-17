import { Button } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { AppSettings } from "../components/dashboard/settings/AppSettings";
import { Aria2AdvancedSettings } from "../components/dashboard/settings/Aria2AdvancedSettings";
import { Aria2FullOptionsSettings } from "../components/dashboard/settings/Aria2FullOptionsSettings";
import { Aria2GeneralSettings } from "../components/dashboard/settings/Aria2GeneralSettings";
import { Aria2NetworkSettings } from "../components/dashboard/settings/Aria2NetworkSettings";
import { ConnectionSettings } from "../components/dashboard/settings/ConnectionSettings";
import { globalOptionOptions, useGlobalOption } from "../hooks/useAria2";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/settings/$category")({
	component: SettingsPage,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl } = useSettingsStore.getState();
		if (!rpcUrl) return;
		await queryClient.ensureQueryData(globalOptionOptions(rpcUrl));
	},
});

function SettingsPage() {
	const navigate = useNavigate();
	const { category } = Route.useParams();
	const { data: options } = useGlobalOption();

	const renderContent = () => {
		switch (category) {
			case "connection":
				return <ConnectionSettings />;
			case "app":
				return <AppSettings />;
			case "aria2-options":
				return options ? <Aria2FullOptionsSettings options={options} /> : null;
			case "aria2-general":
				return options ? <Aria2GeneralSettings options={options} /> : null;
			case "aria2-network":
				return options ? <Aria2NetworkSettings options={options} /> : null;
			case "advanced":
				return options ? <Aria2AdvancedSettings options={options} /> : null;
			default:
				return <ConnectionSettings />;
		}
	};

	const getTitle = () => {
		switch (category) {
			case "connection":
				return "Connection Settings";
			case "app":
				return "App Preferences";
			case "aria2-options":
				return "Aria2 Options";
			case "aria2-general":
				return "Aria2 General";
			case "aria2-network":
				return "Aria2 Network";
			case "advanced":
				return "Advanced Settings";
			default:
				return "Settings";
		}
	};

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center gap-4 shrink-0">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/", search: { status: "all" } })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<h2 className="text-2xl font-bold tracking-tight">{getTitle()}</h2>
			</div>

			<div className="flex-1 bg-muted-background/50 rounded-3xl border border-border overflow-hidden min-h-0">
				<div className="h-full p-8 bg-background overflow-y-auto">
					<div className="max-w-4xl mx-auto">{renderContent()}</div>
				</div>
			</div>
		</div>
	);
}
