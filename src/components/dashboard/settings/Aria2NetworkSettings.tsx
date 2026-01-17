import type React from "react";
import { useAria2Actions } from "../../../hooks/useAria2";
import aria2Options from "../../../lib/aria2-options.json";
import { SettingField } from "./SettingField";

export const Aria2NetworkSettings: React.FC<{
	options: Record<string, string>;
}> = ({ options }) => {
	const { changeGlobalOption } = useAria2Actions();

	const networkOptions = [
		"max-connection-per-server",
		"split",
		"all-proxy",
		"uri-selector",
		"stream-piece-selector",
	];
	const metadata = aria2Options.filter((opt) =>
		networkOptions.includes(opt.name),
	);

	const handleUpdate = (name: string, value: string) => {
		changeGlobalOption.mutate({ [name]: value });
	};

	return (
		<div className="space-y-6">
			<div className="border-b border-border pb-2">
				<h3 className="text-lg font-bold">Network & Connection</h3>
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
	);
};
