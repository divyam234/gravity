import { Button, Card, ScrollShadow } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconFolder from "~icons/gravity-ui/folder";
import { useSettingsStore } from "../store/useSettingsStore";
import { api } from "../lib/api";
import { FormTextField, FormSwitch, FormSelect } from "../components/ui/FormFields";
import { toast } from "sonner";

export const Route = createFileRoute("/settings/downloads")({
  component: DownloadsSettingsPage,
});

function DownloadsSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <DownloadsSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
    />
  );
}

function DownloadsSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: any;
  updateServerSettings: any;
}) {
  const navigate = useNavigate();
  const { download } = serverSettings;

  const form = useForm({
    defaultValues: {
      downloadDir: download.downloadDir,
      maxConcurrentDownloads: download.maxConcurrentDownloads,
      maxDownloadSpeed: download.maxDownloadSpeed,
      maxUploadSpeed: download.maxUploadSpeed,
      preferredEngine: download.preferredEngine,
      preferredMagnetEngine: download.preferredMagnetEngine,
      split: download.split,
      autoResume: download.autoResume,
      preAllocateSpace: download.preAllocateSpace,
      diskCache: download.diskCache,
      minSplitSize: download.minSplitSize,
    },
    onSubmit: async ({ value }) => {
      const updated = {
        ...serverSettings,
        download: {
          ...serverSettings.download,
          downloadDir: value.downloadDir,
          maxConcurrentDownloads: Number(value.maxConcurrentDownloads),
          maxDownloadSpeed: value.maxDownloadSpeed,
          maxUploadSpeed: value.maxUploadSpeed,
          preferredEngine: value.preferredEngine,
          preferredMagnetEngine: value.preferredMagnetEngine,
          split: Number(value.split),
          autoResume: value.autoResume,
          preAllocateSpace: value.preAllocateSpace,
          diskCache: value.diskCache,
          minSplitSize: value.minSplitSize,
        },
      };

      try {
        await api.updateSettings(updated);
        updateServerSettings(updated);
        toast.success("Settings saved successfully");
      } catch (err) {
        console.error(err);
        toast.error("Failed to save settings");
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
          <p className="text-xs text-muted">
            Speed, queue, storage & automation
          </p>
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
                        isPending={isSubmitting}
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
            {/* Engine Preferences */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Engine Preferences</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <FormSelect
                            form={form}
                            name="preferredEngine"
                            label="Default Download Engine"
                            items={[{value: "aria2", label: "Aria2 (Recommended)"}, {value: "native", label: "Native (Experimental)"}]}
                        />
                        <FormSelect
                            form={form}
                            name="preferredMagnetEngine"
                            label="Default Magnet Engine"
                            items={[{value: "aria2", label: "Aria2"}, {value: "native", label: "Native"}]}
                        />
                    </div>
                </Card.Content>
              </Card>
            </section>

            {/* Storage */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Storage</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <FormTextField
                    form={form}
                    name="downloadDir"
                    label="Download Folder"
                    startContent={<IconFolder className="text-muted w-4 h-4" />}
                    placeholder="/downloads"
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
                            name="maxDownloadSpeed"
                            label="Max Download Speed"
                            placeholder="0 (Unlimited)"
                        />
                        <FormTextField
                            form={form}
                            name="maxUploadSpeed"
                            label="Max Upload Speed"
                            placeholder="0 (Unlimited)"
                        />
                    </div>
                    
                    <div className="h-px bg-border" />

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <FormTextField
                            form={form}
                            name="maxConcurrentDownloads"
                            label="Simultaneous Downloads"
                            type="number"
                            placeholder="3"
                        />
                    </div>
                </Card.Content>
              </Card>
            </section>

            {/* Advanced Configuration */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Advanced Configuration</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <FormTextField
                        form={form}
                        name="split"
                        label="Max Splits"
                        type="number"
                    />
                    <div className="h-px bg-border" />
                    <div className="flex flex-col gap-4">
                        <FormSwitch
                            form={form}
                            name="autoResume"
                            label="Auto-Resume Downloads"
                        />
                        <FormSwitch
                            form={form}
                            name="preAllocateSpace"
                            label="Pre-allocate Disk Space"
                        />
                    </div>
                    <div className="h-px bg-border" />
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <FormTextField
                            form={form}
                            name="diskCache"
                            label="Disk Cache"
                            placeholder="32M"
                        />
                        <FormTextField
                            form={form}
                            name="minSplitSize"
                            label="Min Split Size"
                            placeholder="1M"
                        />
                    </div>
                </Card.Content>
              </Card>
            </section>

            {/* Upload Configuration removed from here */}
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
