import { Input, Label, ListBox, Select, Slider, Switch, TextField } from "@heroui/react";
import type React from "react";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconFolder from "~icons/gravity-ui/folder";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const AppSettings: React.FC = () => {
	const {
		pollingInterval,
		setPollingInterval,
		theme,
		setTheme,
		enableNotifications,
		setEnableNotifications,
        serverSettings,
        updateServerSettings,
	} = useSettingsStore();

	const handleDownloadDirChange = (val: string) => {
		updateServerSettings((prev) => ({
            ...prev,
            download: {
                ...prev.download,
                downloadDir: val,
            }
        }));
	};

	return (
		<div className="space-y-8">
			<div className="border-b border-border pb-2">
				<h3 className="text-lg font-bold">Downloads</h3>
				<p className="text-sm text-muted">Configure download behavior.</p>
			</div>

			<div className="space-y-6">
				<TextField className="flex flex-col gap-2">
					<Label className="text-sm font-bold tracking-tight">
						Download Directory
					</Label>
					<div className="relative">
						<IconFolder className="absolute left-3 top-1/2 -translate-y-1/2 text-muted z-10 w-4 h-4" />
						<Input
							value={serverSettings?.download.downloadDir || ""}
							onChange={(e) => handleDownloadDirChange(e.target.value)}
							placeholder="/downloads"
							className="pl-10 h-10 bg-default/10 rounded-xl border-none focus:bg-default/20 transition-all outline-none"
							fullWidth
						/>
					</div>
					<p className="text-[10px] text-muted uppercase font-black tracking-widest">
						Default directory for downloaded files on the server.
					</p>
				</TextField>
			</div>

			<div className="border-b border-border pb-2">
				<h3 className="text-lg font-bold">UI Preferences</h3>
				<p className="text-sm text-muted">Customize the dashboard behavior.</p>
			</div>

			<div className="space-y-6">
				<div className="flex flex-col gap-4">
					<div className="flex items-center justify-between p-4 rounded-2xl bg-default/5 border border-border/50">
						<div className="flex flex-col gap-0.5">
							<Label className="text-sm font-bold tracking-tight">
								Desktop Notifications
							</Label>
							<span className="text-xs text-muted">
								Get notified when downloads complete or fail.
							</span>
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
				</div>

				<div className="space-y-4">
					<div className="flex justify-between items-center">
						<Label className="text-sm font-bold tracking-tight">
							Update Interval
						</Label>
						<span className="text-xs font-mono bg-default/10 px-2 py-0.5 rounded-md font-bold">
							{pollingInterval}ms
						</span>
					</div>
					<Slider
						value={pollingInterval}
						onChange={(val) => setPollingInterval(val as number)}
						minValue={500}
						maxValue={10000}
						step={500}
					>
						<Slider.Track className="h-1.5 bg-default/10">
							<Slider.Fill className="bg-accent" />
							<Slider.Thumb className="w-5 h-5 border-2 border-accent bg-background" />
						</Slider.Track>
					</Slider>
					<p className="text-[10px] text-muted uppercase font-black tracking-widest">
						Frequency of RPC polling. Lower is more real-time.
					</p>
				</div>

				<Select
					value={theme}
					onChange={(key) => setTheme(key as any)}
					className="flex flex-col gap-2"
				>
					<Label className="text-sm font-bold tracking-tight">Appearance</Label>
					<Select.Trigger className="h-11 px-4 bg-default/10 rounded-2xl hover:bg-default/20 transition-colors border-none outline-none max-w-[240px]">
						<Select.Value className="text-sm font-medium" />
						<Select.Indicator className="text-muted">
							<IconChevronDown className="w-4 h-4" />
						</Select.Indicator>
					</Select.Trigger>
					<Select.Popover className="min-w-[240px] p-2 bg-background border border-border rounded-3xl shadow-xl">
						<ListBox
							items={[
								{ id: "light", name: "Light" },
								{ id: "dark", name: "Dark" },
								{ id: "system", name: "System" },
							]}
						>
							{(item) => (
								<ListBox.Item
									id={item.id}
									textValue={item.name}
									className="px-4 py-2.5 rounded-xl data-[hover=true]:bg-default/15 text-sm cursor-pointer outline-none"
								>
									<Label>{item.name}</Label>
								</ListBox.Item>
							)}
						</ListBox>
					</Select.Popover>
				</Select>
			</div>
		</div>
	);
};

