import {
	Description,
	Input,
	Label,
	Slider,
	Switch,
	TextField,
} from "@heroui/react";
import type React from "react";

export const Aria2GeneralSettings: React.FC<{ options: any }> = ({
	options,
}) => {
	return (
		<div className="space-y-6">
			<div className="border-b border-default-100 pb-2">
				<h3 className="text-lg font-bold">General aria2 Options</h3>
			</div>
			<TextField>
				<Label className="text-small font-medium block mb-2">
					Download Directory
				</Label>
				<Input defaultValue={options.dir} />
				<Description>Default path for downloaded files.</Description>
			</TextField>

			<div className="space-y-4">
				<Label className="text-small font-medium">
					Max Concurrent Downloads
				</Label>
				<Slider
					defaultValue={Number(options["max-concurrent-downloads"])}
					minValue={1}
					maxValue={16}
				>
					<Slider.Track>
						<Slider.Fill />
						<Slider.Thumb />
					</Slider.Track>
					<Slider.Output />
				</Slider>
			</div>

			<Switch defaultSelected={options.continue === "true"}>
				<Switch.Control>
					<Switch.Thumb />
				</Switch.Control>
				<Label className="ml-2">Continue partially downloaded files</Label>
			</Switch>
		</div>
	);
};
