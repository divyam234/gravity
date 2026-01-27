import {
  Button,
  Card,
  Label,
  ScrollShadow,
  ListBox,
  Chip,
  Checkbox,
} from "@heroui/react";
import type { Selection } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useMemo } from "react";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnifyingGlass from "~icons/gravity-ui/magnifier";
import IconCheck from "~icons/gravity-ui/check";
import IconCloud from "~icons/gravity-ui/cloud";
import IconFunnel from "~icons/gravity-ui/funnel";
import IconGear from "~icons/gravity-ui/gear";
import IconClock from "~icons/gravity-ui/clock";
import IconArrowsRotateRight from "~icons/gravity-ui/arrows-rotate-right";
import { useSettingsStore } from "../store/useSettingsStore";
import { useSearch } from "../hooks/useSearch";
import { useEngineActions } from "../hooks/useEngine";
import { FormSelect, FormTextField } from "../components/ui/FormFields";
import { formatBytes, cn } from "../lib/utils";
import type { components } from "../gen/api";

type RemoteIndexConfig = components["schemas"]["model.RemoteIndexConfig"];
type Settings = components["schemas"]["model.Settings"];

export const Route = createFileRoute("/settings/browser")({
  component: BrowserSettingsPage,
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

const CACHE_MODE_OPTIONS = [
  { value: "off", label: "Off" },
  { value: "minimal", label: "Minimal" },
  { value: "writes", label: "Writes" },
  { value: "full", label: "Full" },
];

function BrowserSettingsPage() {
  const navigate = useNavigate();
  const { serverSettings, updateServerSettings } = useSettingsStore();
  const { configs, triggerIndex, updateConfigs } = useSearch();
  const { changeGlobalOption } = useEngineActions();

  const vfs = serverSettings?.vfs;

  const form = useForm({
    defaultValues: {
      cacheMode: vfs?.cacheMode || "off",
      cacheMaxSize: vfs?.cacheMaxSize || "10G",
      readAhead: vfs?.readAhead || "128M",
      readChunkSize: vfs?.readChunkSize || "128M",
      readChunkSizeLimit: vfs?.readChunkSizeLimit || "off",
      dirCacheTime: vfs?.dirCacheTime || "5m",
      cacheMaxAge: vfs?.cacheMaxAge || "24h",
      writeBack: vfs?.writeBack || "5s",
      pollInterval: vfs?.pollInterval || "1m",
      readChunkStreams: vfs?.readChunkStreams || 0,
    },
    onSubmit: async ({ value }) => {
      if (!serverSettings) return;
      const updated: Settings = {
        ...serverSettings,
        vfs: {
          ...serverSettings.vfs,
          ...value,
          cacheMode: value.cacheMode as NonNullable<Settings["vfs"]>["cacheMode"],
        },
      };

      try {
        await changeGlobalOption.mutateAsync({ body: updated });
        updateServerSettings(updated);
      } catch (err) {
        // Error handled by mutation
      }
    },
  });

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  if (!vfs) {
    return <div className="p-8">VFS settings not found.</div>;
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
          <h2 className="text-2xl font-bold tracking-tight">Browser</h2>
          <p className="text-xs text-muted">
            File browsing, VFS cache & search indexing
          </p>
        </div>
        <div className="ml-auto">
          <form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting]}>
            {([canSubmit, isSubmitting]) => (
              <Button
                variant="primary"
                onPress={() => form.handleSubmit()}
                isDisabled={!canSubmit}
                isPending={isSubmitting as boolean}
                className="rounded-xl font-bold"
              >
                Save VFS Changes
              </Button>
            )}
          </form.Subscribe>
        </div>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        <ScrollShadow className="h-full">
          <div className="max-w-4xl mx-auto p-8 space-y-10">
            {/* VFS Options */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-primary rounded-full" />
                <h3 className="text-lg font-bold">VFS Cache (Streaming)</h3>
              </div>

              <div className="grid gap-4">
                {/* VFS Cache Mode Card */}
                <Card className="p-6 bg-background/50 border-border">
                  <div className="flex items-center justify-between">
                    <div className="flex-1 mr-8">
                      <Label className="text-sm font-bold">
                        VFS Cache Mode
                      </Label>
                      <p className="text-xs text-muted mt-0.5">
                        "full" is required for seeking in videos and random
                        access.
                      </p>
                    </div>

                    <FormSelect
                      form={form}
                      name="cacheMode"
                      className="w-48"
                      items={CACHE_MODE_OPTIONS}
                    />
                  </div>
                </Card>

                {/* Sizes Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Card className="p-6 bg-background/50 border-border">
                    <FormTextField
                      form={form}
                      name="cacheMaxSize"
                      label="Max Cache Size"
                      description="Max disk space for cache (e.g. 10G, 100G)"
                    />
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <FormTextField
                      form={form}
                      name="readAhead"
                      label="Read Ahead"
                      description='Bytes to read ahead in "full" mode (e.g. 128M)'
                    />
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <FormTextField
                      form={form}
                      name="readChunkSize"
                      label="Read Chunk Size"
                      description="Initial chunk size for reads (e.g. 128M)"
                    />
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <FormTextField
                      form={form}
                      name="readChunkSizeLimit"
                      label="Chunk Size Limit"
                      description='Max doubled chunk size (e.g. 512M, or "off")'
                    />
                  </Card>
                </div>

                {/* Durations Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Card className="p-6 bg-background/50 border-border">
                    <FormTextField
                      form={form}
                      name="dirCacheTime"
                      label="Dir Cache Time"
                      description="How long to cache directory listings (e.g. 5m)"
                    />
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <FormTextField
                      form={form}
                      name="cacheMaxAge"
                      label="Max Cache Age"
                      description="Max age of cached files (e.g. 1h, 24h)"
                    />
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <FormTextField
                      form={form}
                      name="writeBack"
                      label="Write Back Delay"
                      description="Delay before uploading changed files (e.g. 5s)"
                    />
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <FormTextField
                      form={form}
                      name="pollInterval"
                      label="Poll Interval"
                      description="How often to poll for changes (e.g. 1m)"
                    />
                  </Card>
                </div>

                {/* Others */}
                <Card className="p-6 bg-background/50 border-border">
                  <div className="flex items-center justify-between">
                    <div className="flex-1 mr-8">
                      <Label className="text-sm font-bold">
                        Read Chunk Streams
                      </Label>
                      <p className="text-xs text-muted mt-0.5">
                        Number of parallel streams to use for reading chunks (0
                        to disable).
                      </p>
                    </div>

                    <FormTextField
                      form={form}
                      name="readChunkStreams"
                      type="number"
                      className="w-48"
                    />
                  </div>
                </Card>
              </div>
            </section>

            {/* Search Indexing */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Search Indexing</h3>
              </div>

              {configs.length === 0 ? (
                <Card className="p-8 bg-background/50 border-border border-dashed flex flex-col items-center justify-center text-center rounded-2xl">
                  <div className="w-16 h-16 bg-default/10 rounded-full flex items-center justify-center mb-4">
                    <IconMagnifyingGlass className="w-8 h-8 text-muted" />
                  </div>
                  <h4 className="font-bold text-lg mb-2">
                    No remotes available
                  </h4>
                  <p className="text-sm text-muted mb-6">
                    Configure cloud remotes first to enable indexing.
                  </p>
                  <Button
                    variant="primary"
                    onPress={() => navigate({ to: "/settings/uploads" })}
                    className="rounded-xl font-bold"
                  >
                    Go to Uploads
                  </Button>
                </Card>
              ) : (
                <SearchSettingsLayout
                  configs={configs}
                  updateConfigs={updateConfigs}
                  triggerIndex={triggerIndex}
                />
              )}
            </section>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}

// Components from SearchSettingsPage
interface SearchSettingsLayoutProps {
  configs: RemoteIndexConfig[];
  updateConfigs: ReturnType<typeof useSearch>['updateConfigs'];
  triggerIndex: ReturnType<typeof useSearch>['triggerIndex'];
}

function SearchSettingsLayout({
  configs,
  updateConfigs,
  triggerIndex,
}: SearchSettingsLayoutProps) {
  const [selectedRemotes, setSelectedRemotes] = useState<Selection>(new Set());

  const selectedConfigs = useMemo(() => {
    if (selectedRemotes === "all") return configs;
    const selectedSet = selectedRemotes as Set<string>;
    return configs.filter((c) => !!c.remote && selectedSet.has(c.remote));
  }, [configs, selectedRemotes]);

  return (
    <div className="space-y-10">
        {/* Remotes Section */}
        <section>
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <span className="text-xs font-medium text-muted bg-default/10 px-2 py-0.5 rounded-full">
                {configs.length} Remotes
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
                  id={config.remote || ""}
                  textValue={config.remote}
                  className={cn(
                    "w-full text-left transition-all duration-200 border rounded-2xl overflow-hidden outline-none group bg-background/50 border-border p-0",
                    "data-[selected=true]:border-accent/30 data-[selected=true]:bg-accent/5",
                    "data-[hover=true]:bg-default/10 data-[hover=true]:border-border/60",
                  )}
                >
                  {({ isSelected }: { isSelected: boolean }) => (
                    <RemoteCard
                      config={config}
                      isSelected={isSelected}
                      onIndex={() => triggerIndex.mutate({ params: { path: { remote: config.remote || "" } } })}
                      isIndexing={
                        triggerIndex.isPending &&
                        triggerIndex.variables?.params?.path?.remote === config.remote
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
  );
}

interface RemoteCardProps {
    config: RemoteIndexConfig;
    isSelected: boolean;
    onIndex: () => void;
    isIndexing: boolean;
}

function RemoteCard({ config, isSelected, onIndex, isIndexing }: RemoteCardProps) {
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
            {formatBytes(config.minSizeBytes || 0)}
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
  selectedConfigs: RemoteIndexConfig[];
  updateConfigs: ReturnType<typeof useSearch>['updateConfigs'];
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
      const batch: Record<string, components["schemas"]["api.UpdateConfigRequest"]> = {};

      selectedConfigs.forEach((config) => {
        if (config.remote) {
            batch[config.remote] = {
                interval: value.interval,
                excludedPatterns: value.excludedPatterns,
                includedExtensions: value.includedExtensions,
                minSizeBytes: value.minSizeBytes,
            };
        }
      });

      await updateConfigs.mutateAsync({ body: { configs: batch } });
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
            format={(val) => String(Math.floor((Number(val) || 0) / (1024 * 1024)))}
            parse={(val) => Number(val || "0") * 1024 * 1024}
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
              isPending={isSubmitting as boolean}
            >
              Save Configuration
            </Button>
          )}
        </form.Subscribe>
      </div>
    </div>
  );
}
