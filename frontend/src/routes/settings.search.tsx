import { Button, Card, ScrollShadow, Spinner, Chip, ListBox, Checkbox, Tooltip } from "@heroui/react";
import type { Selection } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnifyingGlass from "~icons/gravity-ui/magnifier";
import IconArrowsRotateRight from "~icons/gravity-ui/arrows-rotate-right";
import IconCircleCheckFill from "~icons/gravity-ui/circle-check-fill";
import IconGear from "~icons/gravity-ui/gear";
import { useSearch } from "../hooks/useSearch";
import { cn } from "../lib/utils";
import { FormSelect, FormTextField } from "../components/ui/FormFields";

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
					<Button variant="ghost" isIconOnly onPress={() => navigate({ to: "/settings" })}>
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
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{configs.length === 0 ? (
							<Card className="p-8 bg-background/50 border-border border-dashed text-center">
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
						) : (
							<SearchSettingsForm
								key={configs.map(c => c.remote).join(',')}
								configs={configs}
								updateConfigs={updateConfigs}
								triggerIndex={triggerIndex}
							/>
						)}
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}

interface SearchSettingsFormProps {
  configs: any[];
  updateConfigs: any;
  triggerIndex: any;
}

function SearchSettingsForm({ configs, updateConfigs, triggerIndex }: SearchSettingsFormProps) {
	const [includedRemotes, setIncludedRemotes] = useState<Selection>(() => {
    const included = new Set(
      configs
        .filter((c) => c.autoIndexIntervalMin > 0 || c.lastIndexedAt)
        .map((c) => c.remote),
    );
    // Default to all if nothing configured yet
    if (configs.every((c) => c.autoIndexIntervalMin === 0 && !c.lastIndexedAt)) {
      return "all";
    }
    return included;
  });

	const [isExpanded, setIsExpanded] = useState(false);

  // Derive global defaults from the best available template in the server data
  const defaultValues = useState(() => {
    const template = configs.find(c => c.autoIndexIntervalMin > 0 || c.excludedPatterns || c.includedExtensions || (c.minSizeBytes ?? 0) > 0) || configs[0];
    return {
      interval: template.autoIndexIntervalMin || 1440,
      excludedPatterns: template.excludedPatterns || "",
      includedExtensions: template.includedExtensions || "",
      minSizeBytes: template.minSizeBytes || 0,
    };
  })[0];

	const form = useForm({
		defaultValues,
		validators: {
			onChange: searchSettingsSchema,
		},
		onSubmit: async ({ value }) => {
			const batch: Record<string, any> = {};
			configs.forEach((config) => {
				const isIncluded =
					includedRemotes === "all" || includedRemotes.has(config.remote);
				batch[config.remote] = {
					...value,
					interval: isIncluded ? value.interval : 0,
				};
			});
			await updateConfigs.mutateAsync(batch);
		},
	});

	return (
    <>
      <section className="space-y-6">
        <div className="flex items-center gap-3">
          <div className="w-1.5 h-6 bg-accent rounded-full" />
          <h3 className="text-lg font-bold">Global Configuration</h3>
        </div>

        <Card className="bg-background/50 border-border overflow-hidden">
          <div className="p-6 flex flex-col md:flex-row md:items-center justify-between gap-6">
            <div className="flex-1">
              <h4 className="font-bold text-lg">Indexing Rules</h4>
              <p className="text-sm text-muted">
                Settings applied to all selected remotes below
              </p>
            </div>

            <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 shrink-0">
              <div className="flex flex-col gap-1.5">
                <FormSelect
                  form={form}
                  name="interval"
                  label="Update Frequency"
                  items={INTERVAL_OPTIONS}
                />
              </div>

              <div className="flex items-end h-full pt-5 md:pt-0 gap-2">
                <Button
                  variant="ghost"
                  isIconOnly
                  className={cn(
                    "h-[38px] w-[38px] rounded-xl",
                    isExpanded && "bg-accent/10 text-accent",
                  )}
                  onPress={() => setIsExpanded(!isExpanded)}
                >
                  <IconGear className="w-4 h-4" />
                </Button>
                <form.Subscribe
                  selector={(state) => [state.canSubmit, state.isSubmitting]}
                >
                  {([canSubmit, isSubmitting]) => (
                    <Button
                      variant="primary"
                      className="rounded-xl font-bold h-[38px] px-6 shadow-lg shadow-primary/20"
                      onPress={() => form.handleSubmit()}
                      isDisabled={!canSubmit}
                      isPending={isSubmitting}
                    >
                      Apply Rules
                    </Button>
                  )}
                </form.Subscribe>
              </div>
            </div>
          </div>

          {isExpanded && (
            <div className="px-6 pb-8 pt-2 border-t border-border/50 animate-in slide-in-from-top-2 duration-200">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mt-4">
                <div className="space-y-6">
                  <FormTextField
                    form={form}
                    name="excludedPatterns"
                    label="Exclude Patterns (Regex)"
                    placeholder="e.g. /node_modules/|/\.git/"
                  />
                  <FormTextField
                    form={form}
                    name="includedExtensions"
                    label="Include Extensions"
                    placeholder="e.g. mp4, mkv, iso"
                  />
                </div>
                <div className="space-y-6">
                  <FormTextField
                    form={form}
                    name="minSizeBytes"
                    label="Minimum File Size (MB)"
                    placeholder="0"
                    type="number"
                    className="font-bold text-center"
                    format={(val) => String(Math.floor((val || 0) / (1024 * 1024)))}
                    parse={(val) => (parseInt(val || "0") * 1024 * 1024)}
                  />
                  <div className="pt-7">
                    <p className="text-xs text-muted leading-relaxed italic">
                      These filters will skip matching files during
                      the recursive cloud scan. Click "Apply Rules" to
                      update all selected remotes below.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}
        </Card>
      </section>

      <section className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-1.5 h-6 bg-accent rounded-full" />
            <h3 className="text-lg font-bold">Target Remotes</h3>
          </div>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="ghost"
              className="rounded-xl font-bold text-[10px] uppercase tracking-widest"
              onPress={() => setIncludedRemotes("all")}
            >
              Select All
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="rounded-xl font-bold text-[10px] uppercase tracking-widest"
              onPress={() => setIncludedRemotes(new Set())}
            >
              Deselect All
            </Button>
          </div>
        </div>

        <ListBox
          aria-label="Target Remotes"
          items={configs}
          selectionMode="multiple"
          selectedKeys={includedRemotes}
          onSelectionChange={setIncludedRemotes}
          className="p-0 gap-3"
        >
          {(config) => (
            <ListBox.Item
              id={config.remote}
              textValue={config.remote}
              className={cn(
                "p-0 rounded-3xl transition-all duration-200 border border-border outline-none cursor-pointer",
                "bg-background/50 hover:bg-default/5",
                "data-[selected=true]:bg-accent/5 data-[selected=true]:border-accent/20",
              )}
            >
              <div className="p-5 flex items-center justify-between gap-4 w-full">
                <div className="flex items-center gap-5 flex-1 min-w-0">
                  <ListBox.ItemIndicator>
                    {({ isSelected }) => (
                      <Checkbox isSelected={isSelected} isReadOnly className="pointer-events-none" />
                    )}
                  </ListBox.ItemIndicator>
                  <div className="min-w-0">
                    <h4 className="font-bold text-base capitalize">
                      {config.remote}
                    </h4>
                    <div className="flex items-center gap-3 mt-1">
                      {config.status === "indexing" ? (
                        <Chip
                          color="accent"
                          size="sm"
                          variant="soft"
                          className="animate-pulse h-5 text-[9px] font-black uppercase"
                        >
                          Indexing...
                        </Chip>
                      ) : config.lastIndexedAt ? (
                        <span className="text-[10px] text-muted flex items-center gap-1.5 whitespace-nowrap">
                          <IconCircleCheckFill className="w-3 h-3 text-success" />
                          Last indexed:{" "}
                          {new Date(
                            config.lastIndexedAt,
                          ).toLocaleString()}
                        </span>
                      ) : (
                        <span className="text-[10px] text-warning font-bold uppercase tracking-tight">
                          Never indexed
                        </span>
                      )}
                      {config.status === "error" && (
                        <Tooltip>
                          <Tooltip.Trigger>
                            <Chip
                              color="danger"
                              size="sm"
                              variant="soft"
                              className="h-5 text-[9px] font-black uppercase"
                            >
                              Error
                            </Chip>
                          </Tooltip.Trigger>
                          <Tooltip.Content className="p-2 text-xs max-w-xs">
                            {config.errorMsg}
                          </Tooltip.Content>
                        </Tooltip>
                      )}
                    </div>
                  </div>
                </div>

                <div className="shrink-0">
                  <Button
                    size="sm"
                    variant={
                      config.status === "indexing"
                        ? "secondary"
                        : "ghost"
                    }
                    className="rounded-xl font-bold h-10 px-5"
                    isDisabled={config.status === "indexing"}
                    isPending={
                      triggerIndex.isPending &&
                      triggerIndex.variables === config.remote
                    }
                    onPress={() => {
                      triggerIndex.mutate(config.remote);
                    }}
                  >
                    <IconArrowsRotateRight
                      className={cn(
                        "w-4 h-4 mr-2",
                        config.status === "indexing" && "animate-spin",
                      )}
                    />
                    {config.lastIndexedAt
                      ? "Rebuild Index"
                      : "Start Indexing"}
                  </Button>
                </div>
              </div>
            </ListBox.Item>
          )}
        </ListBox>
      </section>
    </>
	);
}
