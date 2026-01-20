import { Button, Card, Label, ScrollShadow, Slider } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useGlobalOption, useEngineActions, globalOptionOptions } from "../hooks/useEngine";

export const Route = createFileRoute("/settings/browser")({
	component: BrowserSettingsPage,
	loader: async ({ context: { queryClient } }) => {
		queryClient.prefetchQuery(globalOptionOptions());
	},
});

function BrowserSettingsPage() {
	const navigate = useNavigate();
	const { data: options } = useGlobalOption();
	const { changeGlobalOption } = useEngineActions();

	const [cacheTTL, setCacheTTL] = useState(5);

	useEffect(() => {
		if (options) {
			const ttl = parseInt(options.fileBrowserCacheTTL || "5");
			setCacheTTL(isNaN(ttl) ? 5 : ttl);
		}
	}, [options]);

	const handleTTLChange = (val: number) => {
		setCacheTTL(val);
		changeGlobalOption.mutate({ fileBrowserCacheTTL: String(val) });
	};

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center gap-4 px-2 shrink-0">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/settings" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<div>
					<h2 className="text-2xl font-bold tracking-tight">Browser</h2>
					<p className="text-xs text-muted">File browsing performance & display</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Performance */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Performance</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-4">
									<div className="flex items-center justify-between">
										<div>
											<Label className="text-sm font-bold">Cache Duration</Label>
											<p className="text-xs text-muted mt-0.5">
												How long to cache file listings to improve performance
											</p>
										</div>
										<span className="text-sm font-bold text-accent bg-accent/10 px-3 py-1 rounded-lg">
											{cacheTTL} min
										</span>
									</div>
									<Slider
										value={cacheTTL}
										onChange={(val) => handleTTLChange(val as number)}
										minValue={1}
										maxValue={60}
										step={1}
									>
										<Slider.Track className="h-2 bg-default/10">
											<Slider.Fill className="bg-accent" />
											<Slider.Thumb className="w-5 h-5 border-2 border-accent bg-background" />
										</Slider.Track>
									</Slider>
									<div className="flex justify-between text-xs text-muted">
										<span>1 min</span>
										<span>30 min</span>
										<span>60 min</span>
									</div>
								</div>
							</Card>
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}
