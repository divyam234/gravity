import { Button, ScrollShadow } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { EngineCategorySettings } from "../components/dashboard/settings/EngineCategorySettings";
import {
	globalOptionOptions,
	useGlobalOption,
} from "../hooks/useEngine";

export const Route = createFileRoute("/settings/engine")({
	component: EngineSettingsPage,
	loader: async ({ context: { queryClient } }) => {
		queryClient.prefetchQuery(globalOptionOptions());
	},
});

function EngineSettingsPage() {
	const navigate = useNavigate();
	const { data: options } = useGlobalOption();

    const sections = [
        {
            title: "Downloads",
            description: "Manage download behavior and limits.",
            component: <EngineCategorySettings category="downloads" options={options || {}} />
        },
        {
            title: "Network",
            description: "Configure connection settings and proxies.",
            component: <EngineCategorySettings category="network" options={options || {}} />
        },
        {
            title: "BitTorrent",
            description: "BitTorrent specific settings.",
            component: <EngineCategorySettings category="bittorrent" options={options || {}} />
        }
    ];

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
					Engine Options
				</h2>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
                    <div className="max-w-4xl mx-auto p-8 space-y-12">
                        {sections.map((section) => (
                            <section key={section.title} className="space-y-6">
                                <div className="space-y-1">
                                    <h3 className="text-xl font-bold">{section.title}</h3>
                                    <p className="text-sm text-muted">{section.description}</p>
                                </div>
                                <div className="bg-background/50 rounded-3xl border border-border p-6 shadow-sm">
                                    {section.component}
                                </div>
                            </section>
                        ))}
                    </div>
				</ScrollShadow>
			</div>
		</div>
	);
}
