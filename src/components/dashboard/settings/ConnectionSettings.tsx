import { Chip, FieldError, Input, Label, TextField } from "@heroui/react";
import type React from "react";
import IconCircleInfo from "~icons/gravity-ui/circle-info";
import { useAria2Version } from "../../../hooks/useAria2";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const ConnectionSettings: React.FC = () => {
	const { rpcUrl, setRpcUrl, rpcSecret, setRpcSecret } = useSettingsStore();
	const { data: version } = useAria2Version();

	const validateUrl = (val: string) => {
		if (!val) return "URL is required";
		try {
			const url = new URL(val);
			if (
				url.protocol !== "http:" &&
				url.protocol !== "https:" &&
				url.protocol !== "ws:" &&
				url.protocol !== "wss:"
			) {
				return "Must be http/https or ws/wss";
			}
		} catch {
			return "Invalid URL format";
		}
		return true;
	};

	return (
		<div className="space-y-8">
			<div className="space-y-6">
				<div className="border-b border-border pb-2">
					<h3 className="text-lg font-bold">RPC Connection</h3>
					<p className="text-sm text-muted">
						Configure how the dashboard connects to aria2c.
					</p>
				</div>

				<TextField
					value={rpcUrl}
					onChange={setRpcUrl}
					validate={validateUrl}
					validationBehavior="aria"
				>
					<div className="flex flex-col gap-2">
						<Label className="text-sm font-bold tracking-tight">
							RPC Server URL
						</Label>
						<div className="relative">
							<Input
								className="w-full h-11 px-4 bg-default/10 rounded-2xl text-sm border border-transparent focus:bg-default/20 focus:border-accent/30 transition-all outline-none data-[invalid=true]:border-danger/50"
								placeholder="http://localhost:6800/jsonrpc"
							/>
							<FieldError className="absolute -bottom-5 right-0 text-[10px] text-danger font-bold uppercase tracking-tight" />
						</div>
						<p className="text-xs text-muted">
							The JSON-RPC endpoint (HTTP or WebSocket).
						</p>
					</div>
				</TextField>

				<TextField value={rpcSecret} onChange={setRpcSecret}>
					<div className="flex flex-col gap-2">
						<Label className="text-sm font-bold tracking-tight">
							RPC Secret Token
						</Label>
						<Input
							type="password"
							className="w-full h-11 px-4 bg-default/10 rounded-2xl text-sm border border-transparent focus:bg-default/20 focus:border-accent/30 transition-all outline-none"
							placeholder="Leave empty if not set"
						/>
						<p className="text-xs text-muted">
							Passed as 'token:SECRET' in RPC calls.
						</p>
					</div>
				</TextField>
			</div>

			{version && (
				<div className="space-y-4 pt-4 border-t border-border">
					<div className="flex items-center gap-2">
						<IconCircleInfo className="w-4 h-4 text-accent" />
						<h4 className="text-sm font-bold uppercase tracking-wider text-foreground/80">
							Aria2 Instance Information
						</h4>
					</div>

					<div className="bg-muted-background p-5 rounded-3xl border border-border space-y-5 shadow-sm">
						<div className="flex justify-between items-center">
							<span className="text-sm font-bold">Version</span>
							<Chip
								size="sm"
								variant="soft"
								color="accent"
								className="font-mono font-bold"
							>
								v{version.version}
							</Chip>
						</div>

						<div className="space-y-3">
							<span className="text-xs text-muted uppercase font-black tracking-widest">
								Enabled Features
							</span>
							<div className="flex flex-wrap gap-2">
								{version.enabledFeatures.map((feature) => (
									<Chip
										key={feature}
										size="sm"
										variant="soft"
										className="text-[10px] uppercase font-bold bg-default/10"
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
