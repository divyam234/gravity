import {
  Button,
  Card,
  ScrollShadow,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { useRemotes } from "../hooks/useRemotes";
import { useEngineActions } from "../hooks/useEngine";
import { FormTextField, FormSwitch, FormSelect } from "../components/ui/FormFields";
import { RemoteSettings } from "../components/dashboard/settings/RemoteSettings";
import type { components } from "../gen/api";

type Settings = components["schemas"]["model.Settings"];
type Remote = components["schemas"]["engine.Remote"];

export const Route = createFileRoute("/settings/uploads")({
  component: UploadSettingsPage,
});

function UploadSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();
  const { data: remotes = [], isLoading: _isLoadingRemotes } = useRemotes();

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
            selector={(state) => [state.canSubmit, state.isSubmitting]}
          >
            {([canSubmit, isSubmitting]) => (
              <Button
                variant="primary"
                onPress={() => form.handleSubmit()}
                isDisabled={!canSubmit}
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
            {/* Auto-Upload */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Automatic Offloading</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <FormSwitch
                    form={form}
                    name="autoUpload"
                    label="Enable Auto-Upload"
                    description="Automatically start uploading to cloud after download completes"
                  />
                  <div className="h-px bg-border" />
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <FormSelect
                      form={form}
                      name="defaultRemote"
                      label="Default Destination"
                      items={remoteOptions}
                    />
                    <FormTextField
                      form={form}
                      name="concurrentUploads"
                      label="Max Concurrent Uploads"
                      type="number"
                      placeholder="1"
                    />
                  </div>
                  <div className="h-px bg-border" />
                  <FormSwitch
                    form={form}
                    name="removeLocal"
                    label="Remove Local Files"
                    description="Delete files from local storage after successful upload"
                  />
                </Card.Content>
              </Card>
            </section>

            {/* Performance */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Transfer Performance</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <FormTextField
                      form={form}
                      name="uploadBandwidth"
                      label="Global Bandwidth Limit"
                      placeholder="0 (Unlimited)"
                      description="Max upload speed (e.g. 500K, 2M)"
                    />
                    <FormTextField
                      form={form}
                      name="chunkSize"
                      label="Upload Chunk Size"
                      placeholder="64M"
                      description="Rclone memory buffer per file (e.g. 64M, 128M)"
                    />
                  </div>
                  <div className="h-px bg-border" />
                  <FormTextField
                    form={form}
                    name="maxRetryAttempts"
                    label="Max Retry Attempts"
                    type="number"
                    placeholder="3"
                  />
                </Card.Content>
              </Card>
            </section>

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
