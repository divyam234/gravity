import type React from "react";
import { useEngineActions } from "../../../hooks/useEngine";
import {
	engineAllOptions,
	engineGlobalAvailableOptions,
} from "../../../lib/engine-options";
import { SettingField } from "./SettingField";

export const EngineCategorySettings: React.FC<{
	category: string;
	options: Record<string, string>;
}> = ({ category, options }) => {
	const { changeGlobalOption } = useEngineActions();

	// Find the internal category key (e.g. "basicOptions" for "Basic Options")
	const categoryKey =
		Object.keys(engineGlobalAvailableOptions).find(
			(key) => key.toLowerCase() === category.replace(/\s+/g, "").toLowerCase(),
		) || category;

	const optionNames = engineGlobalAvailableOptions[categoryKey] || [];
	const metadata = optionNames
		.map((name) => ({ ...engineAllOptions[name], name }))
		.filter((opt) => !!opt.type);

	const handleUpdate = (name: string, value: string) => {
		changeGlobalOption.mutate({ body: { [name]: value } } as any);
	};

	// Some specific options are read-only at runtime
	const isReadOnly = (name: string) => {
		return (
			engineAllOptions[name]?.readonly ||
			["file-allocation", "disk-cache"].includes(name)
		);
	};

	return (
		<div className="space-y-8">
			<div className="flex flex-col">
				{metadata.map((opt) => (
					<SettingField
						key={opt.name}
						opt={opt as typeof engineAllOptions[string] & { name: string }}
						value={options[opt.name] ?? opt.defaultValue ?? ""}
						onUpdate={handleUpdate}
						isReadOnly={isReadOnly(opt.name)}
					/>
				))}
			</div>
		</div>
	);
};
