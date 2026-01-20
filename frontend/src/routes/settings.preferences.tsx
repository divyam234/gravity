import { Button, Card, Label, ListBox, ScrollShadow, Select, Switch } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useRef } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconMoon from "~icons/gravity-ui/moon";
import IconSun from "~icons/gravity-ui/sun";
import IconDisplay from "~icons/gravity-ui/display";
import { useSettingsStore } from "../store/useSettingsStore";
import { useServerSettingsActions } from "../hooks/useServerSettings";
import { cn } from "../lib/utils";

export const Route = createFileRoute("/settings/preferences")({
	component: PreferencesSettingsPage,
});

function PreferencesSettingsPage() {
	const navigate = useNavigate();
	const fileInputRef = useRef<HTMLInputElement>(null);
	const {
		theme,
		setTheme,
		pollingInterval,
		setPollingInterval,
		enableNotifications,
		setEnableNotifications,
		viewMode,
		setViewMode,
	} = useSettingsStore();

	const { resetSettings, importSettings, handleExport } = useServerSettingsActions();

	const handleImportClick = () => {
		fileInputRef.current?.click();
	};

	const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0];
		if (!file) return;

		const reader = new FileReader();
		reader.onload = (event) => {
			try {
				const content = event.target?.result as string;
				const settings = JSON.parse(content);
				importSettings.mutate(settings);
			} catch (err) {
				console.error("Invalid JSON", err);
			}
		};
		reader.readAsText(file);
		// Reset input
		e.target.value = "";
	};

	const themes = [
		{ id: "light", label: "Light", icon: <IconSun className="w-4 h-4" /> },
		{ id: "dark", label: "Dark", icon: <IconMoon className="w-4 h-4" /> },
		{ id: "system", label: "System", icon: <IconDisplay className="w-4 h-4" /> },
	] as const;

	const accentColors = [
		{ id: "purple", color: "bg-purple-500", label: "Purple" },
		{ id: "blue", color: "bg-blue-500", label: "Blue" },
		{ id: "green", color: "bg-green-500", label: "Green" },
		{ id: "orange", color: "bg-orange-500", label: "Orange" },
		{ id: "red", color: "bg-red-500", label: "Red" },
		{ id: "pink", color: "bg-pink-500", label: "Pink" },
	];

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center gap-4 px-2 shrink-0">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/settings" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<div>
					<h2 className="text-2xl font-bold tracking-tight">Preferences</h2>
					<p className="text-xs text-muted">Theme, notifications & behavior</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Appearance */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Appearance</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-3">
									<Label className="text-sm font-bold">Theme</Label>
									<div className="flex gap-2">
										{themes.map((t) => (
											<Button
												key={t.id}
												size="sm"
												variant={theme === t.id ? "primary" : "secondary"}
												onPress={() => setTheme(t.id)}
												className="rounded-xl font-bold flex items-center gap-2"
											>
												{t.icon}
												{t.label}
											</Button>
										))}
									</div>
								</div>

								<div className="h-px bg-border" />

								<div className="space-y-3">
									<Label className="text-sm font-bold">Accent Color</Label>
									<div className="flex gap-3">
										{accentColors.map((color) => (
											<button
												key={color.id}
												type="button"
												onClick={() => {
													// TODO: Implement accent color switching
												}}
												className={cn(
													"w-10 h-10 rounded-xl transition-all",
													color.color,
													color.id === "purple" && "ring-2 ring-offset-2 ring-offset-background ring-accent"
												)}
												title={color.label}
											/>
										))}
									</div>
									<p className="text-xs text-muted">
										Accent color customization coming soon
									</p>
								</div>

								<div className="h-px bg-border" />

								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Default View Mode</Label>
										<p className="text-xs text-muted mt-0.5">How tasks are displayed in lists</p>
									</div>
									<div className="flex gap-2">
										<Button
											size="sm"
											variant={viewMode === "list" ? "primary" : "secondary"}
											onPress={() => setViewMode("list")}
											className="rounded-xl font-bold"
										>
											List
										</Button>
										<Button
											size="sm"
											variant={viewMode === "grid" ? "primary" : "secondary"}
											onPress={() => setViewMode("grid")}
											className="rounded-xl font-bold"
										>
											Grid
										</Button>
									</div>
								</div>
							</Card>
						</section>

						{/* Notifications */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Notifications</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Desktop Notifications</Label>
										<p className="text-xs text-muted mt-0.5">Get notified when downloads complete or fail</p>
									</div>
									<Switch
										isSelected={enableNotifications}
										onChange={setEnableNotifications}
									>
										<Switch.Control>
											<Switch.Thumb />
										</Switch.Control>
									</Switch>
								</div>

								{enableNotifications && (
									<>
										<div className="h-px bg-border" />
										
										<div className="space-y-4 animate-in slide-in-from-top-2 duration-200">
											<p className="text-xs text-muted uppercase font-black tracking-widest">
												Notify me when:
											</p>
											<div className="grid grid-cols-2 gap-4">
												<label className="flex items-center gap-3 p-3 rounded-xl bg-default/5 border border-border cursor-pointer hover:border-accent/30 transition-colors">
													<input type="checkbox" defaultChecked className="w-4 h-4 accent-accent" />
													<span className="text-sm font-medium">Download completes</span>
												</label>
												<label className="flex items-center gap-3 p-3 rounded-xl bg-default/5 border border-border cursor-pointer hover:border-accent/30 transition-colors">
													<input type="checkbox" defaultChecked className="w-4 h-4 accent-accent" />
													<span className="text-sm font-medium">Upload completes</span>
												</label>
												<label className="flex items-center gap-3 p-3 rounded-xl bg-default/5 border border-border cursor-pointer hover:border-accent/30 transition-colors">
													<input type="checkbox" defaultChecked className="w-4 h-4 accent-accent" />
													<span className="text-sm font-medium">Download fails</span>
												</label>
												<label className="flex items-center gap-3 p-3 rounded-xl bg-default/5 border border-border cursor-pointer hover:border-accent/30 transition-colors">
													<input type="checkbox" className="w-4 h-4 accent-accent" />
													<span className="text-sm font-medium">Queue becomes empty</span>
												</label>
											</div>
										</div>
									</>
								)}
							</Card>
						</section>

						{/* Performance */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Performance</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Update Interval</Label>
										<p className="text-xs text-muted mt-0.5">How often to refresh task status</p>
									</div>
									<Select
										selectedKey={String(pollingInterval)}
										onSelectionChange={(key) => setPollingInterval(Number(key))}
										className="w-32"
									>
										<Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl border-none">
											<Select.Value className="text-sm font-bold" />
											<Select.Indicator className="text-muted">
												<IconChevronDown className="w-4 h-4" />
											</Select.Indicator>
										</Select.Trigger>
										<Select.Popover className="min-w-[140px] p-2 bg-background border border-border rounded-2xl shadow-xl">
											<ListBox items={[
												{ id: "500", name: "500ms (Fast)" },
												{ id: "1000", name: "1 second" },
												{ id: "2000", name: "2 seconds" },
												{ id: "5000", name: "5 seconds" },
											]}>
												{(item) => (
													<ListBox.Item
														id={item.id}
														textValue={item.name}
														className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 text-sm cursor-pointer outline-none"
													>
														<Label>{item.name}</Label>
													</ListBox.Item>
												)}
											</ListBox>
										</Select.Popover>
									</Select>
								</div>
								<p className="text-xs text-muted">
									Lower intervals mean more real-time updates but use more resources
								</p>
							</Card>
						</section>

						{/* Keyboard Shortcuts */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Keyboard Shortcuts</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border">
								<div className="space-y-3">
									<div className="flex items-center justify-between py-2">
										<span className="text-sm font-medium">Add Download</span>
										<kbd className="px-2 py-1 bg-default/10 rounded-lg text-xs font-mono font-bold">Shift + A</kbd>
									</div>
									<div className="h-px bg-border" />
									<div className="flex items-center justify-between py-2">
										<span className="text-sm font-medium">Purge Completed</span>
										<kbd className="px-2 py-1 bg-default/10 rounded-lg text-xs font-mono font-bold">Shift + C</kbd>
									</div>
									<div className="h-px bg-border" />
									<div className="flex items-center justify-between py-2">
										<span className="text-sm font-medium">Toggle Theme</span>
										<kbd className="px-2 py-1 bg-default/10 rounded-lg text-xs font-mono font-bold">Shift + T</kbd>
									</div>
								</div>
							</Card>
						</section>

						{/* Data */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Data & Backup</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-4">
								<div className="flex gap-3">
									<input
										type="file"
										ref={fileInputRef}
										onChange={handleFileChange}
										className="hidden"
										accept=".json"
									/>
									<Button
										variant="secondary"
										className="rounded-xl font-bold"
										onPress={handleExport}
									>
										📥 Export Settings
									</Button>
									<Button
										variant="secondary"
										className="rounded-xl font-bold"
										onPress={handleImportClick}
										isPending={importSettings.isPending}
									>
										📤 Import Settings
									</Button>
								</div>

								<div className="h-px bg-border" />

								<Button
									variant="ghost"
									className="rounded-xl font-bold text-danger hover:bg-danger/10"
									onPress={() => {
										if (confirm("Reset all settings to defaults? This cannot be undone.")) {
											resetSettings.mutate();
										}
									}}
									isPending={resetSettings.isPending}
								>
									⚠️ Reset All Settings
								</Button>
							</Card>
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}
