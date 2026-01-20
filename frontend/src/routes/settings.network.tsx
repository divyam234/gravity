import { Button, Card, Input, Label, ListBox, ScrollShadow, Select, Switch } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import { useGlobalOption, useEngineActions, globalOptionOptions } from "../hooks/useEngine";

export const Route = createFileRoute("/settings/network")({
	component: NetworkSettingsPage,
	loader: async ({ context: { queryClient } }) => {
		queryClient.prefetchQuery(globalOptionOptions());
	},
});

function NetworkSettingsPage() {
	const navigate = useNavigate();
	const { data: options } = useGlobalOption();
	const { changeGlobalOption } = useEngineActions();

	// Proxy settings
	const [proxyType, setProxyType] = useState<"none" | "http" | "socks5">("none");
	const [proxyHost, setProxyHost] = useState("");
	const [proxyPort, setProxyPort] = useState("1080");
	const [proxyAuth, setProxyAuth] = useState(false);
	const [proxyUser, setProxyUser] = useState("");
	const [proxyPass, setProxyPass] = useState("");

	// Connection settings
	const [maxConnections, setMaxConnections] = useState(16);
	const [timeout, setTimeout] = useState(60);
	const [maxRetries, setMaxRetries] = useState(5);

	// Security
	const [verifyCerts, setVerifyCerts] = useState(true);

	// Advanced
	const [userAgent, setUserAgent] = useState("gravity/1.0");

	// Sync from engine options
	useEffect(() => {
		if (options) {
			// Parse proxy URL if set
			const proxyUrl = options.proxyUrl || "";
			if (proxyUrl) {
				if (proxyUrl.startsWith("socks5://")) {
					setProxyType("socks5");
					const match = proxyUrl.match(/socks5:\/\/([^:]+):(\d+)/);
					if (match) {
						setProxyHost(match[1]);
						setProxyPort(match[2]);
					}
				} else if (proxyUrl.startsWith("http://")) {
					setProxyType("http");
					const match = proxyUrl.match(/http:\/\/([^:]+):(\d+)/);
					if (match) {
						setProxyHost(match[1]);
						setProxyPort(match[2]);
					}
				}
			}

			if (options.proxyUser) {
				setProxyAuth(true);
				setProxyUser(options.proxyUser);
			}

			const conns = parseInt(options.maxConnectionsPerServer || "16");
			setMaxConnections(isNaN(conns) ? 16 : conns);

			const to = parseInt(options.connectTimeout || "60");
			setTimeout(isNaN(to) ? 60 : to);

			const retries = parseInt(options.maxRetries || "5");
			setMaxRetries(isNaN(retries) ? 5 : retries);

			setVerifyCerts(options.checkCertificate !== "false");

			if (options.userAgent) {
				setUserAgent(options.userAgent);
			}
		}
	}, [options]);

	const handleProxyChange = () => {
		if (proxyType === "none") {
			changeGlobalOption.mutate({ proxyUrl: "" });
		} else {
			const url = `${proxyType}://${proxyHost}:${proxyPort}`;
			changeGlobalOption.mutate({ proxyUrl: url });
		}
	};

	const handleProxyAuthChange = () => {
		if (proxyAuth && proxyUser) {
			changeGlobalOption.mutate({ proxyUser, proxyPassword: proxyPass });
		} else {
			changeGlobalOption.mutate({ proxyUser: "", proxyPassword: "" });
		}
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
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-3">
									<Label className="text-sm font-bold">Proxy Type</Label>
									<div className="flex gap-2">
										{(["none", "http", "socks5"] as const).map((type) => (
											<Button
												key={type}
												size="sm"
												variant={proxyType === type ? "primary" : "secondary"}
												onPress={() => {
													setProxyType(type);
													if (type === "none") {
														changeGlobalOption.mutate({ proxyUrl: "" });
													}
												}}
												className="rounded-xl font-bold capitalize"
											>
												{type === "none" ? "No Proxy" : type.toUpperCase()}
											</Button>
										))}
									</div>
								</div>

								{proxyType !== "none" && (
									<>
										<div className="grid grid-cols-3 gap-4 animate-in slide-in-from-top-2 duration-200">
											<div className="col-span-2">
												<Label className="text-sm font-bold mb-2 block">Host</Label>
												<Input
													value={proxyHost}
													onChange={(e) => setProxyHost(e.target.value)}
													onBlur={handleProxyChange}
													placeholder="proxy.example.com"
													className="h-11 bg-default/10 rounded-xl border-none"
												/>
											</div>
											<div>
												<Label className="text-sm font-bold mb-2 block">Port</Label>
												<Input
													value={proxyPort}
													onChange={(e) => setProxyPort(e.target.value)}
													onBlur={handleProxyChange}
													placeholder="1080"
													className="h-11 bg-default/10 rounded-xl border-none"
												/>
											</div>
										</div>

										<div className="h-px bg-border" />

										<div className="flex items-center justify-between">
											<div>
												<Label className="text-sm font-bold">Proxy Authentication</Label>
												<p className="text-xs text-muted mt-0.5">Proxy requires username and password</p>
											</div>
											<Switch isSelected={proxyAuth} onChange={setProxyAuth}>
												<Switch.Control>
													<Switch.Thumb />
												</Switch.Control>
											</Switch>
										</div>

										{proxyAuth && (
											<div className="grid grid-cols-2 gap-4 animate-in slide-in-from-top-2 duration-200">
												<div>
													<Label className="text-sm font-bold mb-2 block">Username</Label>
													<Input
														value={proxyUser}
														onChange={(e) => setProxyUser(e.target.value)}
														onBlur={handleProxyAuthChange}
														placeholder="username"
														className="h-11 bg-default/10 rounded-xl border-none"
													/>
												</div>
												<div>
													<Label className="text-sm font-bold mb-2 block">Password</Label>
													<Input
														type="password"
														value={proxyPass}
														onChange={(e) => setProxyPass(e.target.value)}
														onBlur={handleProxyAuthChange}
														placeholder="password"
														className="h-11 bg-default/10 rounded-xl border-none"
													/>
												</div>
											</div>
										)}
									</>
								)}
							</Card>
						</section>

						{/* Connection Limits */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Connection Limits</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Max Connections per Server</Label>
										<p className="text-xs text-muted mt-0.5">Parallel connections to each server</p>
									</div>
									<Select
										selectedKey={String(maxConnections)}
										onSelectionChange={(key) => {
											const val = Number(key);
											setMaxConnections(val);
											changeGlobalOption.mutate({ maxConnectionsPerServer: String(val) });
										}}
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
										selectedKey={String(timeout)}
										onSelectionChange={(key) => {
											const val = Number(key);
											setTimeout(val);
											changeGlobalOption.mutate({ connectTimeout: String(val) });
										}}
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
										selectedKey={String(maxRetries)}
										onSelectionChange={(key) => {
											const val = Number(key);
											setMaxRetries(val);
											changeGlobalOption.mutate({ maxRetries: String(val) });
										}}
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
							</Card>
						</section>

						{/* Security */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Security</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Verify SSL/TLS Certificates</Label>
										<p className="text-xs text-muted mt-0.5">Reject connections with invalid certificates</p>
									</div>
									<Switch
										isSelected={verifyCerts}
										onChange={(selected) => {
											setVerifyCerts(selected);
											changeGlobalOption.mutate({ checkCertificate: selected ? "true" : "false" });
										}}
									>
										<Switch.Control>
											<Switch.Thumb />
										</Switch.Control>
									</Switch>
								</div>
							</Card>
						</section>

						{/* Advanced */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Advanced</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-3">
									<Label className="text-sm font-bold">User-Agent</Label>
									<Input
										value={userAgent}
										onChange={(e) => setUserAgent(e.target.value)}
										onBlur={() => changeGlobalOption.mutate({ userAgent })}
										placeholder="gravity/1.0"
										className="h-11 bg-default/10 rounded-xl border-none"
									/>
									<p className="text-xs text-muted">
										The User-Agent header sent with HTTP requests
									</p>
								</div>
							</Card>
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}
