import { Chip, Description, Input, Label, TextField } from "@heroui/react";
import type React from "react";
import IconCircleInfo from "~icons/gravity-ui/circle-info";
import { useAria2Version } from "../../../hooks/useAria2";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const ConnectionSettings: React.FC = () => {
	const { rpcUrl, setRpcUrl, rpcSecret, setRpcSecret } = useSettingsStore();
	const { data: version } = useAria2Version();

	return (
		<div className="space-y-8">
			<div className="space-y-6">
				<div className="border-b border-default-100 pb-2">
					<h3 className="text-lg font-bold">RPC Connection</h3>
					<p className="text-small text-default-500">
						Configure how the dashboard connects to aria2c.
					</p>
				</div>

				<TextField>
					<Label className="text-small font-medium block mb-2">
						RPC Server URL
					</Label>
					<Input
						value={rpcUrl}
						onChange={(e) => setRpcUrl(e.target.value)}
						placeholder="http://localhost:6800/jsonrpc"
					/>
					<Description>
						The JSON-RPC endpoint of your aria2 instance.
					</Description>
				</TextField>

				<TextField>
					<Label className="text-small font-medium block mb-2">
						RPC Secret Token
					</Label>
					<Input
						type="password"
						value={rpcSecret}
						onChange={(e) => setRpcSecret(e.target.value)}
						placeholder="Leave empty if not set"
					/>
					<Description>Passed as 'token:SECRET' in RPC calls.</Description>
				</TextField>
			</div>

			{version && (
				<div className="space-y-4 pt-4 border-t border-default-100">
					<div className="flex items-center gap-2">
						<IconCircleInfo className="w-4 h-4 text-primary" />
						<h4 className="text-small font-bold uppercase tracking-wider text-default-600">
							Aria2 Instance Information
						</h4>
					</div>

					<div className="bg-default-50 p-4 rounded-2xl border border-default-100 space-y-4">
						<div className="flex justify-between items-center">
							<span className="text-small text-default-500">Version</span>
							<Chip
								size="sm"
								variant="soft"
								color="accent"
								className="font-mono font-bold"
							>
								v{version.version}
							</Chip>
						</div>

						<div className="space-y-2">
							<span className="text-tiny text-default-500 uppercase font-black">
								Enabled Features
							</span>
							<div className="flex flex-wrap gap-1.5">
								{version.enabledFeatures.map((feature) => (
									<Chip
										key={feature}
										size="sm"
										variant="soft"
										className="text-[10px] uppercase font-bold"
									>
										{feature}
									</Chip>
								))}
							</div>
						</div>
					</div>
				</div>
			)}
		</div>
	);
};
