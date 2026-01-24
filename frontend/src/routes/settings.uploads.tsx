import {
  Button,
  Card,
  ScrollShadow,
  Spinner,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import IconFolder from "~icons/gravity-ui/folder";
import IconCloud from "~icons/gravity-ui/cloud";
import IconCloudArrowUpIn from "~icons/gravity-ui/cloud-arrow-up-in";
import IconCircleCheck from "~icons/gravity-ui/circle-check";
import IconCircle from "~icons/gravity-ui/circle";
import { useRemotes, useRemoteActions } from "../hooks/useRemotes";
import { useSettingsStore } from "../store/useSettingsStore";
import { api } from "../lib/api";
import { FormTextField, FormSwitch } from "../components/ui/FormFields";
import { toast } from "sonner";

export const Route = createFileRoute("/settings/uploads")({
  component: UploadsSettingsPage,
});

function UploadsSettingsPage() {
  const { serverSettings, updateServerSettings, defaultRemote, setDefaultRemote } = useSettingsStore();
  const { data: remotes = [], isLoading } = useRemotes();
  const { deleteRemote } = useRemoteActions();
  const navigate = useNavigate();

  const handleDelete = (remoteName: string) => {
    if (confirm(`Delete remote "${remoteName}"? This cannot be undone.`)) {
      deleteRemote.mutate(remoteName);
      if (defaultRemote?.startsWith(`${remoteName}:`)) {
        setDefaultRemote("");
      }
    }
  };

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <UploadsSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
      remotes={remotes}
      isLoading={isLoading}
      handleDelete={handleDelete}
      navigate={navigate}
    />
  );
}

function UploadsSettingsForm({
  serverSettings,
  updateServerSettings,
  remotes,
  isLoading,
  handleDelete,
  navigate,
}: {
  serverSettings: any;
  updateServerSettings: any;
  remotes: any[];
  isLoading: boolean;
  handleDelete: (name: string) => void;
  navigate: any;
}) {
  const { upload } = serverSettings;

  const form = useForm({
    defaultValues: {
      autoUpload: upload.autoUpload,
      defaultRemote: upload.defaultRemote,
      removeLocal: upload.removeLocal,
      concurrentUploads: upload.concurrentUploads,
      uploadBandwidth: upload.uploadBandwidth,
      maxRetryAttempts: upload.maxRetryAttempts,
      chunkSize: upload.chunkSize,
    },
    onSubmit: async ({ value }) => {
      const updated = {
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
        await api.updateSettings(updated);
        updateServerSettings(updated);
        toast.success("Upload settings saved");
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
          <h2 className="text-2xl font-bold tracking-tight">Uploads</h2>
          <p className="text-xs text-muted">
            Auto-upload behavior & remote connections
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
            {/* Automation */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Automation</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <div className="space-y-3">
                    <FormTextField
                        form={form}
                        name="defaultRemote"
                        label="Target Remote"
                        startContent={
                            <IconCloudArrowUpIn className="text-muted w-4 h-4" />
                        }
                        placeholder="gdrive:/downloads"
                    />
                    <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                        Use "remote:" or "remote:/path" syntax
                    </p>
                  </div>

                  <div className="h-px bg-border" />

                  <FormSwitch
                    form={form}
                    name="autoUpload"
                    label="Auto-Upload to Cloud"
                    description="Automatically upload completed downloads to cloud storage"
                  />

                  <form.Subscribe
                    selector={(state) => [state.values.autoUpload]}
                  >
                    {([autoUpload]) =>
                      autoUpload ? (
                        <div className="pl-0 space-y-3 animate-in slide-in-from-top-2 duration-200">
                          <FormSwitch
                            form={form}
                            name="removeLocal"
                            label="Delete local copy after upload"
                          />
                        </div>
                      ) : null
                    }
                  </form.Subscribe>
                </Card.Content>
              </Card>
            </section>

            {/* Remotes */}
            <section>
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center gap-3">
                  <div className="w-1.5 h-6 bg-accent rounded-full" />
                  <h3 className="text-lg font-bold">Connections</h3>
                </div>
              </div>

              {isLoading ? (
                <div className="flex justify-center py-12">
                  <Spinner size="md" />
                </div>
              ) : remotes.length === 0 ? (
                <Card className="p-8 bg-background/50 border-border border-dashed">
                  <div className="flex flex-col items-center text-center">
                    <div className="w-16 h-16 bg-default/10 rounded-full flex items-center justify-center mb-4">
                      <IconCloud className="w-8 h-8 text-muted" />
                    </div>
                    <h4 className="font-bold text-lg mb-2">
                      No remotes configured
                    </h4>
                    <p className="text-sm text-muted mb-6 max-w-md">
                      Connect your cloud storage via rclone configuration file to enable automatic uploads.
                    </p>
                  </div>
                </Card>
              ) : (
                <div className="space-y-3">
                  {remotes.map((remote: any) => {
                    return (
                      <Card
                        key={remote.name}
                        className="p-5 bg-background/50 border-border"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-4">
                            <div
                              className="w-12 h-12 rounded-2xl flex items-center justify-center text-xl bg-default/10"
                            >
                              {remote.type === "drive"
                                ? "🔵"
                                : remote.type === "s3"
                                  ? "📦"
                                  : remote.type === "dropbox"
                                    ? "🟦"
                                    : remote.type === "onedrive"
                                      ? "🟢"
                                      : remote.type === "local"
                                        ? "💾"
                                        : "📁"}
                            </div>
                            <div>
                              <h4 className="font-bold text-base">
                                {remote.name}
                              </h4>
                              <p className="text-xs text-muted capitalize">
                                {remote.type}
                              </p>
                            </div>
                          </div>

                          <div className="flex items-center gap-2">
                            <form.Subscribe
                                selector={(state) => [state.values.defaultRemote]}
                            >
                                {([defaultRemote]) => {
                                    const isDefault = defaultRemote === `${remote.name}:` || defaultRemote?.startsWith(`${remote.name}:/`);
                                    return (
                                        <Button
                                            size="sm"
                                            variant={isDefault ? "secondary" : "ghost"}
                                            isIconOnly
                                            onPress={() => form.setFieldValue("defaultRemote", `${remote.name}:`)}
                                            className="rounded-xl"
                                        >
                                            {isDefault ? (
                                                <IconCircleCheck className="w-5 h-5 text-success" />
                                            ) : (
                                                <IconCircle className="w-5 h-5 text-muted hover:text-foreground" />
                                            )}
                                        </Button>
                                    )
                                }}
                            </form.Subscribe>
                            <Button
                              size="sm"
                              variant="ghost"
                              onPress={() =>
                                navigate({
                                  to: "/files",
                                  search: { path: `${remote.name}:` },
                                })
                              }
                              className="rounded-xl font-bold"
                            >
                              <IconFolder className="w-4 h-4 mr-1" />
                              Browse
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              isIconOnly
                              onPress={() => handleDelete(remote.name)}
                              className="text-danger"
                            >
                              <IconTrashBin className="w-4 h-4" />
                            </Button>
                          </div>
                        </div>
                      </Card>
                    );
                  })}
                </div>
              )}
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
                      name="uploadBandwidth"
                      label="Upload Speed Limit"
                      placeholder="0 (Unlimited)"
                    />
                    <FormTextField
                      form={form}
                      name="concurrentUploads"
                      label="Concurrent Uploads"
                      type="number"
                    />
                  </div>
                  
                  <div className="h-px bg-border" />

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <FormTextField
                        form={form}
                        name="chunkSize"
                        label="Chunk Size"
                        placeholder="64M"
                    />
                    <FormTextField
                        form={form}
                        name="maxRetryAttempts"
                        label="Max Retry Attempts"
                        type="number"
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