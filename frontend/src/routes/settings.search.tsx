import { Button, Card, ScrollShadow, Spinner, Chip, Checkbox, cn } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useMemo } from "react";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnifyingGlass from "~icons/gravity-ui/magnifier";
import IconArrowsRotateRight from "~icons/gravity-ui/arrows-rotate-right";
import IconCircleCheckFill from "~icons/gravity-ui/circle-check-fill";
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
				<div className="flex items-center gap-4 px-2 shrink-0">
					<Button
						variant="ghost"
						isIconOnly
						onPress={() => navigate({ to: "/settings" })}
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
			<div className="flex items-center gap-4 px-2 shrink-0">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/settings" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<div>
					<h2 className="text-2xl font-bold tracking-tight">Search Indexing</h2>
					<p className="text-xs text-muted">Global indexing rules & remote management</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				{configs.length === 0 ? (
					<div className="flex h-full items-center justify-center p-8">
						<Card className="p-8 bg-background/50 border-border border-dashed text-center max-w-md">
							<div className="w-16 h-16 bg-default/10 rounded-full flex items-center justify-center mb-4 mx-auto">
								<IconMagnifyingGlass className="w-8 h-8 text-muted" />
							</div>
							<h4 className="font-bold text-lg mb-2">No remotes available</h4>
							<p className="text-sm text-muted mb-6">
								Configure cloud remotes first to enable indexing.
							</p>
							<Button
								variant="primary"
								className="rounded-xl font-bold"
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
			<div className="flex-1 flex flex-col min-h-0 border-r border-border/50">
				<div className="p-4 border-b border-border/50 flex items-center justify-between bg-background/30 backdrop-blur-sm sticky top-0 z-10">
					<div className="flex items-center gap-2">
						<h3 className="font-bold text-sm uppercase tracking-wider text-muted">
							Remotes ({configs.length})
						</h3>
					</div>
					<div className="flex gap-2">
						<Button size="sm" variant="ghost" className="h-8 text-xs font-bold" onPress={selectAll}>
							All
						</Button>
						<Button
							size="sm"
							variant="ghost"
							className="h-8 text-xs font-bold"
							onPress={deselectAll}
						>
							None
						</Button>
					</div>
				</div>

				<ScrollShadow className="flex-1 p-4">
					<div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
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
				</ScrollShadow>
			</div>

			{/* Right: Config Panel */}
			<div className="w-full lg:w-[400px] flex flex-col bg-background/50 backdrop-blur-md">
				<div className="p-6 border-b border-border/50">
					<div className="flex items-center gap-3 mb-1">
						<div className="w-1.5 h-6 bg-accent rounded-full" />
						<h3 className="text-lg font-bold">Configuration</h3>
					</div>
					<p className="text-xs text-muted">
						{selectedRemotes.size === 0
							? "Select remotes to edit settings"
							: selectedRemotes.size === 1
								? `Editing ${Array.from(selectedRemotes)[0]}`
								: `Editing ${selectedRemotes.size} remotes`}
					</p>
				</div>

				<ScrollShadow className="flex-1 p-6">
					{selectedRemotes.size > 0 ? (
						<SearchSettingsForm
							key={Array.from(selectedRemotes).sort().join(",")}
							selectedConfigs={selectedConfigs}
							updateConfigs={updateConfigs}
						/>
					) : (
						<div className="h-full flex flex-col items-center justify-center text-center opacity-50 p-8">
							<IconGear className="w-12 h-12 mb-4 text-muted" />
							<p className="font-bold text-muted">No remotes selected</p>
							<p className="text-xs text-muted mt-2">
								Click on a remote card to view and edit its indexing settings.
							</p>
						</div>
					)}
				</ScrollShadow>
			</div>
		</div>
	);
}

function RemoteCard({ config, isSelected, onToggle, onIndex, isIndexing }: any) {
	const intervalLabel =
		INTERVAL_OPTIONS.find((o) => o.value === config.autoIndexIntervalMin)?.label ||
		"Custom";

	return (
		<button
			type="button"
			className={cn(
				"w-full text-left relative transition-all duration-200 border-2 cursor-pointer group hover:-translate-y-0.5 rounded-3xl overflow-hidden",
				isSelected
					? "border-accent bg-accent/5 shadow-lg shadow-accent/10"
					: "border-transparent bg-surface hover:bg-default/10 hover:shadow-md",
			)}
			onClick={onToggle}
			onKeyDown={(e) => {
				if (e.key === "Enter" || e.key === " ") {
					e.preventDefault();
					onToggle();
				}
			}}
		>
			<div className="p-5 space-y-4">
				{/* Header */}
				<div className="flex items-start justify-between gap-3">
					<div className="flex items-center gap-3 overflow-hidden">
						<div
							className={cn(
								"w-10 h-10 rounded-2xl flex items-center justify-center shrink-0 transition-colors",
								isSelected ? "bg-accent text-accent-foreground" : "bg-default/10 text-muted",
							)}
						>
							<IconCloud className="w-5 h-5" />
						</div>
						<div className="min-w-0">
							<h4 className="font-bold text-base truncate">{config.remote}</h4>
							<div className="flex items-center gap-2 mt-0.5">
								{config.status === "indexing" ? (
									<Chip
										color="accent"
										size="sm"
										variant="soft"
										className="h-5 text-[9px] font-black uppercase px-1"
									>
										Indexing...
									</Chip>
								) : config.lastIndexedAt ? (
									<span className="text-[10px] text-success font-bold flex items-center gap-1">
										<IconCircleCheckFill className="w-3 h-3" /> Indexed
									</span>
								) : (
									<span className="text-[10px] text-warning font-bold">Never Indexed</span>
								)}
							</div>
						</div>
					</div>

					<Checkbox
						isSelected={isSelected}
						className="pointer-events-none"
					/>
				</div>

				{/* Config Badges */}
				<div className="flex flex-wrap gap-1.5">
					<Chip size="sm" variant="soft" color="accent" className="h-6 gap-1 pl-1">
						<IconClock className="w-3 h-3" />
						<span className="text-[10px] font-bold">{intervalLabel}</span>
					</Chip>

					{(config.minSizeBytes || 0) > 0 && (
						<Chip size="sm" variant="soft" color="warning" className="h-6 gap-1 pl-1">
							<IconFunnel className="w-3 h-3" />
							<span className="text-[10px] font-bold">
								{">"}
								{formatBytes(config.minSizeBytes)}
							</span>
						</Chip>
					)}

					{config.includedExtensions && (
						<Chip size="sm" variant="soft" color="success" className="h-6 gap-1 pl-1">
							<span className="text-[10px] font-bold">Ext: {config.includedExtensions}</span>
						</Chip>
					)}
				</div>

				{/* Action Bar */}
				<div className="pt-2 flex justify-end border-t border-divider">
					<Button
						size="sm"
						variant="ghost"
						className="h-8 text-xs font-bold rounded-xl text-accent"
						isDisabled={config.status === "indexing" || isIndexing}
						onPress={(e) => {
							e.continuePropagation();
							onIndex();
						}}
					>
						<IconArrowsRotateRight
							className={cn(
								"w-3 h-3 mr-1",
								(config.status === "indexing" || isIndexing) && "animate-spin",
							)}
						/>
						{config.lastIndexedAt ? "Rebuild" : "Start"}
					</Button>
				</div>
			</div>
		</button>
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
		<div className="space-y-8 animate-in slide-in-from-right-4 duration-300">
			{/* Interval */}
			<div className="space-y-4">
				<FormSelect
					form={form}
					name="interval"
					label={
						<span className="flex items-center justify-between w-full">
							Update Frequency
							{isMixed.interval && (
								<Chip size="sm" color="warning" variant="soft" className="h-4 text-[9px] px-1">
									Mixed
								</Chip>
							)}
						</span>
					}
					items={INTERVAL_OPTIONS}
				/>
				{isMixed.interval && (
					<p className="text-[10px] text-warning px-1 font-bold uppercase tracking-wider">
						Mixed frequencies. Saving will overwrite them.
					</p>
				)}
			</div>

			<div className="h-px bg-border/50" />

			{/* Filters */}
			<div className="space-y-6">
				<div>
					<FormTextField
						form={form}
						name="excludedPatterns"
						label={
							<span className="flex items-center justify-between w-full">
								Exclude Patterns (Regex)
								{isMixed.patterns && (
									<Chip size="sm" color="warning" variant="soft" className="h-4 text-[9px] px-1">
										Mixed
									</Chip>
								)}
							</span>
						}
						placeholder={
							isMixed.patterns ? "Mixed values (leave empty to clear)" : "e.g. /node_modules/"
						}
					/>
				</div>

				<div>
					<FormTextField
						form={form}
						name="includedExtensions"
						label={
							<span className="flex items-center justify-between w-full">
								Include Extensions
								{isMixed.extensions && (
									<Chip size="sm" color="warning" variant="soft" className="h-4 text-[9px] px-1">
										Mixed
									</Chip>
								)}
							</span>
						}
						placeholder={
							isMixed.extensions ? "Mixed values (leave empty to clear)" : "e.g. mp4, mkv"
						}
					/>
				</div>

				<div>
					<FormTextField
						form={form}
						name="minSizeBytes"
						label={
							<span className="flex items-center justify-between w-full">
								Minimum File Size (MB)
								{isMixed.size && (
									<Chip size="sm" color="warning" variant="soft" className="h-4 text-[9px] px-1">
										Mixed
									</Chip>
								)}
							</span>
						}
						type="number"
						placeholder={isMixed.size ? "Mixed" : "0"}
						format={(val) => String(Math.floor((val || 0) / (1024 * 1024)))}
						parse={(val) => parseInt(val || "0") * 1024 * 1024}
						endContent={<span className="text-xs text-muted font-bold px-2">MB</span>}
					/>
				</div>
			</div>

			<div className="pt-4">
				<form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting]}>
					{([canSubmit, isSubmitting]) => (
						<Button
							fullWidth
							variant="primary"
							className="font-bold shadow-lg shadow-primary/20 rounded-2xl h-12"
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