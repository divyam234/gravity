import { Button, Card, Spinner, Chip, Checkbox, cn } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useMemo } from "react";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnifyingGlass from "~icons/gravity-ui/magnifier";
import IconArrowsRotateRight from "~icons/gravity-ui/arrows-rotate-right";
import IconGear from "~icons/gravity-ui/gear";
import IconCloud from "~icons/gravity-ui/cloud";
import IconFunnel from "~icons/gravity-ui/funnel";
import IconClock from "~icons/gravity-ui/clock";
import { useSearch } from "../hooks/useSearch";
import { FormSelect, FormTextField } from "../components/ui/FormFields";
import { formatBytes } from "../lib/utils";

export const Route = createFileRoute("/settings/search")({
	component: SearchSettingsPage,
});

const searchSettingsSchema = z.object({
	interval: z.number().min(0),
	excludedPatterns: z.string(),
	includedExtensions: z.string(),
	minSizeBytes: z.number().min(0),
});

const INTERVAL_OPTIONS = [
	{ value: 0, label: "Disabled" },
	{ value: 60, label: "Hourly" },
	{ value: 360, label: "Every 6 Hours" },
	{ value: 720, label: "Every 12 Hours" },
	{ value: 1440, label: "Daily" },
	{ value: 10080, label: "Weekly" },
];

function SearchSettingsPage() {
	const navigate = useNavigate();
	const { configs, isLoading, triggerIndex, updateConfigs } = useSearch();

	if (isLoading) {
		return (
			<div className="flex flex-col h-full space-y-6">
				<div className="flex items-center gap-4 px-6 shrink-0">
					<Button
						variant="ghost"
						isIconOnly
						onPress={() => navigate({ to: "/settings" })}
						className="rounded-2xl"
					>
						<IconChevronLeft className="w-5 h-5" />
					</Button>
					<h2 className="text-2xl font-bold tracking-tight">Search Indexing</h2>
				</div>
				<div className="flex-1 flex items-center justify-center">
					<Spinner size="lg" />
				</div>
			</div>
		);
	}

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center gap-4 px-6 shrink-0">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/settings" })}
					className="rounded-2xl"
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<div>
					<h2 className="text-3xl font-black tracking-tight uppercase">Search Indexing</h2>
					<p className="text-[10px] text-muted font-black uppercase tracking-[0.2em] opacity-70">
						Global indexing rules & remote management
					</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/20 rounded-[2.5rem] border border-border/50 overflow-hidden min-h-0 mx-2 mb-2">
				{configs.length === 0 ? (
					<div className="flex h-full items-center justify-center p-8 text-center">
						<Card className="p-12 bg-background/50 border-border border-dashed text-center max-w-md rounded-[3rem]">
							<div className="w-20 h-20 bg-default/10 rounded-full flex items-center justify-center mb-6 mx-auto">
								<IconMagnifyingGlass className="w-10 h-10 text-muted" />
							</div>
							<h4 className="font-bold text-xl mb-2">No remotes available</h4>
							<p className="text-sm text-muted mb-8">
								Configure cloud remotes first to enable indexing.
							</p>
							<Button
								variant="primary"
								className="rounded-xl font-bold h-12 px-8"
								onPress={() => navigate({ to: "/settings/cloud" })}
							>
								Go to Cloud Settings
							</Button>
						</Card>
					</div>
				) : (
					<SearchSettingsLayout
						configs={configs}
						updateConfigs={updateConfigs}
						triggerIndex={triggerIndex}
					/>
				)}
			</div>
		</div>
	);
}

interface SearchSettingsLayoutProps {
	configs: any[];
	updateConfigs: any;
	triggerIndex: any;
}

function SearchSettingsLayout({
	configs,
	updateConfigs,
	triggerIndex,
}: SearchSettingsLayoutProps) {
	const [selectedRemotes, setSelectedRemotes] = useState<Set<string>>(new Set());

	const toggleSelection = (remote: string) => {
		const next = new Set(selectedRemotes);
		if (next.has(remote)) {
			next.delete(remote);
		} else {
			next.add(remote);
		}
		setSelectedRemotes(next);
	};

	const selectAll = () => {
		setSelectedRemotes(new Set(configs.map((c) => c.remote)));
	};

	const deselectAll = () => {
		setSelectedRemotes(new Set());
	};

	const selectedConfigs = useMemo(() => {
		return configs.filter((c) => selectedRemotes.has(c.remote));
	}, [configs, selectedRemotes]);

	return (
		<div className="flex flex-col lg:flex-row h-full">
			{/* Left: Remote Grid */}
			<div className="flex-1 flex flex-col min-h-0">
				<div className="p-6 border-b border-border/30 flex items-center justify-between bg-background/40 backdrop-blur-md sticky top-0 z-10">
					<div className="flex items-center gap-3">
						<div className="w-1.5 h-5 bg-accent rounded-full" />
						<h3 className="font-black text-[10px] uppercase tracking-[0.2em] text-muted">
							Remotes ({configs.length})
						</h3>
					</div>
					<div className="flex gap-2">
						<Button
							size="sm"
							variant="ghost"
							className="h-8 rounded-xl text-[10px] font-black uppercase tracking-widest"
							onPress={selectAll}
						>
							Select All
						</Button>
						<Button
							size="sm"
							variant="ghost"
							className="h-8 rounded-xl text-[10px] font-black uppercase tracking-widest text-danger"
							onPress={deselectAll}
						>
							Clear
						</Button>
					</div>
				</div>

				<div className="flex-1 overflow-y-auto p-6">
					<div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
						{configs.map((config) => (
							<RemoteCard
								key={config.remote}
								config={config}
								isSelected={selectedRemotes.has(config.remote)}
								onToggle={() => toggleSelection(config.remote)}
								onIndex={() => triggerIndex.mutate(config.remote)}
								isIndexing={
									triggerIndex.isPending && triggerIndex.variables === config.remote
								}
							/>
						))}
					</div>
				</div>
			</div>

			{/* Right: Config Panel */}
			<div className="w-full lg:w-[400px] flex flex-col bg-surface/60 border-l border-border/30 shadow-xl">
				<div className="p-8 border-b border-border/30">
					<div className="flex items-center gap-3 mb-2">
						<div className="w-2 h-8 bg-accent rounded-full" />
						<h3 className="text-2xl font-black tracking-tight uppercase">Configuration</h3>
					</div>
					<p className="text-[10px] text-muted font-black uppercase tracking-[0.2em] opacity-70">
						{selectedRemotes.size === 0
							? "Select remotes to edit"
							: selectedRemotes.size === 1
								? `Editing ${Array.from(selectedRemotes)[0]}`
								: `Editing ${selectedRemotes.size} target remotes`}
					</p>
				</div>

				<div className="flex-1 overflow-y-auto p-8">
					{selectedRemotes.size > 0 ? (
						<SearchSettingsForm
							key={Array.from(selectedRemotes).sort().join(",")}
							selectedConfigs={selectedConfigs}
							updateConfigs={updateConfigs}
						/>
					) : (
						<div className="h-full flex flex-col items-center justify-center text-center p-12">
							<div className="w-24 h-24 bg-default/5 rounded-[2.5rem] flex items-center justify-center mb-6 border border-border/50 shadow-inner">
								<IconGear className="w-10 h-10 text-muted/40 animate-[spin_10s_linear_infinite]" />
							</div>
							<p className="font-black uppercase tracking-[0.2em] text-muted text-[10px]">
								No Selection
							</p>
							<p className="text-xs text-muted mt-4 leading-relaxed font-medium">
								Select one or more cloud remotes from the grid to configure their indexing rules.
							</p>
						</div>
					)}
				</div>
			</div>
		</div>
	);
}

function RemoteCard({ config, isSelected, onToggle, onIndex, isIndexing }: any) {
	const intervalLabel =
		INTERVAL_OPTIONS.find((o) => o.value === config.autoIndexIntervalMin)?.label ||
		"Custom";

	return (
		<div
			role="button"
			tabIndex={0}
			className={cn(
				"w-full text-left relative transition-all duration-300 border-2 cursor-pointer group hover:-translate-y-1 rounded-[2.2rem] overflow-hidden outline-none focus-visible:ring-2 focus-visible:ring-accent",
				isSelected
					? "border-accent bg-accent/5 shadow-md scale-[1.02] z-10"
					: "border-border/20 bg-surface/40 hover:bg-surface hover:border-border/60 shadow-sm",
			)}
			onClick={onToggle}
			onKeyDown={(e) => {
				if (e.key === "Enter" || e.key === " ") {
					e.preventDefault();
					onToggle();
				}
			}}
		>
			<div className="p-6 space-y-5">
				{/* Header */}
				<div className="flex items-start justify-between gap-3">
					<div className="flex items-center gap-4 overflow-hidden">
						<div
							className={cn(
								"w-12 h-12 rounded-2xl flex items-center justify-center shrink-0 transition-all duration-500",
								isSelected ? "bg-accent text-accent-foreground shadow-sm" : "bg-default/10 text-muted",
							)}
						>
							<IconCloud className="w-6 h-6" />
						</div>
						<div className="min-w-0">
							<h4 className="font-bold text-base truncate tracking-tight">{config.remote}</h4>
							<div className="flex items-center gap-2 mt-0.5">
								{config.status === "indexing" ? (
									<Chip
										color="accent"
										size="sm"
										variant="soft"
										className="h-5 text-[9px] font-black uppercase px-2 animate-pulse"
									>
										Indexing...
									</Chip>
								) : config.lastIndexedAt ? (
									<span className="text-[10px] text-success font-black uppercase tracking-widest flex items-center gap-1.5">
										<div className="w-1.5 h-1.5 rounded-full bg-success shadow-sm" />
										Indexed
									</span>
								) : (
									<span className="text-[10px] text-warning font-black uppercase tracking-widest opacity-60">
										Idle
									</span>
								)}
							</div>
						</div>
					</div>

					<div
						className={cn(
							"transition-transform duration-300",
							isSelected ? "scale-110" : "scale-100 opacity-40",
						)}
					>
						<Checkbox isSelected={isSelected} className="pointer-events-none" />
					</div>
				</div>

				{/* Config Badges */}
				<div className="flex flex-wrap gap-2">
					<Chip
						size="sm"
						variant="soft"
						color="accent"
						className="h-7 gap-1.5 pl-2 rounded-xl bg-accent/10 border border-accent/10"
					>
						<IconClock className="w-3.5 h-3.5" />
						<span className="text-[9px] font-black uppercase tracking-tight">{intervalLabel}</span>
					</Chip>

					{(config.minSizeBytes || 0) > 0 && (
						<Chip
							size="sm"
							variant="soft"
							color="warning"
							className="h-7 gap-1.5 pl-2 rounded-xl bg-warning/10 border border-warning/10"
						>
							<IconFunnel className="w-3.5 h-3.5" />
							<span className="text-[9px] font-black uppercase tracking-tight">
								{">"}
								{formatBytes(config.minSizeBytes)}
							</span>
						</Chip>
					)}

					{config.includedExtensions && (
						<Chip
							size="sm"
							variant="soft"
							color="success"
							className="h-7 gap-1.5 pl-2 rounded-xl bg-success/10 border border-success/10"
						>
							<span className="text-[9px] font-black uppercase tracking-tight">
								Ext: {config.includedExtensions}
							</span>
						</Chip>
					)}
				</div>

				{/* Action Bar */}
				<div className="pt-3 flex items-center justify-between border-t border-border/20">
					<div className="text-[9px] text-muted font-black uppercase tracking-widest opacity-50 truncate pr-2">
						{config.lastIndexedAt && new Date(config.lastIndexedAt).toLocaleDateString()}
					</div>
					<Button
						size="sm"
						variant="ghost"
						className="h-9 text-[10px] font-black uppercase tracking-widest rounded-xl text-accent hover:bg-accent/10 transition-all border border-transparent hover:border-accent/20"
						isDisabled={config.status === "indexing" || isIndexing}
						onPress={(e) => {
							e.continuePropagation();
							onIndex();
						}}
					>
						<IconArrowsRotateRight
							className={cn(
								"w-3.5 h-3.5 mr-1.5",
								(config.status === "indexing" || isIndexing) && "animate-spin",
							)}
						/>
						{config.lastIndexedAt ? "Rebuild" : "Start"}
					</Button>
				</div>
			</div>
		</div>
	);
}

function SearchSettingsForm({
	selectedConfigs,
	updateConfigs,
}: { selectedConfigs: any[]; updateConfigs: any }) {
	// Determine common values
	const commonValues = useMemo(() => {
		if (selectedConfigs.length === 0) return null;

		const first = selectedConfigs[0];
		const common = {
			interval: first.autoIndexIntervalMin,
			excludedPatterns: first.excludedPatterns || "",
			includedExtensions: first.includedExtensions || "",
			minSizeBytes: first.minSizeBytes || 0,
		};

		// Check consistency
		for (let i = 1; i < selectedConfigs.length; i++) {
			const c = selectedConfigs[i];
			if (c.autoIndexIntervalMin !== common.interval) common.interval = -1; // -1 indicates mixed
			if ((c.excludedPatterns || "") !== common.excludedPatterns)
				common.excludedPatterns = "__mixed__";
			if ((c.includedExtensions || "") !== common.includedExtensions)
				common.includedExtensions = "__mixed__";
			if ((c.minSizeBytes || 0) !== common.minSizeBytes) common.minSizeBytes = -1;
		}

		return common;
	}, [selectedConfigs]);

	const defaultValues = {
		interval:
			commonValues?.interval === -1 ? 0 : (commonValues?.interval ?? 1440),
		excludedPatterns:
			commonValues?.excludedPatterns === "__mixed__"
				? ""
				: (commonValues?.excludedPatterns ?? ""),
		includedExtensions:
			commonValues?.includedExtensions === "__mixed__"
				? ""
				: (commonValues?.includedExtensions ?? ""),
		minSizeBytes:
			commonValues?.minSizeBytes === -1 ? 0 : (commonValues?.minSizeBytes ?? 0),
	};

	const form = useForm({
		defaultValues,
		validators: {
			onChange: searchSettingsSchema,
		},
		onSubmit: async ({ value }) => {
			const batch: Record<string, any> = {};

			selectedConfigs.forEach((config) => {
				batch[config.remote] = {
					...value,
				};
			});

			await updateConfigs.mutateAsync(batch);
		},
	});

	// Track mixed states for UI hints
	const isMixed = {
		interval: commonValues?.interval === -1,
		patterns: commonValues?.excludedPatterns === "__mixed__",
		extensions: commonValues?.includedExtensions === "__mixed__",
		size: commonValues?.minSizeBytes === -1,
	};

	return (
		<div className="space-y-10 animate-in slide-in-from-right-8 duration-500 ease-expo">
			{/* Interval */}
			<div className="space-y-4">
				<FormSelect
					form={form}
					name="interval"
					label={
						<span className="flex items-center justify-between w-full">
							Update Frequency
							{isMixed.interval && (
								<Chip size="sm" color="warning" variant="soft" className="h-4 text-[8px] font-black uppercase px-1">
									Mixed
								</Chip>
							)}
						</span>
					}
					items={INTERVAL_OPTIONS}
				/>
				{isMixed.interval && (
					<p className="text-[10px] text-warning px-1 font-black uppercase tracking-wider opacity-80">
						Mixed frequencies. Saving will overwrite them.
					</p>
				)}
			</div>

			<div className="h-px bg-border/20" />

			{/* Filters */}
			<div className="space-y-8">
				<FormTextField
					form={form}
					name="excludedPatterns"
					label={
						<span className="flex items-center justify-between w-full">
							Exclude Patterns (Regex)
							{isMixed.patterns && (
								<Chip size="sm" color="warning" variant="soft" className="h-4 text-[8px] font-black uppercase px-1">
									Mixed
								</Chip>
							)}
						</span>
					}
					placeholder={
						isMixed.patterns ? "Mixed values (leave empty to clear)" : "e.g. /node_modules/"
					}
				/>

				<FormTextField
					form={form}
					name="includedExtensions"
					label={
						<span className="flex items-center justify-between w-full">
							Include Extensions
							{isMixed.extensions && (
								<Chip size="sm" color="warning" variant="soft" className="h-4 text-[8px] font-black uppercase px-1">
									Mixed
								</Chip>
							)}
						</span>
					}
					placeholder={
						isMixed.extensions ? "Mixed values (leave empty to clear)" : "e.g. mp4, mkv"
					}
				/>

				<FormTextField
					form={form}
					name="minSizeBytes"
					label={
						<span className="flex items-center justify-between w-full">
							Minimum File Size (MB)
							{isMixed.size && (
								<Chip size="sm" color="warning" variant="soft" className="h-4 text-[8px] font-black uppercase px-1">
									Mixed
								</Chip>
							)}
						</span>
					}
					type="number"
					placeholder={isMixed.size ? "Mixed" : "0"}
					format={(val) => String(Math.floor((val || 0) / (1024 * 1024)))}
					parse={(val) => parseInt(val || "0") * 1024 * 1024}
					endContent={<span className="text-[10px] text-muted font-black uppercase px-2">MB</span>}
				/>
			</div>

			<div className="pt-6 sticky bottom-0 bg-background/80 backdrop-blur-md pb-4 -mx-2 px-2">
				<form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting]}>
					{([canSubmit, isSubmitting]) => (
						<Button
							fullWidth
							variant="primary"
							className="font-black uppercase tracking-widest shadow-lg rounded-[1.5rem] h-14"
							onPress={() => form.handleSubmit()}
							isDisabled={!canSubmit}
							isPending={isSubmitting}
						>
							Apply to {selectedConfigs.length} Remote
							{selectedConfigs.length !== 1 ? "s" : ""}
						</Button>
					)}
				</form.Subscribe>
			</div>
		</div>
	);
}