import { Label, ListBox, Select, Slider } from "@heroui/react";
import type React from "react";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const AppSettings: React.FC = () => {
	const { pollingInterval, setPollingInterval, theme, setTheme } =
		useSettingsStore();

	return (
		<div className="space-y-8">
			<div className="border-b border-border pb-2">
				<h3 className="text-lg font-bold">UI Preferences</h3>
				<p className="text-sm text-muted">Customize the dashboard behavior.</p>
			</div>

			<div className="space-y-6">
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

				<div className="flex flex-col gap-2">
					<Label className="text-sm font-bold tracking-tight">Appearance</Label>
					<Select
						selectedKey={theme}
						onSelectionChange={(key) => setTheme(key as any)}
						className="max-w-[240px]"
					>
						<Select.Trigger className="h-11 px-4 bg-default/10 rounded-2xl hover:bg-default/20 transition-colors border-none outline-none">
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
		</div>
	);
};
