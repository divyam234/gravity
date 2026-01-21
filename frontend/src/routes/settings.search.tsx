import {
  Button,
  Card,
  Chip,
  Checkbox,
  cn,
  ScrollShadow,
  Spinner,
  ListBox,
} from "@heroui/react";
import type { Selection } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useMemo } from "react";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnifyingGlass from "~icons/gravity-ui/magnifier";
import IconArrowsRotateRight from "~icons/gravity-ui/arrows-rotate-right";
import IconCircleCheckFill from "~icons/gravity-ui/circle-check-fill";
import IconCloud from "~icons/gravity-ui/cloud";
import IconFunnel from "~icons/gravity-ui/funnel";
import IconGear from "~icons/gravity-ui/gear";
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
          <p className="text-xs text-muted">
            Global indexing rules & remote management
          </p>
        </div>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        {configs.length === 0 ? (
          <div className="flex h-full items-center justify-center p-8 text-center">
            <Card className="p-12 bg-surface border-border border-dashed max-w-md rounded-[2.5rem] shadow-none">
              <div className="w-20 h-20 bg-default/10 rounded-full flex items-center justify-center mb-6 mx-auto">
                <IconMagnifyingGlass className="w-10 h-10 text-muted" />
              </div>
              <h4 className="font-bold text-xl mb-2 text-foreground">
                No remotes available
              </h4>
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
  const [selectedRemotes, setSelectedRemotes] = useState<Selection>(new Set());

  const selectedConfigs = useMemo(() => {
    if (selectedRemotes === "all") return configs;
    return configs.filter((c) => selectedRemotes.has(c.remote));
  }, [configs, selectedRemotes]);

  return (
    <ScrollShadow className="h-full">
      <div className="max-w-4xl mx-auto p-8 space-y-10">
        {/* Remotes Section */}
        <section>
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <div className="w-1.5 h-6 bg-accent rounded-full" />
              <h3 className="text-lg font-bold">Target Remotes</h3>
              <span className="text-xs font-medium text-muted bg-default/10 px-2 py-0.5 rounded-full">
                {configs.length}
              </span>
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="ghost"
                className="rounded-xl font-bold text-xs"
                onPress={() => setSelectedRemotes("all")}
              >
                Select All
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="rounded-xl font-bold text-xs text-danger"
                isDisabled={
                  selectedRemotes !== "all" && selectedRemotes.size === 0
                }
                onPress={() => setSelectedRemotes(new Set())}
              >
                Clear
              </Button>
            </div>
          </div>

          <ListBox
            aria-label="Target Remotes"
            items={configs}
            selectionMode="multiple"
            selectedKeys={selectedRemotes}
            onSelectionChange={setSelectedRemotes}
            className="grid grid-cols-1 md:grid-cols-2 gap-4 bg-transparent p-0"
          >
            {(config) => {
              return (
                <ListBox.Item
                  id={config.remote}
                  textValue={config.remote}
                  className="p-0 rounded-[2.2rem] outline-none"
                >
                  {({ isSelected }) => (
                    <RemoteCard
                      config={config}
                      isSelected={isSelected}
                      onIndex={() => triggerIndex.mutate(config.remote)}
                      isIndexing={
                        triggerIndex.isPending &&
                        triggerIndex.variables === config.remote
                      }
                    />
                  )}
                </ListBox.Item>
              );
            }}
          </ListBox>
        </section>

        {/* Configuration Section */}
        <section className="animate-in slide-in-from-bottom-4 duration-500">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-1.5 h-6 bg-accent rounded-full" />
            <h3 className="text-lg font-bold">Configuration</h3>
            {selectedConfigs.length > 0 && (
              <span className="text-xs font-medium text-muted bg-accent/10 text-accent px-2 py-0.5 rounded-full">
                Editing {selectedConfigs.length} Remote
                {selectedConfigs.length !== 1 ? "s" : ""}
              </span>
            )}
          </div>

          {selectedConfigs.length > 0 ? (
            <Card className="bg-background/50 border-border overflow-hidden">
              <div className="p-8">
                <SearchSettingsForm
                  key={selectedConfigs
                    .map((c) => c.remote)
                    .sort()
                    .join(",")}
                  selectedConfigs={selectedConfigs}
                  updateConfigs={updateConfigs}
                />
              </div>
            </Card>
          ) : (
            <Card className="bg-background/50 border-border border-dashed p-12 flex flex-col items-center justify-center text-center opacity-60">
              <div className="w-16 h-16 bg-default/10 rounded-2xl flex items-center justify-center mb-4">
                <IconGear className="w-8 h-8 text-muted" />
              </div>
              <p className="font-bold text-muted">No Remotes Selected</p>
              <p className="text-xs text-muted mt-2 max-w-xs">
                Select one or more remotes from the list above to configure
                their indexing rules.
              </p>
            </Card>
          )}
        </section>
      </div>
    </ScrollShadow>
  );
}

function RemoteCard({ config, isSelected, onIndex, isIndexing }: any) {
  const intervalLabel =
    INTERVAL_OPTIONS.find((o) => o.value === config.autoIndexIntervalMin)
      ?.label || "Custom";

  return (
    <div
      className={cn(
        "w-full text-left transition-all duration-200 border-2 overflow-hidden",
        isSelected
          ? "border-accent bg-accent/5"
          : "border-border/40 bg-background/50 hover:bg-background/20",
      )}
    >
      <div className="p-5 space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3 overflow-hidden">
            <div
              className={cn(
                "w-10 h-10 rounded-xl flex items-center justify-center shrink-0 transition-colors",
                isSelected
                  ? "bg-accent text-accent-foreground"
                  : "bg-default/10 text-muted",
              )}
            >
              <IconCloud className="w-5 h-5" />
            </div>
            <div className="min-w-0">
              <h4 className="font-bold text-sm truncate text-foreground">
                {config.remote}
              </h4>
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
                  <span className="text-[10px] text-success font-bold uppercase tracking-wide flex items-center gap-1">
                    <IconCircleCheckFill className="w-3 h-3" /> Indexed
                  </span>
                ) : (
                  <span className="text-[10px] text-muted font-bold uppercase tracking-wide opacity-70">
                    Idle
                  </span>
                )}
              </div>
            </div>
          </div>

          <Checkbox isSelected={isSelected} className="pointer-events-none" />
        </div>

        {/* Badges */}
        <div className="flex flex-wrap gap-2">
          <Chip
            size="sm"
            variant="soft"
            color="accent"
            className="h-6 gap-1.5 pl-2 rounded-lg bg-accent/10 border border-accent/10"
          >
            <IconClock className="w-3 h-3" />
            <span className="text-[9px] font-bold uppercase tracking-wide">
              {intervalLabel}
            </span>
          </Chip>

          {(config.minSizeBytes || 0) > 0 && (
            <Chip
              size="sm"
              variant="soft"
              color="warning"
              className="h-6 gap-1.5 pl-2 rounded-lg bg-warning/10 border border-warning/10"
            >
              <IconFunnel className="w-3 h-3" />
              <span className="text-[9px] font-bold uppercase tracking-wide">
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
              className="h-6 gap-1.5 pl-2 rounded-lg bg-success/10 border border-success/10"
            >
              <span className="text-[9px] font-bold uppercase tracking-wide">
                Ext: {config.includedExtensions}
              </span>
            </Chip>
          )}
        </div>

        {/* Footer */}
        <div className="pt-3 flex items-center justify-between border-t border-border/10">
          <div className="text-[10px] text-muted font-medium opacity-60 truncate pr-2">
            {config.lastIndexedAt
              ? `Last: ${new Date(config.lastIndexedAt).toLocaleDateString()}`
              : "Never indexed"}
          </div>
          <Button
            size="sm"
            variant="ghost"
            className="h-7 text-[10px] font-bold uppercase tracking-wider rounded-lg text-accent hover:bg-accent/10"
            isDisabled={config.status === "indexing"}
            onPress={(e) => {
              e.continuePropagation();
              onIndex();
            }}
          >
            <IconArrowsRotateRight
              className={cn(
                "w-3 h-3 mr-1.5",
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
}: {
  selectedConfigs: any[];
  updateConfigs: any;
}) {
  const commonValues = useMemo(() => {
    if (selectedConfigs.length === 0) return null;

    const first = selectedConfigs[0];
    const common = {
      interval: first.autoIndexIntervalMin,
      excludedPatterns: first.excludedPatterns || "",
      includedExtensions: first.includedExtensions || "",
      minSizeBytes: first.minSizeBytes || 0,
    };

    for (let i = 1; i < selectedConfigs.length; i++) {
      const c = selectedConfigs[i];
      if (c.autoIndexIntervalMin !== common.interval) common.interval = -1;
      if ((c.excludedPatterns || "") !== common.excludedPatterns)
        common.excludedPatterns = "__mixed__";
      if ((c.includedExtensions || "") !== common.includedExtensions)
        common.includedExtensions = "__mixed__";
      if ((c.minSizeBytes || 0) !== common.minSizeBytes)
        common.minSizeBytes = -1;
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

  const isMixed = {
    interval: commonValues?.interval === -1,
    patterns: commonValues?.excludedPatterns === "__mixed__",
    extensions: commonValues?.includedExtensions === "__mixed__",
    size: commonValues?.minSizeBytes === -1,
  };

  return (
    <div className="space-y-8">
      {/* Interval */}
      <div className="space-y-4">
        <FormSelect
          form={form}
          name="interval"
          label={
            <span className="flex items-center gap-2">
              Update Frequency
              {isMixed.interval && (
                <Chip
                  size="sm"
                  color="warning"
                  variant="soft"
                  className="h-4 text-[8px] font-black uppercase px-1"
                >
                  Mixed
                </Chip>
              )}
            </span>
          }
          items={INTERVAL_OPTIONS}
        />
        {isMixed.interval && (
          <p className="text-[10px] text-warning px-1 font-bold uppercase tracking-wider opacity-80">
            Mixed frequencies. Saving will overwrite them.
          </p>
        )}
      </div>

      <div className="h-px bg-border/40" />

      {/* Filters */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="md:col-span-2 space-y-6">
          <FormTextField
            form={form}
            name="excludedPatterns"
            label={
              <span className="flex items-center gap-2">
                Exclude Patterns (Regex)
                {isMixed.patterns && (
                  <Chip
                    size="sm"
                    color="warning"
                    variant="soft"
                    className="h-4 text-[8px] font-black uppercase px-1"
                  >
                    Mixed
                  </Chip>
                )}
              </span>
            }
            placeholder={
              isMixed.patterns
                ? "Mixed values (leave empty to clear)"
                : "e.g. /node_modules/"
            }
          />

          <FormTextField
            form={form}
            name="includedExtensions"
            label={
              <span className="flex items-center gap-2">
                Include Extensions
                {isMixed.extensions && (
                  <Chip
                    size="sm"
                    color="warning"
                    variant="soft"
                    className="h-4 text-[8px] font-black uppercase px-1"
                  >
                    Mixed
                  </Chip>
                )}
              </span>
            }
            placeholder={
              isMixed.extensions
                ? "Mixed values (leave empty to clear)"
                : "e.g. mp4, mkv"
            }
          />
        </div>

        <div className="md:col-span-1">
          <FormTextField
            form={form}
            name="minSizeBytes"
            label={
              <span className="flex items-center gap-2">
                Minimum File Size
                {isMixed.size && (
                  <Chip
                    size="sm"
                    color="warning"
                    variant="soft"
                    className="h-4 text-[8px] font-black uppercase px-1"
                  >
                    Mixed
                  </Chip>
                )}
              </span>
            }
            type="number"
            placeholder={isMixed.size ? "Mixed" : "0"}
            format={(val) => String(Math.floor((val || 0) / (1024 * 1024)))}
            parse={(val) => parseInt(val || "0") * 1024 * 1024}
            endContent={
              <span className="text-[10px] text-muted font-bold uppercase px-2">
                MB
              </span>
            }
          />
        </div>
      </div>

      <div className="pt-4">
        <form.Subscribe
          selector={(state) => [state.canSubmit, state.isSubmitting]}
        >
          {([canSubmit, isSubmitting]) => (
            <div className="flex justify-end">
              <Button
                variant="primary"
                className="font-bold rounded-xl h-11 px-8"
                onPress={() => form.handleSubmit()}
                isDisabled={!canSubmit}
                isPending={isSubmitting}
              >
                Save Changes
              </Button>
            </div>
          )}
        </form.Subscribe>
      </div>
    </div>
  );
}
