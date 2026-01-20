import { Button, Card, Input, Label, ScrollShadow, Select, Slider, Switch, ListBox } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconFolder from "~icons/gravity-ui/folder";
import IconCloudArrowUpIn from "~icons/gravity-ui/cloud-arrow-up-in";
import { useServerSettings, useUpdateServerSettings } from "../hooks/useServerSettings";
import { useSettingsStore } from "../store/useSettingsStore";
import { useGlobalOption, useEngineActions, globalOptionOptions } from "../hooks/useEngine";
import { cn } from "../lib/utils";

export const Route = createFileRoute("/settings/downloads")({
	component: DownloadsSettingsPage,
	loader: async ({ context: { queryClient } }) => {
		queryClient.prefetchQuery(globalOptionOptions());
	},
});

function DownloadsSettingsPage() {
	const navigate = useNavigate();
	const { defaultRemote, setDefaultRemote } = useSettingsStore();
	const { data: serverSettings } = useServerSettings();
	const updateSettings = useUpdateServerSettings();
	const { data: options } = useGlobalOption();
	const { changeGlobalOption } = useEngineActions();

	const [showAdvanced, setShowAdvanced] = useState(false);
	const [downloadDir, setDownloadDir] = useState("");
	const [autoUpload, setAutoUpload] = useState(!!defaultRemote);
	const [deleteAfterUpload, setDeleteAfterUpload] = useState(false);

	// Speed limits
	const [downloadLimited, setDownloadLimited] = useState(false);
	const [downloadLimit, setDownloadLimit] = useState(0); // MB/s
	const [uploadLimited, setUploadLimited] = useState(false);
	const [uploadLimit, setUploadLimit] = useState(0);

	// Queue
	const [maxConcurrent, setMaxConcurrent] = useState(3);
	const [autoStart, setAutoStart] = useState(true);

	// Automation
	const [retryFailed, setRetryFailed] = useState(true);
	const [retryAttempts, setRetryAttempts] = useState(5);
	const [resumeIncomplete, setResumeIncomplete] = useState(true);

	// Advanced
	const [connectionsPerDownload, setConnectionsPerDownload] = useState(16);

	// Sync from server settings
	useEffect(() => {
		if (serverSettings?.download_dir) {
			setDownloadDir(serverSettings.download_dir);
		}
	}, [serverSettings]);

	// Sync from engine options
	useEffect(() => {
		if (options) {
			const maxConc = parseInt(options.maxConcurrentDownloads || "3");
			setMaxConcurrent(isNaN(maxConc) ? 3 : maxConc);

			const dlLimit = options.globalDownloadSpeedLimit || "0";
			if (dlLimit !== "0") {
				setDownloadLimited(true);
				// Parse limit (e.g., "50M" -> 50)
				const match = dlLimit.match(/^(\d+)([KkMm]?)$/);
				if (match) {
					let val = parseInt(match[1]);
					if (match[2]?.toLowerCase() === 'k') val = val / 1024;
					setDownloadLimit(val);
				}
			}

			const ulLimit = options.globalUploadSpeedLimit || "0";
			if (ulLimit !== "0") {
				setUploadLimited(true);
				const match = ulLimit.match(/^(\d+)([KkMm]?)$/);
				if (match) {
					let val = parseInt(match[1]);
					if (match[2]?.toLowerCase() === 'k') val = val / 1024;
					setUploadLimit(val);
				}
			}

			const maxRetries = parseInt(options.maxRetries || "5");
			setRetryAttempts(isNaN(maxRetries) ? 5 : maxRetries);
			setRetryFailed(maxRetries > 0);

			setResumeIncomplete(options.continueDownloads === "true");

			const conns = parseInt(options.maxConnectionsPerServer || "16");
			setConnectionsPerDownload(isNaN(conns) ? 16 : conns);
		}
	}, [options]);

	const handleDownloadDirSave = () => {
		if (downloadDir && downloadDir !== serverSettings?.download_dir) {
			updateSettings.mutate({ download_dir: downloadDir });
		}
	};

	const handleMaxConcurrentChange = (value: number) => {
		setMaxConcurrent(value);
		changeGlobalOption.mutate({ maxConcurrentDownloads: String(value) });
	};

	const handleDownloadLimitChange = (limited: boolean, value?: number) => {
		setDownloadLimited(limited);
		if (value !== undefined) setDownloadLimit(value);
		const limitStr = limited && (value || downloadLimit) > 0 ? `${value || downloadLimit}M` : "0";
		changeGlobalOption.mutate({ globalDownloadSpeedLimit: limitStr });
	};

	const handleUploadLimitChange = (limited: boolean, value?: number) => {
		setUploadLimited(limited);
		if (value !== undefined) setUploadLimit(value);
		const limitStr = limited && (value || uploadLimit) > 0 ? `${value || uploadLimit}M` : "0";
		changeGlobalOption.mutate({ globalUploadSpeedLimit: limitStr });
	};

	const handleRetryChange = (enabled: boolean, attempts?: number) => {
		setRetryFailed(enabled);
		if (attempts !== undefined) setRetryAttempts(attempts);
		changeGlobalOption.mutate({ maxRetries: enabled ? String(attempts || retryAttempts) : "0" });
	};

	const handleResumeChange = (enabled: boolean) => {
		setResumeIncomplete(enabled);
		changeGlobalOption.mutate({ continueDownloads: enabled ? "true" : "false" });
	};

	const handleConnectionsChange = (value: number) => {
		setConnectionsPerDownload(value);
		changeGlobalOption.mutate({ maxConnectionsPerServer: String(value) });
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
					<h2 className="text-2xl font-bold tracking-tight">Downloads</h2>
					<p className="text-xs text-muted">Speed, queue, storage & automation</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Storage */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Storage</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-3">
									<Label className="text-sm font-bold">Download Folder</Label>
									<div className="relative">
										<IconFolder className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted" />
										<Input
											value={downloadDir}
											onChange={(e) => setDownloadDir(e.target.value)}
											onBlur={handleDownloadDirSave}
											onKeyDown={(e) => e.key === "Enter" && handleDownloadDirSave()}
											placeholder="/downloads"
											className="pl-11 h-12 bg-default/10 rounded-2xl border-none"
											fullWidth
										/>
									</div>
									<p className="text-[10px] text-muted uppercase font-black tracking-widest">
										Where files are saved on the server
									</p>
								</div>

								<div className="h-px bg-border" />

								<div className="space-y-4">
									<div className="flex items-center justify-between">
										<div>
											<Label className="text-sm font-bold">Auto-Upload to Cloud</Label>
											<p className="text-xs text-muted mt-0.5">
												Automatically upload completed downloads to cloud storage
											</p>
										</div>
										<Switch
											isSelected={autoUpload}
											onChange={(selected) => {
												setAutoUpload(selected);
												if (!selected) setDefaultRemote("");
											}}
										>
											<Switch.Control>
												<Switch.Thumb />
											</Switch.Control>
										</Switch>
									</div>

									{autoUpload && (
										<div className="pl-0 space-y-3 animate-in slide-in-from-top-2 duration-200">
											<div className="relative">
												<IconCloudArrowUpIn className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted" />
												<Input
													value={defaultRemote}
													onChange={(e) => setDefaultRemote(e.target.value)}
													placeholder="gdrive:/downloads"
													className="pl-11 h-12 bg-default/10 rounded-2xl border-none"
													fullWidth
												/>
											</div>
											<p className="text-[10px] text-muted uppercase font-black tracking-widest">
												Use "remote:" or "remote:/path" syntax
											</p>

											<div className="flex items-center justify-between pt-2">
												<div>
													<Label className="text-sm font-medium">Delete local copy after upload</Label>
												</div>
												<Switch
													isSelected={deleteAfterUpload}
													onChange={setDeleteAfterUpload}
													size="sm"
												>
													<Switch.Control>
														<Switch.Thumb />
													</Switch.Control>
												</Switch>
											</div>
										</div>
									)}
								</div>
							</Card>
						</section>

						{/* Speed Limits */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Speed Limits</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-4">
									<div className="flex items-center justify-between">
										<Label className="text-sm font-bold">Limit Download Speed</Label>
										<Switch
											isSelected={downloadLimited}
											onChange={(selected) => handleDownloadLimitChange(selected)}
										>
											<Switch.Control>
												<Switch.Thumb />
											</Switch.Control>
										</Switch>
									</div>
									{downloadLimited && (
										<div className="space-y-3 animate-in slide-in-from-top-2 duration-200">
											<div className="flex items-center justify-between">
												<span className="text-xs text-muted">0 MB/s</span>
												<span className="text-sm font-bold text-accent">{downloadLimit} MB/s</span>
												<span className="text-xs text-muted">100 MB/s</span>
											</div>
											<Slider
												value={downloadLimit}
												onChange={(val) => handleDownloadLimitChange(true, val as number)}
												minValue={1}
												maxValue={100}
												step={1}
											>
												<Slider.Track className="h-2 bg-default/10">
													<Slider.Fill className="bg-accent" />
													<Slider.Thumb className="w-5 h-5 border-2 border-accent bg-background" />
												</Slider.Track>
											</Slider>
										</div>
									)}
								</div>

								<div className="h-px bg-border" />

								<div className="space-y-4">
									<div className="flex items-center justify-between">
										<Label className="text-sm font-bold">Limit Upload Speed</Label>
										<Switch
											isSelected={uploadLimited}
											onChange={(selected) => handleUploadLimitChange(selected)}
										>
											<Switch.Control>
												<Switch.Thumb />
											</Switch.Control>
										</Switch>
									</div>
									{uploadLimited && (
										<div className="space-y-3 animate-in slide-in-from-top-2 duration-200">
											<div className="flex items-center justify-between">
												<span className="text-xs text-muted">0 MB/s</span>
												<span className="text-sm font-bold text-cyan-500">{uploadLimit} MB/s</span>
												<span className="text-xs text-muted">100 MB/s</span>
											</div>
											<Slider
												value={uploadLimit}
												onChange={(val) => handleUploadLimitChange(true, val as number)}
												minValue={1}
												maxValue={100}
												step={1}
											>
												<Slider.Track className="h-2 bg-default/10">
													<Slider.Fill className="bg-cyan-500" />
													<Slider.Thumb className="w-5 h-5 border-2 border-cyan-500 bg-background" />
												</Slider.Track>
											</Slider>
										</div>
									)}
								</div>
							</Card>
						</section>

						{/* Queue Management */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Queue Management</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="space-y-4">
									<div className="flex items-center justify-between">
										<div>
											<Label className="text-sm font-bold">Simultaneous Downloads</Label>
											<p className="text-xs text-muted mt-0.5">Maximum parallel downloads</p>
										</div>
										<Select
											selectedKey={String(maxConcurrent)}
											onSelectionChange={(key) => handleMaxConcurrentChange(Number(key))}
											className="w-24"
										>
											<Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl border-none">
												<Select.Value className="text-sm font-bold" />
												<Select.Indicator className="text-muted">
													<IconChevronDown className="w-4 h-4" />
												</Select.Indicator>
											</Select.Trigger>
											<Select.Popover className="min-w-[100px] p-2 bg-background border border-border rounded-2xl shadow-xl">
												<ListBox items={[1, 2, 3, 4, 5, 6, 8, 10].map(n => ({ id: String(n), name: String(n) }))}>
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
								</div>

								<div className="h-px bg-border" />

								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Auto-start Downloads</Label>
										<p className="text-xs text-muted mt-0.5">Start downloads immediately when added</p>
									</div>
									<Switch
										isSelected={autoStart}
										onChange={setAutoStart}
									>
										<Switch.Control>
											<Switch.Thumb />
										</Switch.Control>
									</Switch>
								</div>
							</Card>
						</section>

						{/* Automation */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Automation</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-6">
								<div className="flex items-center justify-between">
									<div className="flex-1">
										<Label className="text-sm font-bold">Retry Failed Downloads</Label>
										<p className="text-xs text-muted mt-0.5">Automatically retry when downloads fail</p>
									</div>
									<div className="flex items-center gap-3">
										{retryFailed && (
											<Select
												selectedKey={String(retryAttempts)}
												onSelectionChange={(key) => handleRetryChange(true, Number(key))}
												className="w-20"
											>
												<Select.Trigger className="h-9 px-3 bg-default/10 rounded-xl border-none">
													<Select.Value className="text-sm font-bold" />
													<Select.Indicator className="text-muted">
														<IconChevronDown className="w-3 h-3" />
													</Select.Indicator>
												</Select.Trigger>
												<Select.Popover className="min-w-[80px] p-2 bg-background border border-border rounded-2xl shadow-xl">
													<ListBox items={[1, 2, 3, 5, 10].map(n => ({ id: String(n), name: String(n) }))}>
														{(item) => (
															<ListBox.Item
																id={item.id}
																textValue={item.name}
																className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 text-sm cursor-pointer outline-none"
															>
																<Label>{item.name}x</Label>
															</ListBox.Item>
														)}
													</ListBox>
												</Select.Popover>
											</Select>
										)}
										<Switch
											isSelected={retryFailed}
											onChange={(selected) => handleRetryChange(selected)}
										>
											<Switch.Control>
												<Switch.Thumb />
											</Switch.Control>
										</Switch>
									</div>
								</div>

								<div className="h-px bg-border" />

								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Resume Incomplete Downloads</Label>
										<p className="text-xs text-muted mt-0.5">Continue partially downloaded files</p>
									</div>
									<Switch
										isSelected={resumeIncomplete}
										onChange={handleResumeChange}
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
							<button
								type="button"
								onClick={() => setShowAdvanced(!showAdvanced)}
								className="flex items-center gap-3 mb-6 w-full text-left group"
							>
								<div className="w-1.5 h-6 bg-muted rounded-full group-hover:bg-accent transition-colors" />
								<h3 className="text-lg font-bold text-muted group-hover:text-foreground transition-colors">
									Advanced Engine Options
								</h3>
								<IconChevronDown className={cn(
									"w-4 h-4 text-muted transition-transform",
									showAdvanced && "rotate-180"
								)} />
							</button>
							
							{showAdvanced && (
								<Card className="p-6 bg-background/50 border-border space-y-6 animate-in slide-in-from-top-2 duration-200">
									<div className="space-y-4">
										<div className="flex items-center justify-between">
											<div>
												<Label className="text-sm font-bold">Connections per Download</Label>
												<p className="text-xs text-muted mt-0.5">Split downloads into multiple connections</p>
											</div>
											<Select
												selectedKey={String(connectionsPerDownload)}
												onSelectionChange={(key) => handleConnectionsChange(Number(key))}
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
									</div>

									<p className="text-xs text-muted italic">
										More advanced options (file allocation, disk cache, etc.) are available in the server configuration.
									</p>
								</Card>
							)}
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}
