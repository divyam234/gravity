import { Button, Card, Input, Label, ListBox, ScrollShadow, Select, Slider, Switch } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/settings/torrents")({
	component: TorrentsSettingsPage,
});

function TorrentsSettingsPage() {
	const navigate = useNavigate();
    const { serverSettings, updateServerSettings } = useSettingsStore();

    if (!serverSettings) {
        return <div className="p-8">Loading settings...</div>;
    }

    const { torrent } = serverSettings;

    const seedRatio = parseFloat(torrent.seedRatio);
    const seedEnabled = seedRatio > 0;

	const handleSeedRatioChange = (enabled: boolean, ratio?: number) => {
		const finalRatio = enabled ? (ratio ?? (seedRatio || 1.0)) : 0;
        updateServerSettings(prev => ({
            ...prev,
            torrent: { ...prev.torrent, seedRatio: String(finalRatio) }
        }));
	};

	const handleSeedTimeChange = (hours: number) => {
		updateServerSettings(prev => ({
            ...prev,
            torrent: { ...prev.torrent, seedTime: hours }
        }));
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
					<h2 className="text-2xl font-bold tracking-tight">Torrents</h2>
					<p className="text-xs text-muted">BitTorrent seeding, privacy & performance</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Info Banner */}
						<Card className="p-5 bg-warning/5 border-warning/20">
							<p className="text-sm text-muted">
								<span className="font-bold text-warning">ℹ️ Note:</span> These settings apply when downloading via BitTorrent (P2P)
								instead of through a premium debrid service.
							</p>
						</Card>

						{/* Seeding */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Seeding</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Seed After Download</Label>
										<p className="text-xs text-muted mt-0.5">Share downloaded files with other peers</p>
									</div>
									<Switch
										isSelected={seedEnabled}
										onChange={(selected) => handleSeedRatioChange(selected)}
									>
										<Switch.Control>
											<Switch.Thumb />
										</Switch.Control>
									</Switch>
								</div>

								{seedEnabled && (
									<>
										<div className="h-px bg-border" />

										<div className="space-y-4 animate-in slide-in-from-top-2 duration-200">
											<div className="flex items-center justify-between">
												<Label className="text-sm font-bold">Stop Seeding at Ratio</Label>
												<span className="text-sm font-bold text-accent bg-accent/10 px-3 py-1 rounded-lg">
													{seedRatio.toFixed(1)}
												</span>
											</div>
											<Slider
												value={seedRatio}
												onChange={(val) => handleSeedRatioChange(true, val as number)}
												minValue={0.1}
												maxValue={5.0}
												step={0.1}
											>
												<Slider.Track className="h-2 bg-default/10">
													<Slider.Fill className="bg-accent" />
													<Slider.Thumb className="w-5 h-5 border-2 border-accent bg-background" />
												</Slider.Track>
											</Slider>
											<div className="flex justify-between text-xs text-muted">
												<span>0.1 (minimal)</span>
												<span>1.0 (fair share)</span>
												<span>5.0 (generous)</span>
											</div>
										</div>

										<div className="h-px bg-border" />

										<div className="flex items-center justify-between">
											<div>
												<Label className="text-sm font-bold">Maximum Seed Time</Label>
												<p className="text-xs text-muted mt-0.5">Stop seeding after this duration</p>
											</div>
											<Select
												selectedKey={String(torrent.seedTime)}
												onSelectionChange={(key) => handleSeedTimeChange(Number(key))}
												className="w-32"
											>
												<Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl border-none">
													<Select.Value className="text-sm font-bold" />
													<Select.Indicator className="text-muted">
														<IconChevronDown className="w-4 h-4" />
													</Select.Indicator>
												</Select.Trigger>
												<Select.Popover className="min-w-[140px] p-2 bg-background border border-border rounded-2xl shadow-xl">
													<ListBox items={[
														{ id: "1", name: "1 hour" },
														{ id: "6", name: "6 hours" },
														{ id: "12", name: "12 hours" },
														{ id: "24", name: "24 hours" },
														{ id: "48", name: "48 hours" },
														{ id: "168", name: "1 week" },
													]}>
														{(item) => (
															<ListBox.Item
																id={item.id}
																textValue={item.name}
																className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 text-sm cursor-pointer outline-none"
															>
																<Label>{item.name}</Label>
															</ListBox.Item>
														)}
													</ListBox>
												</Select.Popover>
											</Select>
										</div>
									</>
								)}
							</Card>
						</section>

						{/* Privacy & Encryption */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Privacy & Encryption</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-3">
									<Label className="text-sm font-bold">Protocol Encryption</Label>
									<div className="flex gap-2">
										{([
											{ id: "disabled", label: "Disabled" },
											{ id: "plain", label: "Prefer Encrypted" },
											{ id: "required", label: "Require Encrypted" },
										] as const).map((opt) => (
											<Button
												key={opt.id}
												size="sm"
												variant={torrent.encryption === opt.id ? "primary" : "secondary"}
												onPress={() => updateServerSettings(prev => ({ ...prev, torrent: { ...prev.torrent, encryption: opt.id } }))}
												className="rounded-xl font-bold"
											>
												{opt.label}
											</Button>
										))}
									</div>
									<p className="text-xs text-muted">
										Encrypted connections help prevent ISP throttling
									</p>
								</div>

								<div className="h-px bg-border" />

								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">DHT (Distributed Hash Table)</Label>
										<p className="text-xs text-muted mt-0.5">Find peers without relying on trackers</p>
									</div>
									<Switch 
                                        isSelected={torrent.enableDht} 
                                        onChange={(selected) => updateServerSettings(prev => ({ ...prev, torrent: { ...prev.torrent, enableDht: selected } }))}
                                    />
								</div>

								<div className="h-px bg-border" />

								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">PEX (Peer Exchange)</Label>
										<p className="text-xs text-muted mt-0.5">Share peer lists with connected peers</p>
									</div>
									<Switch 
                                        isSelected={torrent.enablePex} 
                                        onChange={(selected) => updateServerSettings(prev => ({ ...prev, torrent: { ...prev.torrent, enablePex: selected } }))}
                                    />
								</div>

								<div className="h-px bg-border" />

								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">LPD (Local Peer Discovery)</Label>
										<p className="text-xs text-muted mt-0.5">Find peers on your local network</p>
									</div>
									<Switch 
                                        isSelected={torrent.enableLpd} 
                                        onChange={(selected) => updateServerSettings(prev => ({ ...prev, torrent: { ...prev.torrent, enableLpd: selected } }))}
                                    />
								</div>
							</Card>
						</section>

						{/* Listening Port */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Listening Port</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-3">
									<Label className="text-sm font-bold">Listen Port</Label>
									<div className="flex items-center gap-3">
										<Input
											value={String(torrent.listenPort)}
											onChange={(e) => updateServerSettings(prev => ({ ...prev, torrent: { ...prev.torrent, listenPort: Number(e.target.value) } }))}
											placeholder="6881"
											className="w-32 h-11 bg-default/10 rounded-xl border-none"
										/>
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