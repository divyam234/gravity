import { Button, Card, Input, Label, ScrollShadow, Select, Slider, Switch, ListBox, TextField } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconFolder from "~icons/gravity-ui/folder";
import IconCloudArrowUpIn from "~icons/gravity-ui/cloud-arrow-up-in";
import { useSettingsStore } from "../store/useSettingsStore";
import { cn } from "../lib/utils";

export const Route = createFileRoute("/settings/downloads")({
	component: DownloadsSettingsPage,
});

function DownloadsSettingsPage() {
	const navigate = useNavigate();
	const { serverSettings, updateServerSettings } = useSettingsStore();

	const [showAdvanced, setShowAdvanced] = useState(false);

    // Local state to keep track of toggle state even if value is 0
    const [dlLimitEnabled, setDlLimitEnabled] = useState(false);
    const [ulLimitEnabled, setUlLimitEnabled] = useState(false);

    const parseSpeed = (speed?: string) => {
        if (!speed || speed === "0") return { limited: false, value: 0 };
        const match = speed.match(/^(\d+)([KkMm]?)$/);
        if (match) {
            let val = parseInt(match[1]);
            if (match[2]?.toLowerCase() === 'k') val = Math.round(val / 1024);
            return { limited: true, value: val };
        }
        return { limited: false, value: 0 };
    };

    useEffect(() => {
        if (serverSettings) {
            const dl = parseSpeed(serverSettings.download.maxDownloadSpeed);
            const ul = parseSpeed(serverSettings.download.maxUploadSpeed);
            if (dl.limited) setDlLimitEnabled(true);
            if (ul.limited) setUlLimitEnabled(true);
        }
    }, [serverSettings]);

    if (!serverSettings) {
        return <div className="p-8">Loading settings...</div>;
    }

    const { download, upload } = serverSettings;
    const dlSpeed = parseSpeed(download.maxDownloadSpeed);
    const ulSpeed = parseSpeed(download.maxUploadSpeed);

	const handleDownloadDirChange = (val: string) => {
		updateServerSettings((prev) => ({
            ...prev,
            download: { ...prev.download, downloadDir: val }
        }));
	};

	const handleMaxConcurrentChange = (value: number) => {
		updateServerSettings((prev) => ({
            ...prev,
            download: { ...prev.download, maxConcurrentDownloads: value }
        }));
	};

	const handleDownloadLimitChange = (limited: boolean, value?: number) => {
		setDlLimitEnabled(limited);
        const val = value !== undefined ? value : (dlSpeed.value || 10);
        const speedStr = limited && val > 0 ? `${val}M` : "0";
        updateServerSettings((prev) => ({
            ...prev,
            download: { ...prev.download, maxDownloadSpeed: speedStr }
        }));
	};

	const handleUploadLimitChange = (limited: boolean, value?: number) => {
		setUlLimitEnabled(limited);
        const val = value !== undefined ? value : (ulSpeed.value || 10);
        const speedStr = limited && val > 0 ? `${val}M` : "0";
        updateServerSettings((prev) => ({
            ...prev,
            download: { ...prev.download, maxUploadSpeed: speedStr }
        }));
	};

    const handleAutoUploadChange = (enabled: boolean) => {
        updateServerSettings((prev) => ({
            ...prev,
            upload: { ...prev.upload, autoUpload: enabled }
        }));
    };

    const handleDefaultRemoteChange = (val: string) => {
        updateServerSettings((prev) => ({
            ...prev,
            upload: { ...prev.upload, defaultRemote: val }
        }));
    };

    const handleRemoveLocalChange = (enabled: boolean) => {
        updateServerSettings((prev) => ({
            ...prev,
            upload: { ...prev.upload, removeLocal: enabled }
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
							<Card className="bg-background/50 border-border overflow-hidden">
								<Card.Content className="p-6 space-y-6">
									<TextField className="space-y-3">
										<Label className="text-sm font-bold">Download Folder</Label>
										<div className="relative group">
											<IconFolder className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted z-10 group-focus-within:text-accent transition-colors" />
											<Input
												value={download.downloadDir}
												onChange={(e) => handleDownloadDirChange(e.target.value)}
												placeholder="/downloads"
												className="pl-11 h-12 bg-default/10 rounded-2xl border-none"
												fullWidth
											/>
										</div>
										<p className="text-[10px] text-muted uppercase font-black tracking-widest">
											Where files are saved on the server
										</p>
									</TextField>

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
												isSelected={upload.autoUpload}
												onChange={handleAutoUploadChange}
											/>
										</div>

										{upload.autoUpload && (
											<div className="pl-0 space-y-3 animate-in slide-in-from-top-2 duration-200">
												<TextField className="relative group">
													<IconCloudArrowUpIn className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted z-10 group-focus-within:text-accent transition-colors" />
													<Input
														value={upload.defaultRemote}
														onChange={(e) => handleDefaultRemoteChange(e.target.value)}
														placeholder="gdrive:/downloads"
														className="pl-11 h-12 bg-default/10 rounded-2xl border-none"
														fullWidth
													/>
												</TextField>
												<p className="text-[10px] text-muted uppercase font-black tracking-widest">
													Use "remote:" or "remote:/path" syntax
												</p>

												<div className="flex items-center justify-between pt-2">
													<div>
														<Label className="text-sm font-medium">Delete local copy after upload</Label>
													</div>
													<Switch
														isSelected={upload.removeLocal}
														onChange={handleRemoveLocalChange}
														size="sm"
													/>
												</div>
											</div>
										)}
									</div>
								</Card.Content>
							</Card>
						</section>

						{/* Speed Limits */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Speed Limits</h3>
							</div>
							<Card className="bg-background/50 border-border overflow-hidden">
								<Card.Content className="p-6 space-y-6">
									<div className="space-y-4">
										<div className="flex items-center justify-between">
											<Label className="text-sm font-bold">Limit Download Speed</Label>
											<Switch
												isSelected={dlLimitEnabled}
												onChange={(selected) => handleDownloadLimitChange(selected)}
											/>
										</div>
										{dlLimitEnabled && (
											<div className="space-y-3 animate-in slide-in-from-top-2 duration-200">
												<div className="flex items-center justify-between">
													<span className="text-xs text-muted">0 MB/s</span>
													<span className="text-sm font-bold text-accent">{dlSpeed.value} MB/s</span>
													<span className="text-xs text-muted">100 MB/s</span>
												</div>
												<Slider
													value={dlSpeed.value || 10}
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
												isSelected={ulLimitEnabled}
												onChange={(selected) => handleUploadLimitChange(selected)}
											/>
										</div>
										{ulLimitEnabled && (
											<div className="space-y-3 animate-in slide-in-from-top-2 duration-200">
												<div className="flex items-center justify-between">
													<span className="text-xs text-muted">0 MB/s</span>
													<span className="text-sm font-bold text-cyan-500">{ulSpeed.value} MB/s</span>
													<span className="text-xs text-muted">100 MB/s</span>
												</div>
												<Slider
													value={ulSpeed.value || 10}
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
								</Card.Content>
							</Card>
						</section>

						{/* Queue Management */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Queue Management</h3>
							</div>
							<Card className="bg-background/50 border-border overflow-hidden">
								<Card.Content className="p-6 space-y-6">
									<div className="space-y-4">
										<div className="flex items-center justify-between">
											<div>
												<Label className="text-sm font-bold">Simultaneous Downloads</Label>
												<p className="text-xs text-muted mt-0.5">Maximum parallel downloads</p>
											</div>
											<Select
												selectedKey={String(download.maxConcurrentDownloads)}
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
								</Card.Content>
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
								<Card className="bg-background/50 border-border overflow-hidden animate-in slide-in-from-top-2 duration-200">
									<Card.Content className="p-6 space-y-6">
										<div className="space-y-4">
											<div className="flex items-center justify-between">
												<div>
													<Label className="text-sm font-bold">Connections per Download</Label>
													<p className="text-xs text-muted mt-0.5">Split downloads into multiple connections</p>
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
										</div>

										<p className="text-xs text-muted italic">
											More advanced options (file allocation, disk cache, etc.) are available in the server configuration.
										</p>
									</Card.Content>
								</Card>
							)}
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}
