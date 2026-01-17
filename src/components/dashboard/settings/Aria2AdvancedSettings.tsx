import type React from "react";
import aria2Options from "../../../lib/aria2-options.json";
import { SettingField } from "./SettingField";

export const Aria2AdvancedSettings: React.FC<{
	options: Record<string, string>;
}> = ({ options }) => {
	const advancedOptions = [
		"file-allocation",
		"disk-cache",
		"min-tls-version",
		"event-poll",
	];
	const metadata = aria2Options.filter((opt) =>
		advancedOptions.includes(opt.name),
	);

	return (
		<div className="space-y-6">
			<div className="border-b border-border pb-2">
				<h3 className="text-lg font-bold">Advanced Tunings</h3>
				<p className="text-xs text-muted mt-1">
					Some of these settings may require a restart to take effect.
				</p>
			</div>
			<div className="flex flex-col">
				{metadata.map((opt) => (
					<SettingField
						key={opt.name}
						opt={opt as any}
						value={options[opt.name] ?? opt.default ?? ""}
						onUpdate={() => {}} // Read-only or requires restart
						isReadOnly={
							opt.name === "file-allocation" || opt.name === "disk-cache"
						}
					/>
				))}
			</div>
		</div>
	);
};
