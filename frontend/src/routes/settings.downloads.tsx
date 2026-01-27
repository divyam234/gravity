import { Button, Card, ScrollShadow } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { useEngineActions } from "../hooks/useEngine";
import { FormTextField, FormSwitch, FormSelect } from "../components/ui/FormFields";
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

function DownloadSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: Settings;
  updateServerSettings: (settings: Settings) => void;
}) {
  const navigate = useNavigate();
  const download = serverSettings.download;
  const { changeGlobalOption } = useEngineActions();

  const form = useForm({
    defaultValues: {
      maxConcurrentDownloads: download?.maxConcurrentDownloads || 3,
      downloadDir: download?.downloadDir || "",
      autoResume: !!download?.autoResume,
      preferredEngine: download?.preferredEngine || "aria2",
      lowestSpeedLimit: download?.lowestSpeedLimit || "0",
      diskCache: download?.diskCache || "16M",
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
        download: {
          ...serverSettings.download,
          downloadDir: value.downloadDir,
          maxConcurrentDownloads: Number(value.maxConcurrentDownloads),
          autoResume: value.autoResume,
          preferredEngine: value.preferredEngine as any,
          preferredMagnetEngine: serverSettings.download?.preferredMagnetEngine || "aria2",
          lowestSpeedLimit: value.lowestSpeedLimit,
          diskCache: value.diskCache,
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
          <h2 className="text-2xl font-bold tracking-tight">Downloads</h2>
          <p className="text-xs text-muted">Core engine & file storage options</p>
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
            {/* General */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">General Options</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <FormSelect
                      form={form}
                      name="preferredEngine"
                      label="Preferred Engine"
                      items={[
                        { value: "aria2", label: "aria2c (External)" },
                        { value: "native", label: "Native (Go-based)" },
                      ]}
                    />
                    <FormTextField
                      form={form}
                      name="maxConcurrentDownloads"
                      label="Max Concurrent Tasks"
                      type="number"
                      placeholder="3"
                    />
                  </div>

                  <div className="h-px bg-border" />

                  <FormTextField
                    form={form}
                    name="downloadDir"
                    label="Default Download Directory"
                    placeholder="/downloads"
                    description="The location on disk where files are saved"
                  />

                  <div className="h-px bg-border" />

                  <FormSwitch
                    form={form}
                    name="autoResume"
                    label="Auto Resume"
                    description="Automatically resume unfinished tasks on startup"
                  />
                </Card.Content>
              </Card>
            </section>

            {/* Performance */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Performance</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <FormTextField
                      form={form}
                      name="diskCache"
                      label="Disk Cache"
                      placeholder="16M"
                      description="Aria2 disk cache size (e.g. 16M, 64M)"
                    />
                    <FormTextField
                      form={form}
                      name="lowestSpeedLimit"
                      label="Lowest Speed Limit"
                      placeholder="0"
                      description="Abort download if speed drops below (e.g. 10K)"
                    />
                  </div>
                </Card.Content>
              </Card>
            </section>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}