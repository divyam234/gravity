import { Input, Label } from "@heroui/react";
import React, { useEffect } from "react";
import { useSettingsStore } from "../../../store/useSettingsStore";

export const RcloneSettings: React.FC = () => {
	const { rcloneTargetRemote, setRcloneTargetRemote } = useSettingsStore();

	useEffect(() => {
		// Sync config to backend
		fetch("/api/config", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ targetRemote: rcloneTargetRemote }),
		}).catch((err) => console.error("Failed to sync config", err));
	}, [rcloneTargetRemote]);

	return (
		<div className="space-y-8">
			<div className="border-b border-border pb-2">
				<h3 className="text-lg font-bold">Rclone Integration</h3>
				<p className="text-sm text-muted">
					Configure auto-upload settings for completed downloads.
				</p>
			</div>

			<div className="space-y-6">
				<div className="space-y-4">
					<div className="flex flex-col gap-2">
						<Label className="text-sm font-bold tracking-tight">
							Target Remote
						</Label>
						<Input
							value={rcloneTargetRemote}
							onChange={(e) => setRcloneTargetRemote(e.target.value)}
							placeholder="e.g. gdrive: or remote:/path"
							className="w-full bg-default/10 rounded-xl"
						/>
						<p className="text-[10px] text-muted uppercase font-black tracking-widest">
							Files will be copied here after download completes.
						</p>
					</div>
				</div>
			</div>
		</div>
	);
};
