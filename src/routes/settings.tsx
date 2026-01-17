import { Button, Tabs } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import React, { useId } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconDisplay from "~icons/gravity-ui/display";
import IconGear from "~icons/gravity-ui/gear";
import IconGlobe from "~icons/gravity-ui/globe";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import IconShieldCheck from "~icons/gravity-ui/shield-check";
import { AppSettings } from "../components/dashboard/settings/AppSettings";
import { Aria2AdvancedSettings } from "../components/dashboard/settings/Aria2AdvancedSettings";
import { Aria2FullOptionsSettings } from "../components/dashboard/settings/Aria2FullOptionsSettings";
import { Aria2GeneralSettings } from "../components/dashboard/settings/Aria2GeneralSettings";
import { Aria2NetworkSettings } from "../components/dashboard/settings/Aria2NetworkSettings";
import { ConnectionSettings } from "../components/dashboard/settings/ConnectionSettings";
import { globalOptionOptions, useGlobalOption } from "../hooks/useAria2";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/settings")({
	component: SettingsPage,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl } = useSettingsStore.getState();
		await queryClient.ensureQueryData(globalOptionOptions(rpcUrl));
	},
});

function SettingsPage() {
	const navigate = useNavigate();
	const { data: options } = useGlobalOption();
	const [selectedTab, setSelectedTab] = React.useState<React.Key>("connection");
	const baseId = useId();

	return (
		<div className="max-w-6xl mx-auto space-y-6">
			<div className="flex items-center gap-4">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<h2 className="text-2xl font-bold tracking-tight">Settings</h2>
			</div>

			<div className="bg-default-50/50 rounded-3xl border border-default-100 flex flex-col md:flex-row overflow-hidden min-h-[600px]">
				{/* Sidebar Tabs */}
				<div className="w-full md:w-64 border-b md:border-b-0 md:border-r border-default-100 p-6 bg-default-50/30">
					<Tabs
						aria-label="Settings Categories"
						orientation="vertical"
						selectedKey={selectedTab as string}
						onSelectionChange={setSelectedTab}
						className="w-full"
					>
						<Tabs.ListContainer>
							<Tabs.List className="w-full">
								<Tabs.Tab id={`${baseId}-connection`}>
									<div className="flex items-center gap-2">
										<IconGlobe className="w-4 h-4" />
										<span>Connection</span>
									</div>
									<Tabs.Indicator />
								</Tabs.Tab>
								<Tabs.Tab id={`${baseId}-app`}>
									<div className="flex items-center gap-2">
										<IconDisplay className="w-4 h-4" />
										<span>App Prefs</span>
									</div>
									<Tabs.Indicator />
								</Tabs.Tab>
								<Tabs.Tab id={`${baseId}-all-aria2`}>
									<div className="flex items-center gap-2">
										<IconGear className="w-4 h-4" />
										<span>Aria2 Options</span>
									</div>
									<Tabs.Indicator />
								</Tabs.Tab>
								<Tabs.Tab id={`${baseId}-general`}>
									<div className="flex items-center gap-2">
										<IconGear className="w-4 h-4" />
										<span>Aria2 General</span>
									</div>
									<Tabs.Indicator />
								</Tabs.Tab>
								<Tabs.Tab id={`${baseId}-connection-aria2`}>
									<div className="flex items-center gap-2">
										<IconNodesDown className="w-4 h-4" />
										<span>Aria2 Network</span>
									</div>
									<Tabs.Indicator />
								</Tabs.Tab>
								<Tabs.Tab id={`${baseId}-advanced`}>
									<div className="flex items-center gap-2">
										<IconShieldCheck className="w-4 h-4" />
										<span>Advanced</span>
									</div>
									<Tabs.Indicator />
								</Tabs.Tab>
							</Tabs.List>
						</Tabs.ListContainer>
					</Tabs>
				</div>

				{/* Content Area */}
				<div className="flex-1 overflow-y-auto p-8 bg-background">
					{selectedTab === `${baseId}-connection` && <ConnectionSettings />}
					{selectedTab === `${baseId}-app` && <AppSettings />}
					{selectedTab === `${baseId}-all-aria2` && options && (
						<Aria2FullOptionsSettings options={options} />
					)}

					{options && (
						<div className="max-w-3xl">
							{selectedTab === `${baseId}-general` && (
								<Aria2GeneralSettings options={options} />
							)}
							{selectedTab === `${baseId}-connection-aria2` && (
								<Aria2NetworkSettings options={options} />
							)}
							{selectedTab === `${baseId}-advanced` && (
								<Aria2AdvancedSettings options={options} />
							)}
						</div>
					)}
				</div>
			</div>
		</div>
	);
}
