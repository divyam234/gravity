import { Alert, Button, Chip, Label, Popover, Slider } from "@heroui/react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import React from "react";
import { DropZone, type FileDropItem } from "react-aria-components";
import IconBars from "~icons/gravity-ui/bars";
import IconCloud from "~icons/gravity-ui/cloud";
import IconCloudSlash from "~icons/gravity-ui/cloud-slash";
import IconDisplayPulse from "~icons/gravity-ui/display-pulse";
import IconFileArrowUp from "~icons/gravity-ui/file-arrow-up";
import IconMagicWand from "~icons/gravity-ui/magic-wand";
import IconMoon from "~icons/gravity-ui/moon";
import IconSun from "~icons/gravity-ui/sun";
import IconTriangleExclamation from "~icons/gravity-ui/triangle-exclamation";
import { globalStatOptions, useAria2Actions } from "../hooks/useAria2";
import { useShortcuts } from "../hooks/useShortcuts";
import { aria2 } from "../lib/aria2-rpc";
import { useFileStore } from "../store/useFileStore";
import { useSettingsStore } from "../store/useSettingsStore";
import { MobileSidebar, Sidebar } from "./Sidebar";

export const Layout: React.FC<{ children: React.ReactNode }> = ({
	children,
}) => {
	useShortcuts();
	const navigate = useNavigate();
	const { theme, setTheme, rpcUrl, pollingInterval } = useSettingsStore();
	const { setPendingFile } = useFileStore();
	const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false);

	// Safe query that only runs when configured
	const { isError, isLoading } = useQuery({
		...globalStatOptions(rpcUrl, pollingInterval),
		enabled: !!rpcUrl,
	});

	const { purgeDownloadResult } = useAria2Actions();

	const isDark =
		theme === "dark" ||
		(theme === "system" &&
			window.matchMedia("(prefers-color-scheme: dark)").matches);

	React.useEffect(() => {
		if (isDark) {
			document.documentElement.classList.add("dark");
		} else {
			document.documentElement.classList.remove("dark");
		}
	}, [isDark]);

	const toggleTheme = () => {
		setTheme(isDark ? "light" : "dark");
	};

	const [globalOptions, setGlobalOptions] = React.useState<
		Record<string, string>
	>({});

	React.useEffect(() => {
		if (rpcUrl) {
			aria2
				.getGlobalOption()
				.then(setGlobalOptions)
				.catch(() => {});
		}
	}, [rpcUrl]);

	const handleLimitChange = (key: string, value: number) => {
		const limit = value === 0 ? "0" : `${value}K`;
		aria2.changeGlobalOption({ [key]: limit });
		setGlobalOptions((prev) => ({ ...prev, [key]: limit }));
	};

	const handleDrop = async (e: any) => {
		const item = e.items.find((i: any) => i.kind === "file") as FileDropItem;
		if (item) {
			const file = await item.getFile();
			setPendingFile(file);
			navigate({ to: "/add" });
		}
	};

	return (
		<DropZone
			onDrop={handleDrop}
			className="min-h-screen h-screen bg-background text-foreground flex overflow-hidden outline-none"
		>
			{({ isDropTarget }) => (
				<>
					{isDropTarget && (
						<div className="fixed inset-0 z-100 bg-accent/10 backdrop-blur-sm flex flex-col items-center justify-center border-4 border-dashed border-accent m-4 rounded-3xl pointer-events-none animate-in fade-in duration-200">
							<IconFileArrowUp className="w-20 h-20 text-accent mb-4 animate-bounce" />
							<p className="text-2xl font-bold text-accent">
								Drop to Add Download
							</p>
						</div>
					)}

					{rpcUrl && (
						<>
							<Sidebar />
							<MobileSidebar
								isOpen={isMobileMenuOpen}
								onClose={() => setIsMobileMenuOpen(false)}
							/>
						</>
					)}

					<div className="flex-1 flex flex-col min-w-0 h-full overflow-hidden">
						{/* Header */}
						<header className="h-16 border-b border-default-100 flex items-center justify-between px-4 md:px-8 bg-background shrink-0">
							<div className="flex items-center gap-3">
								{rpcUrl && (
									<Button
										isIconOnly
										variant="ghost"
										className="md:hidden -ml-2"
										onPress={() => setIsMobileMenuOpen(true)}
									>
										<IconBars className="w-5 h-5" />
									</Button>
								)}
								{!isLoading && (
									<Chip
										size="sm"
										variant="soft"
										color={isError ? "danger" : "success"}
										className="h-6"
									>
										<div className="flex items-center gap-1.5">
											{isError ? (
												<IconCloudSlash className="w-3 h-3" />
											) : (
												<IconCloud className="w-3 h-3" />
											)}
											<span className="text-[10px] uppercase font-black hidden sm:inline">
												{isError ? "Offline" : "Connected"}
											</span>
										</div>
									</Chip>
								)}
							</div>

							<div className="flex items-center gap-2 md:gap-3">
								<Popover>
									<Popover.Trigger>
										<Button isIconOnly variant="ghost" aria-label="Limits">
											<IconDisplayPulse className="w-5 h-5" />
										</Button>
									</Popover.Trigger>
									<Popover.Content className="w-80">
										<Popover.Dialog className="p-4 space-y-6">
											<div className="space-y-4">
												<div className="flex justify-between items-center">
													<Label className="text-sm font-bold">
														Download Limit
													</Label>
													<span className="text-xs bg-default-100 px-2 py-0.5 rounded font-mono">
														{globalOptions["max-overall-download-limit"] || "0"}
													</span>
												</div>
												<Slider
													minValue={0}
													maxValue={10240}
													step={128}
													value={parseInt(
														globalOptions["max-overall-download-limit"] || "0",
													)}
													onChange={(val) =>
														handleLimitChange(
															"max-overall-download-limit",
															val as number,
														)
													}
												>
													<Slider.Track>
														<Slider.Fill />
														<Slider.Thumb />
													</Slider.Track>
												</Slider>
											</div>
											<div className="space-y-4">
												<div className="flex justify-between items-center">
													<Label className="text-sm font-bold">
														Upload Limit
													</Label>
													<span className="text-xs bg-default-100 px-2 py-0.5 rounded font-mono">
														{globalOptions["max-overall-upload-limit"] || "0"}
													</span>
												</div>
												<Slider
													minValue={0}
													maxValue={10240}
													step={128}
													value={parseInt(
														globalOptions["max-overall-upload-limit"] || "0",
													)}
													onChange={(val) =>
														handleLimitChange(
															"max-overall-upload-limit",
															val as number,
														)
													}
												>
													<Slider.Track>
														<Slider.Fill />
														<Slider.Thumb />
													</Slider.Track>
												</Slider>
											</div>
										</Popover.Dialog>
									</Popover.Content>
								</Popover>

								<Button
									isIconOnly
									variant="ghost"
									onPress={() => purgeDownloadResult.mutate()}
									className="text-default-500 hover:text-danger"
									aria-label="Purge"
								>
									<IconMagicWand className="w-5 h-5" />
								</Button>

								<div className="w-px h-6 bg-default-100 mx-1" />

								<Button
									isIconOnly
									variant="ghost"
									onPress={toggleTheme}
									aria-label="Theme"
								>
									{isDark ? (
										<IconSun className="w-5 h-5" />
									) : (
										<IconMoon className="w-5 h-5" />
									)}
								</Button>
							</div>
						</header>

						<main className="flex-1 overflow-y-auto p-4 md:p-8 relative">
							{isError && (
								<div className="mb-6">
									<Alert
										status="danger"
										className="rounded-2xl border-danger-soft-hover shadow-lg shadow-danger/5 animate-in slide-in-from-top duration-300"
									>
										<Alert.Indicator>
											<IconTriangleExclamation className="w-5 h-5" />
										</Alert.Indicator>
										<Alert.Content>
											<Alert.Title className="font-bold text-base">
												RPC Connection Failed
											</Alert.Title>
											<Alert.Description className="text-sm opacity-90">
												Unable to connect to aria2 at the specified URL. Please
												check your settings.
											</Alert.Description>
										</Alert.Content>
										<Button
											size="sm"
											variant="secondary"
											className="ml-auto font-bold"
											onPress={() =>
												navigate({
													to: "/settings/$category",
													params: { category: "connection" },
												})
											}
										>
											Fix
										</Button>
									</Alert>
								</div>
							)}
							{children}
						</main>
					</div>
				</>
			)}
		</DropZone>
	);
};
