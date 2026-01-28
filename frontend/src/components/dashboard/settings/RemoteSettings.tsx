import { Button, Card, Input, Label, Spinner } from "@heroui/react";
import React, { useState } from "react";
import { useForm } from "@tanstack/react-form";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import IconPlus from "~icons/gravity-ui/plus";
import IconCloud from "~icons/gravity-ui/cloud";
import { useRemoteActions, useRemotes } from "../../../hooks/useRemotes";
import { useSettingsStore } from "../../../store/useSettingsStore";
import type { components } from "../../../gen/api";

type Remote = components["schemas"]["engine.Remote"];

export const RemoteSettings: React.FC = () => {
  const { serverSettings, updateServerSettings } = useSettingsStore();
  const { data: remotes = [], isLoading } = useRemotes();
  const { deleteRemote, createRemote } = useRemoteActions();

  const [isAdding, setIsAdding] = useState(false);

  const form = useForm({
    defaultValues: {
      name: "",
      type: "",
      parameters: "{}",
    },
    onSubmit: async ({ value }) => {
      try {
        const config = JSON.parse(value.parameters);
        createRemote.mutate(
          {
            body: {
              name: value.name,
              type: value.type,
              config,
            },
          },
          {
            onSuccess: () => {
              setIsAdding(false);
              form.reset();
            },
          },
        );
      } catch (e) {
        console.error("Invalid JSON in parameters");
      }
    },
  });

  const defaultRemote = serverSettings?.upload?.defaultRemote || "";
  const setDefaultRemote = (val: string) => {
    if (!serverSettings) return;
    updateServerSettings({
      upload: { ...serverSettings.upload!, defaultRemote: val },
    });
  };

  return (
    <div className="space-y-10">
      {/* Remotes List */}
      <section className="space-y-6">
        <div className="flex items-center justify-between border-b border-border pb-2">
          <div>
            <h3 className="text-lg font-bold">Cloud Remotes</h3>
            <p className="text-sm text-muted">
              Manage your cloud storage connections.
            </p>
          </div>
          <Button
            size="sm"
            variant="ghost"
            onPress={() => setIsAdding(!isAdding)}
            className="rounded-xl"
          >
            <IconPlus className="w-4 h-4 mr-2" />
            New Remote
          </Button>
        </div>

        {isAdding && (
          <Card className="p-4 bg-default/5 border-border shadow-none rounded-2xl space-y-4 animate-in slide-in-from-top-2">
            <div className="grid grid-cols-2 gap-4">
              <form.Field name="name">
                {(field: any) => (
                  <div className="space-y-1.5">
                    <Label className="text-xs font-bold uppercase tracking-wider text-muted">
                      Name
                    </Label>
                    <Input
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      placeholder="my-gdrive"
                      className="bg-background/50 rounded-lg"
                    />
                  </div>
                )}
              </form.Field>
              <form.Field name="type">
                {(field: any) => (
                  <div className="space-y-1.5">
                    <Label className="text-xs font-bold uppercase tracking-wider text-muted">
                      Type
                    </Label>
                    <Input
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      placeholder="drive, s3, dropbox..."
                      className="bg-background/50 rounded-lg"
                    />
                  </div>
                )}
              </form.Field>
            </div>
            <form.Field name="parameters">
              {(field: any) => (
                <div className="space-y-1.5">
                  <Label className="text-xs font-bold uppercase tracking-wider text-muted">
                    Parameters (JSON)
                  </Label>
                  <Input
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    placeholder='{"token": "..."}'
                    className="bg-background/50 rounded-lg font-mono"
                  />
                </div>
              )}
            </form.Field>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onPress={() => setIsAdding(false)}>
                Cancel
              </Button>
              <Button
                className="bg-accent text-accent-foreground"
                onPress={() => form.handleSubmit()}
                isPending={createRemote.isPending}
              >
                {createRemote.isPending ? "Creating..." : "Create"}
              </Button>
            </div>
          </Card>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {isLoading ? (
            <div className="col-span-full flex justify-center py-8">
              <Spinner size="sm" />
            </div>
          ) : (
            remotes?.map((remote: Remote) => (
              <Card
                key={remote.name}
                className="flex flex-row items-center justify-between p-3 px-4 bg-default/5 border-border shadow-none rounded-xl group"
              >
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-full bg-background flex items-center justify-center text-lg">
                    {remote.type === "drive"
                      ? "🔵"
                      : remote.type === "s3"
                        ? "📦"
                        : remote.type === "dropbox"
                          ? "🟦"
                          : "☁️"}
                  </div>
                  <div className="flex flex-col">
                    <span className="font-bold tracking-tight text-sm">
                      {remote.name}
                    </span>
                    <span className="text-[10px] text-muted uppercase font-bold tracking-wider">
                      {remote.type}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  {(() => {
                    const isDefault = defaultRemote === `${remote.name}:`;
                    return (
                      <Button
                        isIconOnly
                        size="sm"
                        variant={isDefault ? "primary" : "ghost"}
                        onPress={() => setDefaultRemote(`${remote.name}:`)}
                        className="h-8 w-8 min-w-0"
                      >
                        <span className="text-[10px] font-black">DEF</span>
                      </Button>
                    );
                  })()}
                  <Button
                    isIconOnly
                    size="sm"
                    variant="ghost"
                    className="text-danger h-8 w-8 min-w-0"
                    onPress={() => {
                      if (confirm(`Delete remote "${remote.name}"?`)) {
                        deleteRemote.mutate({
                          params: { path: { name: remote.name || "" } },
                        });
                      }
                    }}
                    isPending={deleteRemote.isPending}
                  >
                    <IconTrashBin className="w-4 h-4" />
                  </Button>
                </div>
              </Card>
            ))
          )}
          {!isLoading && remotes?.length === 0 && (
            <div className="col-span-full flex flex-col items-center justify-center py-12 text-muted gap-4 border-2 border-dashed border-border rounded-2xl">
              <IconCloud className="w-12 h-12 opacity-20" />
              <p>No remotes configured yet.</p>
              <Button
                size="sm"
                variant="secondary"
                onPress={() => setIsAdding(true)}
              >
                Connect Cloud Storage
              </Button>
            </div>
          )}
        </div>
      </section>
    </div>
  );
};
