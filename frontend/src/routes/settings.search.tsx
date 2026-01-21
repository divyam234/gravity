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
import IconCheck from "~icons/gravity-ui/check";
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
            <Card className="p-12 bg-background/50 border-border border-dashed max-w-md rounded-2xl shadow-none">
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
            className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 bg-transparent p-0"
          >
            {(config) => {
              return (
                <ListBox.Item
                  id={config.remote}
                  textValue={config.remote}
                  className={cn(
                    "w-full text-left transition-all duration-200 border rounded-2xl overflow-hidden outline-none group bg-background/50 border-border p-0",
                    "data-[selected=true]:border-accent/30 data-[selected=true]:bg-accent/5",
                    "data-[hovered=true]:bg-default/10 data-[hovered=true]:border-border/60",
                  )}
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
              <Chip
                size="sm"
                color="accent"
                variant="soft"
                className="h-5 px-2 text-[10px] font-black uppercase tracking-widest"
              >
                {selectedConfigs.length} Remote
                {selectedConfigs.length !== 1 ? "s" : ""}
              </Chip>
            )}
          </div>

          {selectedConfigs.length > 0 ? (
            <Card className="bg-background/50 border-border overflow-hidden rounded-2xl">
              <Card.Content className="p-8">
                <SearchSettingsForm
                  key={selectedConfigs
                    .map((c) => c.remote)
                    .sort()
                    .join(",")}
                  selectedConfigs={selectedConfigs}
                  updateConfigs={updateConfigs}
                />
              </Card.Content>
            </Card>
          ) : (
            <Card className="bg-background/50 border-border border-dashed p-12 flex flex-col items-center justify-center text-center rounded-2xl opacity-60">
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
    <div className="p-5 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-4 min-w-0">
          <div
            className={cn(
              "w-12 h-12 rounded-2xl flex items-center justify-center shrink-0 transition-all duration-200",
              isSelected
                ? "bg-accent/10 text-accent"
                : "bg-default/10 text-muted group-hover:bg-accent/10 group-hover:text-accent",
            )}
          >
            <IconCloud className="w-5 h-5" />
          </div>
          <div className="min-w-0">
            <h4 className="font-bold text-base truncate text-foreground leading-tight group-hover:text-accent transition-colors">
              {config.remote}
            </h4>
            <div className="flex items-center gap-2 mt-1">
              {config.status === "indexing" ? (
                <div className="flex items-center gap-1.5 text-[10px] text-accent font-black uppercase tracking-widest">
                  <span className="w-1.5 h-1.5 rounded-full bg-accent animate-pulse" />
                  Indexing
                </div>
              ) : config.lastIndexedAt ? (
                <Chip
                  size="sm"
                  variant="soft"
                  color="success"
                  className="h-4 px-1.5 text-[10px] font-black uppercase tracking-widest"
                >
                  <IconCheck className="w-3 h-3 mr-1" />
                  Ready
                </Chip>
              ) : (
                <div className="text-[10px] text-muted font-bold uppercase tracking-widest opacity-60">
                  Idle
                </div>
              )}
            </div>
          </div>
        </div>

        <Checkbox isSelected={isSelected} className="pointer-events-none" />
      </div>

      {/* Badges - Compact */}
      <div className="flex flex-wrap gap-1.5">
        <Chip
          size="sm"
          variant="soft"
          className="h-6 px-2 text-[10px] font-bold border-none bg-default/10"
        >
          <IconClock className="w-3 h-3 mr-1.5 opacity-60" />
          {intervalLabel}
        </Chip>

        {(config.minSizeBytes || 0) > 0 && (
          <Chip
            size="sm"
            variant="soft"
            className="h-6 px-2 text-[10px] font-bold border-none bg-default/10"
          >
            <IconFunnel className="w-3 h-3 mr-1.5 opacity-60" />
            {formatBytes(config.minSizeBytes)}
          </Chip>
        )}

        {config.includedExtensions && (
          <Chip
            size="sm"
            variant="soft"
            className="h-6 px-2 text-[10px] font-bold border-none bg-default/10 max-w-[120px] truncate"
          >
            {config.includedExtensions}
          </Chip>
        )}
      </div>

      {/* Footer */}
      <div className="pt-3 flex items-center justify-between border-t border-border/10">
        <div className="text-[10px] text-muted font-medium opacity-60 truncate pr-2">
          {config.lastIndexedAt
            ? `Sync: ${new Date(config.lastIndexedAt).toLocaleDateString()}`
            : "Never synced"}
        </div>
        <Button
          size="sm"
          variant="ghost"
          className="h-7 min-w-0 px-3 gap-2 text-[10px] font-black uppercase tracking-widest rounded-xl hover:bg-accent/10 hover:text-accent transition-all"
          isDisabled={config.status === "indexing"}
          onPress={(e) => {
            e.continuePropagation();
            onIndex();
          }}
        >
          <IconArrowsRotateRight
            className={cn(
              "w-3 h-3",
              (config.status === "indexing" || isIndexing) && "animate-spin",
            )}
          />
          {config.lastIndexedAt ? "Sync" : "Start"}
        </Button>
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
      <div className="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-6">
        {/* Row 1: Frequency & Size */}
        <div className="space-y-3">
          <FormSelect
            form={form}
            name="interval"
            label={
              <span className="flex items-center gap-2 text-xs font-bold text-foreground/80">
                Update Frequency
                {isMixed.interval && (
                  <Chip
                    size="sm"
                    color="warning"
                    variant="soft"
                    className="h-3.5 text-[7px] font-black uppercase px-1"
                  >
                    Mixed
                  </Chip>
                )}
              </span>
            }
            items={INTERVAL_OPTIONS}
          />
          {isMixed.interval && (
            <p className="text-[9px] text-warning px-1 font-bold uppercase tracking-wider opacity-80">
              Mixed frequencies. Overwriting...
            </p>
          )}
        </div>

        <div className="space-y-3">
          <FormTextField
            form={form}
            name="minSizeBytes"
            label={
              <span className="flex items-center gap-2 text-xs font-bold text-foreground/80">
                Minimum File Size
                {isMixed.size && (
                  <Chip
                    size="sm"
                    color="warning"
                    variant="soft"
                    className="h-3.5 text-[7px] font-black uppercase px-1"
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
              <span className="text-[9px] text-muted font-black uppercase px-2">
                MB
              </span>
            }
          />
        </div>

        {/* Row 2: Patterns */}
        <div className="md:col-span-2 h-px bg-border/20 my-2" />

        <div className="md:col-span-1">
          <FormTextField
            form={form}
            name="excludedPatterns"
            label={
              <span className="flex items-center gap-2 text-xs font-bold text-foreground/80">
                Exclude Patterns (Regex)
                {isMixed.patterns && (
                  <Chip
                    size="sm"
                    color="warning"
                    variant="soft"
                    className="h-3.5 text-[7px] font-black uppercase px-1"
                  >
                    Mixed
                  </Chip>
                )}
              </span>
            }
            placeholder={
              isMixed.patterns ? "Mixed values" : "e.g. /node_modules/"
            }
          />
        </div>

        <div className="md:col-span-1">
          <FormTextField
            form={form}
            name="includedExtensions"
            label={
              <span className="flex items-center gap-2 text-xs font-bold text-foreground/80">
                Include Extensions
                {isMixed.extensions && (
                  <Chip
                    size="sm"
                    color="warning"
                    variant="soft"
                    className="h-3.5 text-[7px] font-black uppercase px-1"
                  >
                    Mixed
                  </Chip>
                )}
              </span>
            }
            placeholder={isMixed.extensions ? "Mixed values" : "e.g. mp4, mkv"}
          />
        </div>
      </div>

      <div className="pt-2 flex justify-end">
        <form.Subscribe
          selector={(state) => [state.canSubmit, state.isSubmitting]}
        >
          {([canSubmit, isSubmitting]) => (
            <Button
              variant="primary"
              className="font-black text-xs rounded-xl h-10 px-8 uppercase tracking-widest"
              onPress={() => form.handleSubmit()}
              isDisabled={!canSubmit}
              isPending={isSubmitting}
            >
              Save Configuration
            </Button>
          )}
        </form.Subscribe>
      </div>
    </div>
  );
}
