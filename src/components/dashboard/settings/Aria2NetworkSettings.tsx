import { Input, Label, Slider, TextField } from "@heroui/react";
import type React from "react";

export const Aria2NetworkSettings: React.FC<{ options: any }> = ({
	options,
}) => {
	return (
		<div className="space-y-6">
			<div className="border-b border-border pb-2">
				<h3 className="text-lg font-bold">Network & Connection</h3>
			</div>
			<div className="space-y-4">
				<Label className="text-sm font-medium">
					Max Connections per Server
				</Label>
				<Slider
					defaultValue={Number(options["max-connection-per-server"])}
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

			<TextField>
				<Label className="text-sm font-medium block mb-2">
					Global Proxy
				</Label>
				<Input
					defaultValue={options["all-proxy"]}
					placeholder="http://proxy:8080"
				/>
			</TextField>
		</div>
	);
};
