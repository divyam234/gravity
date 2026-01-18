import { Button, Input } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import React from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnifier from "~icons/gravity-ui/magnifier";
import { AppSettings } from "../components/dashboard/settings/AppSettings";
import { Aria2CategorySettings } from "../components/dashboard/settings/Aria2CategorySettings";
import { ConnectionSettings } from "../components/dashboard/settings/ConnectionSettings";
import { SettingField } from "../components/dashboard/settings/SettingField";
import {
	globalOptionOptions,
	useAria2Actions,
	useGlobalOption,
} from "../hooks/useAria2";
import {
	aria2AllOptions,
	aria2GlobalAvailableOptions,
} from "../lib/aria2-options";
import { formatCategoryName } from "../lib/utils";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/settings/$category")({
	component: SettingsPage,
	loader: async ({ context: { queryClient } }) => {
		const { rpcUrl } = useSettingsStore.getState();
		if (!rpcUrl) return;
		queryClient.prefetchQuery(globalOptionOptions(rpcUrl));
	},
});

function SettingsPage() {
	const navigate = useNavigate();
	const { category } = Route.useParams();
	const { data: options } = useGlobalOption();
	const { changeGlobalOption } = useAria2Actions();
	const [search, setSearch] = React.useState("");

	const handleUpdate = (name: string, value: string) => {
		changeGlobalOption.mutate({ [name]: value });
	};

	const renderContent = () => {
		if (search.trim()) {
			const filtered = Object.keys(aria2AllOptions)
				.filter(
					(name) =>
						name.toLowerCase().includes(search.toLowerCase()) ||
						aria2AllOptions[name].description
							?.toLowerCase()
							.includes(search.toLowerCase()),
				)
				.map((name) => ({ ...aria2AllOptions[name], name }));

			return (
				<div className="space-y-6">
					<div className="border-b border-border pb-2">
						<h3 className="text-lg font-bold text-accent">Search Results</h3>
						<p className="text-xs text-muted">
							Found {filtered.length} matches for "{search}"
						</p>
					</div>
					<div className="flex flex-col">
						{filtered.map((opt) => (
							<SettingField
								key={opt.name}
								opt={opt as any}
								value={options?.[opt.name] ?? opt.defaultValue ?? ""}
								onUpdate={handleUpdate}
							/>
						))}
					</div>
				</div>
			);
		}

		switch (category) {
			case "connection":
				return <ConnectionSettings />;
			case "app":
				return <AppSettings />;
			default: {
				const categoryKey = Object.keys(aria2GlobalAvailableOptions).find(
					(key) => key.toLowerCase().replace(/[^a-z0-9]+/g, "-") === category,
				);

				if (categoryKey && options) {
					return (
						<Aria2CategorySettings category={categoryKey} options={options} />
					);
				}
				return <ConnectionSettings />;
			}
		}
	};

	const getTitle = () => {
		if (search.trim()) return "Global Search";
		switch (category) {
			case "connection":
				return "Connection Settings";
			case "app":
				return "App Preferences";
			default: {
				const key = Object.keys(aria2GlobalAvailableOptions).find(
					(k) => k.toLowerCase().replace(/[^a-z0-9]+/g, "-") === category,
				);
				return key ? formatCategoryName(key) : "Settings";
			}
		}
	};

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex flex-col md:flex-row md:items-center justify-between gap-4 shrink-0">
				<div className="flex items-center gap-4 px-2">
					<Button
						variant="ghost"
						isIconOnly
						onPress={() => navigate({ to: "/" })}
					>
						<IconChevronLeft className="w-5 h-5" />
					</Button>
					<h2 className="text-2xl font-bold tracking-tight capitalize">
						{getTitle()}
					</h2>
				</div>

				<div className="relative w-full md:w-80 px-2 md:px-0">
					<IconMagnifier className="absolute left-4 top-1/2 -translate-y-1/2 text-muted z-10 w-4.5 h-4.5" />
					<Input
						placeholder="Search all 180+ options..."
						className="w-full h-11 pl-11 pr-4 bg-default/10 rounded-2xl text-sm outline-none transition-all focus:bg-default/30 focus:ring-2 focus:ring-accent/20"
						value={search}
						onChange={(e) => setSearch(e.target.value)}
					/>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<div className="h-full p-8 bg-background/50 overflow-y-auto custom-scrollbar">
					<div className="max-w-4xl mx-auto">{renderContent()}</div>
				</div>
			</div>
		</div>
	);
}
