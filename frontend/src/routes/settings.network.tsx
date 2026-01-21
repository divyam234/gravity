import { Button, Card, Input, Label, ListBox, ScrollShadow, Select, Switch, TextField } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/settings/network")({
	component: NetworkSettingsPage,
});

function NetworkSettingsPage() {
	const navigate = useNavigate();
    const { serverSettings, updateServerSettings } = useSettingsStore();

    if (!serverSettings) {
        return <div className="p-8">Loading settings...</div>;
    }

    const { network, download } = serverSettings;

    const parseProxyUrl = (url: string) => {
        if (!url) return { type: "none" as const, host: "", port: "1080" };
        const match = url.match(/^(socks5|http):\/\/([^:]+):(\d+)/);
        if (match) {
            return { type: match[1] as "http" | "socks5", host: match[2], port: match[3] };
        }
        return { type: "none" as const, host: "", port: "1080" };
    };

    const proxyInfo = parseProxyUrl(network.proxyUrl);

	const handleProxyChange = (updates: Partial<{ type: "none" | "http" | "socks5", host: string, port: string }>) => {
		const type = updates.type || proxyInfo.type;
        const host = updates.host !== undefined ? updates.host : proxyInfo.host;
        const port = updates.port !== undefined ? updates.port : proxyInfo.port;

        let url = "";
        if (type !== "none" && host) {
            url = `${type}://${host}:${port}`;
        }

        updateServerSettings(prev => ({
            ...prev,
            network: { ...prev.network, proxyUrl: url, proxyEnabled: type !== "none" }
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
					<h2 className="text-2xl font-bold tracking-tight">Network</h2>
					<p className="text-xs text-muted">Proxy, connections & security</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Proxy */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Proxy</h3>
							</div>
							<Card className="bg-background/50 border-border overflow-hidden">
								<Card.Content className="p-6 space-y-6">
									<div className="space-y-3">
										<Label className="text-sm font-bold">Proxy Type</Label>
										<div className="flex gap-2">
											{(["none", "http", "socks5"] as const).map((type) => (
												<Button
													key={type}
													size="sm"
													variant={proxyInfo.type === type ? "primary" : "secondary"}
													onPress={() => handleProxyChange({ type })}
													className="rounded-xl font-bold capitalize"
												>
													{type === "none" ? "No Proxy" : type.toUpperCase()}
												</Button>
											))}
										</div>
									</div>

									{proxyInfo.type !== "none" && (
										<>
											<div className="grid grid-cols-3 gap-4 animate-in slide-in-from-top-2 duration-200">
												<TextField className="col-span-2">
													<Label className="text-sm font-bold mb-2 block">Host</Label>
													<Input
														value={proxyInfo.host}
														onChange={(e) => handleProxyChange({ host: e.target.value })}
														placeholder="proxy.example.com"
														className="h-11 bg-default/10 rounded-xl border-none"
													/>
												</TextField>
												<TextField>
													<Label className="text-sm font-bold mb-2 block">Port</Label>
													<Input
														value={proxyInfo.port}
														onChange={(e) => handleProxyChange({ port: e.target.value })}
														placeholder="1080"
														className="h-11 bg-default/10 rounded-xl border-none"
													/>
												</TextField>
											</div>

											<div className="h-px bg-border" />

											<div className="flex items-center justify-between">
												<div>
													<Label className="text-sm font-bold">Proxy Authentication</Label>
													<p className="text-xs text-muted mt-0.5">Proxy requires username and password</p>
												</div>
												<Switch 
                                                    isSelected={!!network.proxyUser} 
                                                    onChange={(selected) => {
                                                        if (!selected) {
                                                            updateServerSettings(prev => ({ 
                                                                ...prev, 
                                                                network: { ...prev.network, proxyUser: "", proxyPassword: "" } 
                                                            }));
                                                        }
                                                    }} 
                                                />
											</div>

											{(!!network.proxyUser || true) && (
												<div className="grid grid-cols-2 gap-4 animate-in slide-in-from-top-2 duration-200">
													<TextField>
														<Label className="text-sm font-bold mb-2 block">Username</Label>
														<Input
															value={network.proxyUser}
															onChange={(e) => updateServerSettings(prev => ({ ...prev, network: { ...prev.network, proxyUser: e.target.value } }))}
															placeholder="username"
															className="h-11 bg-default/10 rounded-xl border-none"
														/>
													</TextField>
													<TextField>
														<Label className="text-sm font-bold mb-2 block">Password</Label>
														<Input
															type="password"
															value={network.proxyPassword}
															onChange={(e) => updateServerSettings(prev => ({ ...prev, network: { ...prev.network, proxyPassword: e.target.value } }))}
															placeholder="password"
															className="h-11 bg-default/10 rounded-xl border-none"
														/>
													</TextField>
												</div>
											)}
										</>
									)}
								</Card.Content>
							</Card>
						</section>

						{/* Connection Limits */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Connection Limits</h3>
							</div>
							<Card className="bg-background/50 border-border overflow-hidden">
								<Card.Content className="p-6 space-y-6">
									<div className="flex items-center justify-between">
										<div>
											<Label className="text-sm font-bold">Max Connections per Server</Label>
											<p className="text-xs text-muted mt-0.5">Parallel connections to each server</p>
										</div>
										<Select
											selectedKey={String(download.maxConnectionPerServer)}
											onSelectionChange={(key) => updateServerSettings(prev => ({ ...prev, download: { ...prev.download, maxConnectionPerServer: Number(key) } }))}
											className="w-24"
										>
											<Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl border-none">
												<Select.Value className="text-sm font-bold" />
												<Select.Indicator className="text-muted">
													<IconChevronDown className="w-4 h-4" />
												</Select.Indicator>
											</Select.Trigger>
											<Select.Popover className="min-w-[100px] p-2 bg-background border border-border rounded-2xl shadow-xl">
												<ListBox items={[1, 2, 4, 8, 16].map(n => ({ id: String(n), name: String(n) }))}>
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

									<div className="h-px bg-border" />

									<div className="flex items-center justify-between">
										<div>
											<Label className="text-sm font-bold">Connection Timeout</Label>
											<p className="text-xs text-muted mt-0.5">Seconds to wait before giving up</p>
										</div>
										<Select
											selectedKey={String(download.connectTimeout)}
											onSelectionChange={(key) => updateServerSettings(prev => ({ ...prev, download: { ...prev.download, connectTimeout: Number(key) } }))}
											className="w-28"
										>
											<Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl border-none">
												<Select.Value className="text-sm font-bold" />
												<Select.Indicator className="text-muted">
													<IconChevronDown className="w-4 h-4" />
												</Select.Indicator>
											</Select.Trigger>
											<Select.Popover className="min-w-[120px] p-2 bg-background border border-border rounded-2xl shadow-xl">
												<ListBox items={[30, 60, 120, 300].map(n => ({ id: String(n), name: `${n}s` }))}>
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

									<div className="h-px bg-border" />

									<div className="flex items-center justify-between">
										<div>
											<Label className="text-sm font-bold">Max Retries</Label>
											<p className="text-xs text-muted mt-0.5">Number of retry attempts on failure</p>
										</div>
										<Select
											selectedKey={String(download.maxTries)}
											onSelectionChange={(key) => updateServerSettings(prev => ({ ...prev, download: { ...prev.download, maxTries: Number(key) } }))}
											className="w-24"
										>
											<Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl border-none">
												<Select.Value className="text-sm font-bold" />
												<Select.Indicator className="text-muted">
													<IconChevronDown className="w-4 h-4" />
												</Select.Indicator>
											</Select.Trigger>
											<Select.Popover className="min-w-[100px] p-2 bg-background border border-border rounded-2xl shadow-xl">
												<ListBox items={[0, 1, 3, 5, 10].map(n => ({ id: String(n), name: n === 0 ? "∞" : String(n) }))}>
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
								</Card.Content>
							</Card>
						</section>

						{/* Security */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Security</h3>
							</div>
							<Card className="bg-background/50 border-border overflow-hidden">
								<Card.Content className="p-6 space-y-6">
									<div className="flex items-center justify-between">
										<div>
											<Label className="text-sm font-bold">Verify SSL/TLS Certificates</Label>
											<p className="text-xs text-muted mt-0.5">Reject connections with invalid certificates</p>
										</div>
										<Switch
											isSelected={download.checkCertificate}
											onChange={(selected) => updateServerSettings(prev => ({ ...prev, download: { ...prev.download, checkCertificate: selected } }))}
										/>
									</div>
								</Card.Content>
							</Card>
						</section>

						{/* Advanced */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Advanced</h3>
							</div>
							<Card className="bg-background/50 border-border overflow-hidden">
								<Card.Content className="p-6 space-y-6">
									<TextField className="space-y-3">
										<Label className="text-sm font-bold">User-Agent</Label>
										<Input
											value={download.userAgent}
											onChange={(e) => updateServerSettings(prev => ({ ...prev, download: { ...prev.download, userAgent: e.target.value } }))}
											placeholder="gravity/1.0"
											className="h-11 bg-default/10 rounded-xl border-none"
										/>
										<p className="text-xs text-muted">
											The User-Agent header sent with HTTP requests
										</p>
									</TextField>
								</Card.Content>
							</Card>
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}