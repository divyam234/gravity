import {
  Button,
  ScrollShadow,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { useRemotes } from "../hooks/useRemotes";
import { useEngineActions } from "../hooks/useEngine";
import { DynamicSettings, type SettingGroupConfig } from "../components/ui/FormFields";
import { RemoteSettings } from "../components/dashboard/settings/RemoteSettings";
import type { components } from "../gen/api";

type Settings = components["schemas"]["model.Settings"];
type Remote = components["schemas"]["engine.Remote"];

export const Route = createFileRoute("/settings/uploads")({
  component: UploadSettingsPage,
});

function UploadSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();
  const { data: remotes = [] } = useRemotes();

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <UploadSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
      remotes={remotes}
    />
  );
}

const uploadSettingsSchema = z.object({
  autoUpload: z.boolean(),
  defaultRemote: z.string().optional(),
  removeLocal: z.boolean(),
  concurrentUploads: z.number()
    .min(1, "At least 1 concurrent upload task is required")
    .max(10, "Maximum 10 concurrent uploads allowed"),
  uploadBandwidth: z.string()
    .regex(/^\d+[KMGkmg]?$/, "Invalid speed format (e.g. 0, 500K, 5M)"),
  maxRetryAttempts: z.number()
    .min(0, "Retry attempts cannot be negative")
    .max(10, "Maximum 10 retry attempts allowed"),
  chunkSize: z.string()
    .regex(/^\d+[KMGkmg]?$/, "Invalid size format (e.g. 64M, 128M)"),
});

function UploadSettingsForm({
  serverSettings,
  updateServerSettings,
  remotes,
}: {
  serverSettings: Settings;
  updateServerSettings: (settings: Settings) => void;
  remotes: Remote[];
}) {
  const navigate = useNavigate();
  const upload = serverSettings.upload;
  const { changeGlobalOption } = useEngineActions();

  const remoteOptions = [
    { value: "", label: "None (Disabled)" },
    ...remotes.map((r) => ({
      value: `${r.name}:`,
      label: r.name || "unknown",
    })),
  ];

  const form = useForm({
    defaultValues: {
      autoUpload: !!upload?.autoUpload,
      defaultRemote: upload?.defaultRemote || "",
      removeLocal: !!upload?.removeLocal,
      concurrentUploads: Number(upload?.concurrentUploads || 1),
      uploadBandwidth: upload?.uploadBandwidth || "0",
      maxRetryAttempts: Number(upload?.maxRetryAttempts || 3),
      chunkSize: upload?.chunkSize || "64M",
    },
    validators: {
      onChange: uploadSettingsSchema as any,
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
        upload: {
          ...serverSettings.upload,
          autoUpload: value.autoUpload,
          defaultRemote: value.defaultRemote,
          removeLocal: value.removeLocal,
          concurrentUploads: Number(value.concurrentUploads),
          uploadBandwidth: value.uploadBandwidth,
          maxRetryAttempts: Number(value.maxRetryAttempts),
          chunkSize: value.chunkSize,
        },
      };

      try {
        await changeGlobalOption.mutateAsync({ body: updated as any });
        updateServerSettings(updated);
      } catch (err) {
        // Error toast handled by mutation
      }
    },
  });

  const settingGroups: SettingGroupConfig<z.infer<typeof uploadSettingsSchema>>[] = [
    {
        id: "auto-upload",
        title: "Automatic Offloading",
        fields: [
            {
                name: "autoUpload",
                type: "switch",
                label: "Enable Auto-Upload",
                description: "Automatically start uploading to cloud after download completes",
                colSpan: 2,
            },
            { type: "divider" },
            {
                name: "defaultRemote",
                type: "select",
                label: "Default Destination",
                options: remoteOptions,
                description: "Cloud storage remote to upload files to",
            },
            {
                name: "concurrentUploads",
                type: "number",
                label: "Max Concurrent Uploads",
                placeholder: "1",
                description: "Simultaneous upload tasks allowed",
            },
            { type: "divider" },
            {
                name: "removeLocal",
                type: "switch",
                label: "Remove Local Files",
                description: "Delete files from local storage after successful upload",
                colSpan: 2,
            }
        ]
    },
    {
        id: "performance",
        title: "Transfer Performance",
        fields: [
            {
                name: "uploadBandwidth",
                type: "text",
                label: "Global Bandwidth Limit",
                placeholder: "0 (Unlimited)",
                description: "Max upload speed (e.g. 500K, 2M)",
            },
            {
                name: "chunkSize",
                type: "text",
                label: "Upload Chunk Size",
                placeholder: "64M",
                description: "Rclone memory buffer per file (e.g. 64M, 128M)",
            },
            { type: "divider" },
            {
                name: "maxRetryAttempts",
                type: "number",
                label: "Max Retry Attempts",
                placeholder: "3",
                description: "Number of retries before failing an upload task",
                colSpan: 2,
            }
        ]
    }
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
          <h2 className="text-2xl font-bold tracking-tight">Uploads</h2>
          <p className="text-xs text-muted">Cloud offloading & Rclone options</p>
        </div>
        <div className="ml-auto">
          <form.Subscribe
            selector={(state) => [state.canSubmit, state.isSubmitting, state.isDirty]}
          >
            {([canSubmit, isSubmitting, isDirty]) => (
              <Button
                variant="primary"
                onPress={() => form.handleSubmit()}
                isDisabled={!canSubmit || !isDirty}
                isPending={isSubmitting as boolean}
                className="rounded-xl font-bold"
              >
                Save Changes
              </Button>
            )}
          </form.Subscribe>
        </div>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        <ScrollShadow className="h-full">
          <div className="max-w-4xl mx-auto p-8 space-y-10">
            <DynamicSettings form={form} groups={settingGroups} />

            {/* Remotes Management */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Cloud Connections</h3>
              </div>
              <RemoteSettings />
            </section>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
