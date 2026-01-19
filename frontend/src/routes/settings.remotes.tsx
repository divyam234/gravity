import { Button, ScrollShadow } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { RemoteSettings } from "../components/dashboard/settings/RemoteSettings";

export const Route = createFileRoute("/settings/remotes")({
	component: RemotesSettingsPage,
});

function RemotesSettingsPage() {
	const navigate = useNavigate();

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center gap-4 px-2 shrink-0">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<h2 className="text-2xl font-bold tracking-tight">
					Cloud Remotes
				</h2>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
                    <div className="max-w-4xl mx-auto p-8">
                        <div className="bg-background/50 rounded-3xl border border-border p-6 shadow-sm">
                            <RemoteSettings />
                        </div>
                    </div>
				</ScrollShadow>
			</div>
		</div>
	);
}
