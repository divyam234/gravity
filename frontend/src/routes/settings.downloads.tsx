import { Button, ScrollShadow } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { useEngineActions } from "../hooks/useEngine";
import {
  DynamicSettings,
  type SettingGroupConfig,
} from "../components/ui/FormFields";
import type { components } from "../gen/api";

type Settings = components["schemas"]["model.Settings"];

export const Route = createFileRoute("/settings/downloads")({
  component: DownloadSettingsPage,
});

function DownloadSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <DownloadSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
    />
  );
}

const downloadSettingsSchema = z.object({
  downloadDir: z.string().min(1, "Download directory is required"),
  maxConcurrentDownloads: z
    .number()
    .min(1, "Must allow at least 1 concurrent download")
    .max(20, "Maximum 20 concurrent downloads allowed")
    .default(3),
  maxDownloadSpeed: z
    .string()
    .regex(/^(\d+[KMGkmg]?)?$/, "Invalid speed format (e.g. 0, 500K, 5M)")
    .default("0"),
  maxUploadSpeed: z
    .string()
    .regex(/^(\d+[KMGkmg]?)?$/, "Invalid speed format (e.g. 0, 500K, 1M)")
    .default("0"),
  preferredEngine: z
    .enum(["aria2", "native"], {
      error: "Please select a valid engine",
    })
    .default("aria2"),
  preferredMagnetEngine: z
    .enum(["aria2", "native"], {
      error: "Please select a valid magnet engine",
    })
    .default("aria2"),
  split: z
    .number()
    .min(1, "Must have at least 1 split")
    .max(16, "Maximum 16 splits allowed per download")
    .default(8),
  autoResume: z.boolean(),
  preAllocateSpace: z.boolean(),
  diskCache: z
    .string()
    .regex(/^\d+[KMGkmg]?$/, "Invalid cache size (e.g. 16M, 64M)")
    .default("16M"),
  minSplitSize: z
    .string()
    .regex(/^\d+[KMGkmg]?$/, "Invalid size format (e.g. 1M, 10M)")
    .default("1M"),
  lowestSpeedLimit: z
    .string()
    .regex(/^\d+[KMGkmg]?$/, "Invalid speed format (e.g. 0, 10K)")
    .default("0"),
  maxConnectionPerServer: z.number().min(1).max(16).default(8),
  userAgent: z.string().optional(),
  connectTimeout: z.number().min(1).default(60),
  maxTries: z.number().min(0).default(0),
  checkCertificate: z.boolean().default(true),
});

function DownloadSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: Settings;
  updateServerSettings: (settings: Settings) => void;
}) {
  const navigate = useNavigate();
  const { changeGlobalOption } = useEngineActions();

  const form = useForm({
    defaultValues: serverSettings.download,
    validators: {
      onChange: downloadSettingsSchema,
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
        download: {
          ...serverSettings.download,
          ...value,
        },
      };

      try {
        await changeGlobalOption.mutateAsync({ body: updated });
        updateServerSettings(updated);
      } catch (err) {
        console.error(err);
      }
    },
  });

  const settingGroups: SettingGroupConfig<
    z.infer<typeof downloadSettingsSchema>
  >[] = [
    {
      id: "general",
      title: "General Options",
      fields: [
        {
          name: "preferredEngine",
          type: "select",
          label: "Preferred Engine",
          options: [
            { value: "aria2", label: "aria2c (External)" },
            { value: "native", label: "Native (Go-based)" },
          ],
        },
        {
          name: "preferredMagnetEngine",
          type: "select",
          label: "Default Magnet Engine",
          options: [
            { value: "aria2", label: "Aria2" },
            { value: "native", label: "Native" },
          ],
        },
        { type: "divider" },
        {
          name: "downloadDir",
          type: "text",
          label: "Default Download Directory",
          placeholder: "/downloads",
          description: "The location on disk where files are saved",
          colSpan: 2,
        },
      ],
    },
    {
      id: "performance",
      title: "Performance",
      fields: [
        {
          name: "maxDownloadSpeed",
          type: "text",
          label: "Max Download Speed",
          placeholder: "0 (Unlimited)",
          description: "Global limit for download bandwidth (e.g. 5M)",
        },
        {
          name: "maxUploadSpeed",
          type: "text",
          label: "Max Upload Speed",
          placeholder: "0 (Unlimited)",
          description: "Global limit for upload bandwidth (e.g. 1M)",
        },
        { type: "divider" },
        {
          name: "maxConcurrentDownloads",
          type: "number",
          label: "Simultaneous Downloads",
          placeholder: "3",
          description: "Max number of active downloads allowed at once",
        },
        {
          name: "split",
          type: "number",
          label: "Max Splits",
          placeholder: "5",
          description: "Number of connections per download (1-16)",
        },
        {
          name: "maxConnectionPerServer",
          type: "number",
          label: "Connections per Server",
          placeholder: "8",
          description: "Max connections per single server (1-16)",
        },
      ],
    },
    {
      id: "network",
      title: "Network & Security",
      fields: [
        {
          name: "userAgent",
          type: "text",
          label: "User Agent",
          placeholder: "Aria2/1.36.0",
        },
        {
          name: "connectTimeout",
          type: "number",
          label: "Connect Timeout (s)",
          placeholder: "60",
        },
        {
          name: "maxTries",
          type: "number",
          label: "Max Retries",
          placeholder: "0 (Unlimited)",
        },
        {
          name: "checkCertificate",
          type: "switch",
          label: "Check Certificate",
          description: "Verify SSL certificates for HTTPS downloads",
        },
      ],
    },
    {
      id: "advanced",
      title: "Advanced Options",
      fields: [
        {
          name: "diskCache",
          type: "text",
          label: "Disk Cache",
          placeholder: "16M",
          description: "Aria2 disk cache size (e.g. 16M, 64M)",
        },
        {
          name: "minSplitSize",
          type: "text",
          label: "Min Split Size",
          placeholder: "1M",
          description: "Min file size to split (e.g. 1M)",
        },
        {
          name: "lowestSpeedLimit",
          type: "text",
          label: "Lowest Speed Limit",
          placeholder: "0",
          description: "Abort download if speed drops below (e.g. 10K)",
        },
        { type: "divider" },
        {
          name: "autoResume",
          type: "switch",
          label: "Auto-Resume Downloads",
          description: "Automatically resume unfinished tasks on startup",
          colSpan: 2,
        },
        {
          name: "preAllocateSpace",
          type: "switch",
          label: "Pre-allocate Disk Space",
          description:
            "Reserve file space before downloading to reduce fragmentation",
          colSpan: 2,
        },
      ],
    },
  ];

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
          <p className="text-xs text-muted">
            Core engine & file storage options
          </p>
        </div>
        <div className="ml-auto flex items-center gap-4">
          <form.Subscribe
            selector={(state) => [
              state.canSubmit,
              state.isSubmitting,
              state.isDirty,
              state.errors,
            ]}
          >
            {([canSubmit, isSubmitting, isDirty, errors]) => (
              <div className="flex items-center gap-4">
                {errors.length > 0 && (
                  <div className="text-xs text-danger font-medium animate-pulse">
                    {errors.length} error(s) detected
                  </div>
                )}
                <Button
                  variant="primary"
                  onPress={() => form.handleSubmit()}
                  isPending={isSubmitting as boolean}
                  className="rounded-xl font-bold"
                >
                  Save Changes
                </Button>
              </div>
            )}
          </form.Subscribe>
        </div>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        <ScrollShadow className="h-full">
          <div className="max-w-4xl mx-auto p-8 space-y-10">
            <DynamicSettings form={form} groups={settingGroups} />
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
