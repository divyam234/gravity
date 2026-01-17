import {
	Button,
	Description,
	Input,
	Label,
	Slider,
	Switch,
	TextField,
} from "@heroui/react";
import type React from "react";
import IconFloppyDisk from "~icons/gravity-ui/floppy-disk";
import { useAria2Actions } from "../../../hooks/useAria2";

export const Aria2GeneralSettings: React.FC<{ options: any }> = ({
	options,
}) => {
	const { saveSession } = useAria2Actions();

	return (
		<div className="space-y-8">
			<div className="space-y-6">
				<div className="border-b border-border pb-2">
					<h3 className="text-lg font-bold">General aria2 Options</h3>
				</div>
				<TextField>
					<Label className="text-sm font-medium block mb-2">
						Download Directory
					</Label>
					<Input defaultValue={options.dir} />
					<Description>Default path for downloaded files.</Description>
				</TextField>

				<div className="space-y-4">
					<Label className="text-sm font-medium">
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

			<div className="space-y-4 pt-6 border-t border-border">
				<h4 className="text-sm font-bold uppercase tracking-wider text-foreground/80">
					Manual Operations
				</h4>
				<div className="flex flex-col gap-4">
					<div className="bg-muted-background p-4 rounded-2xl border border-border flex items-center justify-between">
						<div className="flex flex-col gap-0.5">
							<span className="text-sm font-bold">Save Current Session</span>
							<p className="text-xs text-muted">
								Force save the current download list to the session file.
							</p>
						</div>
						<Button
							size="sm"
							variant="secondary"
							onPress={() => saveSession.mutate()}
							isDisabled={saveSession.isPending}
						>
							<IconFloppyDisk className="w-4 h-4 mr-2" />
							Save Now
						</Button>
					</div>
				</div>
			</div>
		</div>
	);
};
