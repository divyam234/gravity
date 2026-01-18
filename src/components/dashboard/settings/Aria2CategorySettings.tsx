import type React from "react";
import { useAria2Actions } from "../../../hooks/useAria2";
import {
	aria2AllOptions,
	aria2GlobalAvailableOptions,
} from "../../../lib/aria2-options";
import { SettingField } from "./SettingField";

export const Aria2CategorySettings: React.FC<{
	category: string;
	options: Record<string, string>;
}> = ({ category, options }) => {
	const { changeGlobalOption } = useAria2Actions();

	// Find the internal category key (e.g. "basicOptions" for "Basic Options")
	const categoryKey =
		Object.keys(aria2GlobalAvailableOptions).find(
			(key) => key.toLowerCase() === category.replace(/\s+/g, "").toLowerCase(),
		) || category;

	const optionNames = aria2GlobalAvailableOptions[categoryKey] || [];
	const metadata = optionNames
		.map((name) => ({ ...aria2AllOptions[name], name }))
		.filter((opt) => !!opt.type);

	const handleUpdate = (name: string, value: string) => {
		changeGlobalOption.mutate({ [name]: value });
	};

	// Some specific options are read-only at runtime
	const isReadOnly = (name: string) => {
		return (
			aria2AllOptions[name]?.readonly ||
			["file-allocation", "disk-cache"].includes(name)
		);
	};

	return (
		<div className="space-y-8">
			<div className="flex flex-col">
				{metadata.map((opt) => (
					<SettingField
						key={opt.name}
						opt={opt as any}
						value={options[opt.name] ?? opt.defaultValue ?? ""}
						onUpdate={handleUpdate}
						isReadOnly={isReadOnly(opt.name)}
					/>
				))}
			</div>
		</div>
	);
};
