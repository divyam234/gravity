import {
	Description,
	Input,
	Label,
	ListBox,
	Select,
	TextField,
} from "@heroui/react";
import type React from "react";
import { useId } from "react";

export const Aria2AdvancedSettings: React.FC<{ options: any }> = ({
	options,
}) => {
	const baseId = useId();

	return (
		<div className="space-y-6">
			<div className="border-b border-border pb-2">
				<h3 className="text-lg font-bold">Advanced Tunings</h3>
			</div>
			<div className="opacity-60 grayscale pointer-events-none">
				<Select defaultValue={options["file-allocation"]}>
					<Label className="text-sm font-medium block mb-2">
						File Allocation (Immutable)
					</Label>
					<Select.Trigger>
						<Select.Value />
						<Select.Indicator />
					</Select.Trigger>
					<Select.Popover>
						<ListBox>
							<ListBox.Item id={`${baseId}-none`}>None</ListBox.Item>
							<ListBox.Item id={`${baseId}-prealloc`}>Prealloc</ListBox.Item>
							<ListBox.Item id={`${baseId}-falloc`}>Falloc</ListBox.Item>
						</ListBox>
					</Select.Popover>
				</Select>
				<Description className="mt-1">
					This setting can only be changed at startup.
				</Description>
			</div>

			<TextField isReadOnly>
				<Label className="text-sm font-medium block mb-2 opacity-60">
					Disk Cache (Immutable)
				</Label>
				<Input defaultValue={options["disk-cache"]} />
			</TextField>
		</div>
	);
};
