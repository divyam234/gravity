import { Label, ListBox, Select, Slider } from "@heroui/react";
import type React from "react";
import { useId } from "react";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const AppSettings: React.FC = () => {
	const { pollingInterval, setPollingInterval, theme, setTheme } =
		useSettingsStore();
	const baseId = useId();

	return (
		<div className="space-y-6">
			<div className="border-b border-default-100 pb-2">
				<h3 className="text-lg font-bold">UI Preferences</h3>
				<p className="text-sm text-default-500">
					Customize the dashboard behavior.
				</p>
			</div>

			<div className="space-y-4">
				<Label className="text-sm font-medium">
					Update Interval ({pollingInterval}ms)
				</Label>
				<Slider
					value={pollingInterval}
					onChange={(val) => setPollingInterval(val as number)}
					minValue={500}
					maxValue={10000}
					step={500}
				>
					<Slider.Track>
						<Slider.Fill />
						<Slider.Thumb />
					</Slider.Track>
					<Slider.Output />
				</Slider>
				<p className="text-xs text-default-500">
					Frequency of RPC polling. Lower is more real-time but uses more
					CPU/Network.
				</p>
			</div>

			<div className="opacity-50 pointer-events-none">
				<Select defaultValue={theme}>
					<Label className="text-sm font-medium block mb-2">
						Appearance
					</Label>
					<Select.Trigger>
						<Select.Value />
						<Select.Indicator />
					</Select.Trigger>
					<Select.Popover>
						<ListBox onAction={(key) => setTheme(key as any)}>
							<ListBox.Item id={`${baseId}-light`}>Light</ListBox.Item>
							<ListBox.Item id={`${baseId}-dark`}>Dark</ListBox.Item>
							<ListBox.Item id={`${baseId}-system`}>System</ListBox.Item>
						</ListBox>
					</Select.Popover>
				</Select>
			</div>
		</div>
	);
};
