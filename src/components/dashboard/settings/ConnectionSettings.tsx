import { Description, Input, Label, TextField } from "@heroui/react";
import type React from "react";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const ConnectionSettings: React.FC = () => {
	const { rpcUrl, setRpcUrl, rpcSecret, setRpcSecret } = useSettingsStore();

	return (
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
				<Description>The JSON-RPC endpoint of your aria2 instance.</Description>
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
	);
};
