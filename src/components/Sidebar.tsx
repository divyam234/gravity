import { Link } from "@tanstack/react-router";
import type React from "react";
import IconArrowDown from "~icons/gravity-ui/arrow-down";
import IconCheck from "~icons/gravity-ui/check";
import IconClock from "~icons/gravity-ui/clock";
import IconGear from "~icons/gravity-ui/gear";
import IconLayoutHeaderCellsLarge from "~icons/gravity-ui/layout-header-cells-large";
import IconPulse from "~icons/gravity-ui/pulse";
import { useAllTasks, useGlobalStat } from "../hooks/useAria2";
import { cn, formatBytes } from "../lib/utils";

export const Sidebar: React.FC = () => {
	const { active, waiting, stopped } = useAllTasks();
	const { data: stats } = useGlobalStat();
	const allCount = active.length + waiting.length + stopped.length;

	const navItems = [
		{
			label: "Dashboard",
			icon: <IconLayoutHeaderCellsLarge className="w-5 h-5" />,
			to: "/",
			count: null,
		},
		{
			label: "All Tasks",
			icon: <IconPulse className="w-5 h-5" />,
			to: "/tasks/all",
			count: allCount,
		},
		{
			label: "Downloading",
			icon: <IconArrowDown className="w-5 h-5" />,
			to: "/tasks/active",
			count: active.length,
			color: "text-success",
		},
		{
			label: "Waiting",
			icon: <IconClock className="w-5 h-5" />,
			to: "/tasks/waiting",
			count: waiting.length,
			color: "text-warning",
		},
		{
			label: "Stopped",
			icon: <IconCheck className="w-5 h-5" />,
			to: "/tasks/stopped",
			count: stopped.length,
			color: "text-danger",
		},
		{
			label: "Settings",
			icon: <IconGear className="w-5 h-5" />,
			to: "/settings",
			count: null,
		},
	];

	return (
		<aside className="w-64 border-r border-default-100 bg-default-50/30 flex flex-col h-full shrink-0">
			<div className="p-6 flex items-center gap-3">
				<div className="w-10 h-10 bg-accent rounded-xl flex items-center justify-center text-accent-foreground font-bold text-xl shadow-lg shadow-accent/20">
					A
				</div>
				<div>
					<h1 className="font-bold tracking-tight">Aria2 Manager</h1>
					<p className="text-[10px] text-default-400 uppercase font-black tracking-widest leading-none">
						Control Panel
					</p>
				</div>
			</div>

			<nav className="flex-1 px-3 space-y-1 mt-4">
				{navItems.map((item) => (
					<Link
						key={item.label}
						to={item.to as any}
						className={cn(
							"flex items-center justify-between px-4 py-3 rounded-2xl transition-all group outline-none",
							"hover:bg-default-100/50",
							"data-[status=active]:bg-accent data-[status=active]:text-accent-foreground data-[status=active]:shadow-lg data-[status=active]:shadow-accent/20",
						)}
					>
						<div className="flex items-center gap-3">
							<span className={cn(item.color)}>{item.icon}</span>
							<span className="text-sm font-bold tracking-tight">
								{item.label}
							</span>
						</div>
						{item.count !== null && (
							<span
								className={cn(
									"text-[10px] font-black px-2 py-0.5 rounded-full bg-default-100 group-hover:bg-default-200 transition-colors",
									"group-data-[status=active]:bg-accent-foreground/20 group-data-[status=active]:text-accent-foreground",
								)}
							>
								{item.count}
							</span>
						)}
					</Link>
				))}
			</nav>

			<div className="p-6 mt-auto">
				<div className="p-4 rounded-3xl bg-default-100/50 border border-default-100 flex flex-col gap-2">
					<p className="text-[10px] font-black uppercase text-default-400 tracking-widest">
						Session Speed
					</p>
					<div className="flex flex-col">
						<span className="text-xs font-bold text-success">
							DL: {formatBytes(stats?.downloadSpeed || 0)}/s
						</span>
						<span className="text-xs font-bold text-accent">
							UL: {formatBytes(stats?.uploadSpeed || 0)}/s
						</span>
					</div>
				</div>
			</div>
		</aside>
	);
};
