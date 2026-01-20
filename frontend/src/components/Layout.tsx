import {
	Button,
	Chip,
	Input,
	Kbd,
	Popover,
	Slider,
	Tooltip,
    Label,
    Select,
    ListBox,
    Header,
    Separator,
} from "@heroui/react";
import { useLocation, useNavigate } from "@tanstack/react-router";
import React, { useState } from "react";
import IconBars from "~icons/gravity-ui/bars";
import IconCloud from "~icons/gravity-ui/cloud";
import IconCloudSlash from "~icons/gravity-ui/cloud-slash";
import IconDisplayPulse from "~icons/gravity-ui/display-pulse";
import IconMagicWand from "~icons/gravity-ui/magic-wand";
import IconMagnifier from "~icons/gravity-ui/magnifier";
import IconMoon from "~icons/gravity-ui/moon";
import IconPlus from "~icons/gravity-ui/plus";
import IconPulse from "~icons/gravity-ui/pulse";
import IconSun from "~icons/gravity-ui/sun";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconServer from "~icons/gravity-ui/server";
import IconGear from "~icons/gravity-ui/gear";
import { useEngineActions, useGlobalStat } from "../hooks/useEngine";
import { useShortcuts } from "../hooks/useShortcuts";
import { useSettingsStore } from "../store/useSettingsStore";
import { MobileSidebar, Sidebar } from "./Sidebar";
import { cn } from "../lib/utils";

export const Layout: React.FC<{ children: React.ReactNode }> = ({
	children,
}) => {
	useShortcuts();
	const navigate = useNavigate();
	const location = useLocation();
    const manageServersId = "manage-servers";
	const {
		theme,
		setTheme,
		searchQuery,
		setSearchQuery,
        servers,
        activeServerId,
        setActiveServer,
	} = useSettingsStore();
	const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false);

	const isTaskListPage = location.pathname.startsWith("/tasks");

	// Safe query that only runs when configured
	const { isError, isLoading } = useGlobalStat();

	const { purgeDownloadResult } = useEngineActions();

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

	const [globalOptions, setGlobalOptions] = useState<
		Record<string, string>
	>({});

    // Limits not supported via API yet
	const handleLimitChange = (key: string, value: number) => {
		const limit = value === 0 ? "0" : `${value}K`;
		setGlobalOptions((prev) => ({ ...prev, [key]: limit }));
	};

	return (
		<div className="min-h-screen h-screen bg-background text-foreground flex overflow-hidden outline-none">
            <Sidebar />
            <MobileSidebar
                isOpen={isMobileMenuOpen}
                onClose={() => setIsMobileMenuOpen(false)}
            />

			<div className="flex-1 flex flex-col min-w-0 h-full overflow-hidden">
				{/* Header */}
				<header className="h-16 border-b border-border flex items-center justify-between px-4 md:px-8 bg-background shrink-0 gap-4">
					<div className="flex items-center gap-3 shrink-0">
                        <Button
                            isIconOnly
                            variant="ghost"
                            className="md:hidden -ml-2"
                            onPress={() => setIsMobileMenuOpen(true)}
                        >
                            <IconBars className="w-5 h-5" />
                        </Button>
                        <Chip
                            size="sm"
                            variant="soft"
                            color={isError ? "danger" : isLoading ? "warning" : "success"}
                            className="h-7 px-3"
                        >
                            <div className="flex items-center gap-2">
                                {isError ? (
                                    <IconCloudSlash className="w-3.5 h-3.5" />
                                ) : isLoading ? (
                                    <IconPulse className="w-3.5 h-3.5 animate-pulse" />
                                ) : (
                                    <IconCloud className="w-3.5 h-3.5" />
                                )}
                                <span className="text-[10px] uppercase font-black hidden sm:inline tracking-widest">
                                    {isError
                                        ? "Offline"
                                        : isLoading
                                            ? "Connecting"
                                            : "Connected"}
                                </span>
                            </div>
                        </Chip>
					</div>

					{/* Global Search Bar */}
					{isTaskListPage && (
						<div className="flex-1 max-w-md relative hidden md:block">
							<IconMagnifier className="absolute left-3 top-1/2 -translate-y-1/2 text-muted z-10 w-4 h-4" />
							<Input
								placeholder="Search downloads..."
								className="pl-10 h-10 bg-default/10 rounded-xl border-none focus:bg-default/20 transition-all outline-none"
								value={searchQuery}
								onChange={(e) => setSearchQuery(e.target.value)}
								fullWidth
							/>
						</div>
					)}

					<div className="flex items-center gap-2 md:gap-3 shrink-0 ml-auto">
						<Tooltip>
							<Tooltip.Trigger>
								<Button
									isIconOnly
									variant="primary"
									onPress={() => navigate({ to: "/add" })}
									className="h-9 w-9 min-w-9 rounded-xl shadow-lg shadow-primary/20"
									aria-label="Add Download"
								>
									<IconPlus className="w-5 h-5" />
								</Button>
							</Tooltip.Trigger>
							<Tooltip.Content className="p-2 text-xs flex items-center gap-2">
								Add Download <Kbd>Shift+A</Kbd>
							</Tooltip.Content>
						</Tooltip>

						<Tooltip>
							<Tooltip.Trigger>
								<Button
									isIconOnly
									variant="ghost"
									onPress={() => purgeDownloadResult.mutate()}
									className="text-muted hover:text-danger h-9 w-9 min-w-9"
									aria-label="Purge"
								>
									<IconMagicWand className="w-5 h-5" />
								</Button>
							</Tooltip.Trigger>
							<Tooltip.Content className="p-2 text-xs flex items-center gap-2">
								Purge Finished <Kbd>Shift+C</Kbd>
							</Tooltip.Content>
						</Tooltip>

						<Popover>
							<Popover.Trigger>
								<Button
									isIconOnly
									variant="ghost"
									aria-label="Limits"
									className="h-9 w-9 min-w-9"
								>
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
											<span className="text-xs bg-default/30 px-2 py-0.5 rounded font-mono">
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
											<Label className="text-sm font-bold">Upload Limit</Label>
											<span className="text-xs bg-default/30 px-2 py-0.5 rounded font-mono">
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

						<div className="w-px h-6 bg-default/30 mx-1 hidden sm:block" />

						<Select
							className="hidden sm:block shrink-0"
							selectedKey={activeServerId || undefined}
							onSelectionChange={(key) => {
								if (key === manageServersId) {
									navigate({ to: "/settings/network" });
								} else {
									setActiveServer(String(key));
								}
							}}
							aria-label="Select Server"
						>
							<Select.Trigger className="h-9 px-3 bg-default/10 rounded-xl hover:bg-default/20 transition-colors border-none outline-none flex items-center justify-between gap-3 min-w-0">
								<Select.Value className="text-xs font-bold truncate pr-6" />
								<Select.Indicator className="text-muted shrink-0">
									<IconChevronDown className="w-3 h-3" />
								</Select.Indicator>
							</Select.Trigger>
							<Select.Popover className="min-w-[220px] p-1.5 bg-background border border-border rounded-2xl shadow-2xl">
								<ListBox className="gap-1">
									<ListBox.Section>
										<Header className="px-3 py-2 text-[10px] font-black uppercase text-muted tracking-widest flex items-center gap-2">
											<div className="w-1 h-3 bg-accent rounded-full" />
											Active Servers
										</Header>
										{servers.map((s) => (
											<ListBox.Item
												key={s.id}
												id={s.id}
												textValue={s.name}
												className="px-3 py-2.5 rounded-xl data-[hover=true]:bg-default/15 text-xs font-bold cursor-pointer outline-none flex items-center gap-3 transition-colors group"
											>
												<div className="p-1.5 rounded-lg bg-default/10 group-data-[hover=true]:bg-accent/20 transition-colors">
													<IconServer className="w-3.5 h-3.5 text-muted group-data-[hover=true]:text-accent" />
												</div>
												<span className="flex-1 truncate">{s.name}</span>
												{s.id === activeServerId && (
													<div className="w-2 h-2 rounded-full bg-success shadow-[0_0_8px_rgba(var(--success-rgb),0.5)]" />
												)}
											</ListBox.Item>
										))}
									</ListBox.Section>
									<Separator className="h-px bg-border/50 my-1.5 mx-2" />
									<ListBox.Item
										id={manageServersId}
										textValue="Manage Servers"
										className="px-3 py-2.5 rounded-xl data-[hover=true]:bg-accent/10 text-xs font-black text-accent cursor-pointer outline-none flex items-center gap-3 transition-colors"
									>
										<div className="p-1.5 rounded-lg bg-accent/10">
											<IconGear className="w-3.5 h-3.5" />
										</div>
										<span className="uppercase tracking-widest text-[10px]">
											Manage Servers
										</span>
									</ListBox.Item>
								</ListBox>
							</Select.Popover>
						</Select>

						<Button
							isIconOnly
							variant="ghost"
							onPress={toggleTheme}
							aria-label="Theme"
							className="h-9 w-9 min-w-9"
						>
							{isDark ? (
								<IconSun className="w-5 h-5" />
							) : (
								<IconMoon className="w-5 h-5" />
							)}
						</Button>
					</div>
				</header>

				<main className={cn(
					"flex-1 overflow-y-auto",
					location.pathname === "/files" ? "p-0" : "p-4 md:p-8"
				)}>
					{children}
				</main>
			</div>
		</div>
	);
};