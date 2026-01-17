import { Button } from "@heroui/react";
import type React from "react";
import IconFloppyDisk from "~icons/gravity-ui/floppy-disk";
import { useAria2Actions } from "../../../hooks/useAria2";
import aria2Options from "../../../lib/aria2-options.json";
import { SettingField } from "./SettingField";

export const Aria2GeneralSettings: React.FC<{
	options: Record<string, string>;
}> = ({ options }) => {
	const { saveSession, changeGlobalOption } = useAria2Actions();

	const generalOptions = ["dir", "max-concurrent-downloads", "continue"];
	const metadata = aria2Options.filter((opt) =>
		generalOptions.includes(opt.name),
	);

	const handleUpdate = (name: string, value: string) => {
		changeGlobalOption.mutate({ [name]: value });
	};

	return (
		<div className="space-y-8">
			<div className="space-y-2">
				<div className="border-b border-border pb-2">
					<h3 className="text-lg font-bold">General aria2 Options</h3>
				</div>
				<div className="flex flex-col">
					{metadata.map((opt) => (
						<SettingField
							key={opt.name}
							opt={opt as any}
							value={options[opt.name] ?? opt.default ?? ""}
							onUpdate={handleUpdate}
						/>
					))}
				</div>
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
							className="font-bold"
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
