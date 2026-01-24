import { Button, Card, Chip, Input, Label, TextField } from "@heroui/react";
import type React from "react";
import { useState } from "react";
import { useForm } from "@tanstack/react-form";
import IconCheck from "~icons/gravity-ui/check";
import IconPencil from "~icons/gravity-ui/pencil";
import IconPlus from "~icons/gravity-ui/plus";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import IconServer from "~icons/gravity-ui/server";
import { useSettingsStore, type ServerConfig } from "../../../store/useSettingsStore";
import { cn } from "../../../lib/utils";
import { useGravityVersion } from "../../../hooks/useEngine";

export const ConnectionSettings: React.FC = () => {
  const { servers, activeServerId, addServer, updateServer, removeServer, setActiveServer } = useSettingsStore();
  const { data: version } = useGravityVersion();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [isAdding, setIsAdding] = useState(false);

  const form = useForm({
    defaultValues: {
      name: "",
      serverUrl: "http://localhost:8080/api/v1",
      apiKey: "",
    },
    onSubmit: async ({ value }) => {
      if (editingId) {
        updateServer(editingId, value);
      } else {
        addServer(value);
      }
      resetForm();
    },
  });

  const activeServer = servers.find(s => s.id === activeServerId);

  const resetForm = () => {
    form.reset();
    setEditingId(null);
    setIsAdding(false);
  };

  const startAdd = () => {
    resetForm();
    setIsAdding(true);
  };

  const startEdit = (server: ServerConfig) => {
    form.setFieldValue("name", server.name);
    form.setFieldValue("serverUrl", server.serverUrl);
    form.setFieldValue("apiKey", server.apiKey);
    setEditingId(server.id);
    setIsAdding(false);
  };

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this server?")) {
      removeServer(id);
    }
  };

  const isFormVisible = isAdding || editingId !== null;

  return (
    <div className="space-y-8">
      {/* Active Server Status */}
      {activeServer && (
        <Card className="p-6 border border-accent/20 bg-accent/5">
            <div className="flex items-center gap-4">
                <div className="p-3 bg-accent/10 rounded-full">
                    <IconServer className="w-6 h-6 text-accent" />
                </div>
                <div>
                    <h4 className="font-bold text-lg flex items-center gap-2">
                        {activeServer.name}
                        <Chip size="sm" color="success" variant="soft" className="px-2">
                            <span className="flex items-center gap-1 font-bold">
                                <IconCheck className="w-3 h-3" /> Connected
                            </span>
                        </Chip>
                    </h4>
                    <p className="text-sm text-muted">
                        Version: <span className="font-mono font-bold">{version?.version || "Unknown"}</span>
                        {version?.aria2 && <span className="text-xs text-muted ml-2">Aria2: {version.aria2}</span>}
                        {version?.rclone && <span className="text-xs text-muted ml-2">Rclone: {version.rclone}</span>}
                    </p>
                </div>
            </div>
        </Card>
      )}

      <div className="space-y-6">
        <div className="flex items-center justify-between border-b border-border pb-2">
          <div>
            <h3 className="text-lg font-bold">Servers</h3>
            <p className="text-sm text-muted">
              Manage Gravity server connections.
            </p>
          </div>
          {!isFormVisible && (
            <Button size="sm" variant="primary" onPress={startAdd} className="font-bold">
              <IconPlus className="w-4 h-4 mr-1" />
              Add Server
            </Button>
          )}
        </div>

        {isFormVisible && (
          <Card className="p-6 border border-accent/20 bg-accent/5">
            <h4 className="text-sm font-bold uppercase tracking-wider mb-4 flex items-center gap-2">
              {isAdding ? <IconPlus className="w-4 h-4" /> : <IconPencil className="w-4 h-4" />}
              {isAdding ? "Add New Server" : "Edit Server"}
            </h4>

            <form.Subscribe selector={(s: any) => [s.canSubmit, s.isSubmitting]}>
              {() => (
                <div className="space-y-4">
                  <form.Field
                    name="name"
                    children={(field: any) => (
                      <TextField isRequired value={field.state.value} onChange={(v) => field.handleChange(v)}>
                        <Label className="text-sm font-bold">Name</Label>
                        <Input className="bg-background" placeholder="My Home Server" />
                      </TextField>
                    )}
                  />

                  <form.Field
                    name="serverUrl"
                    children={(field: any) => (
                      <TextField isRequired value={field.state.value} onChange={(v) => field.handleChange(v)}>
                        <Label className="text-sm font-bold">Server URL</Label>
                        <Input className="bg-background" placeholder="http://localhost:8080/api/v1" />
                      </TextField>
                    )}
                  />

                  <form.Field
                    name="apiKey"
                    children={(field: any) => (
                      <TextField value={field.state.value} onChange={(v) => field.handleChange(v)}>
                        <Label className="text-sm font-bold">API Key</Label>
                        <Input className="bg-background" type="password" placeholder="Optional" />
                      </TextField>
                    )}
                  />

                  <div className="flex justify-end gap-2 pt-2">
                    <Button size="sm" variant="secondary" onPress={resetForm}>Cancel</Button>
                    <Button size="sm" variant="primary" onPress={() => form.handleSubmit()}>Save Server</Button>
                  </div>
                </div>
              )}
            </form.Subscribe>
          </Card>
        )}

        <div className="grid gap-3">
          {servers.map((server) => (
            <div
              key={server.id}
              className={cn(
                "group relative p-4 rounded-2xl border transition-all duration-200",
                activeServerId === server.id
                  ? "bg-accent/10 border-accent shadow-sm"
                  : "bg-muted-background border-border hover:border-accent/50",
              )}
            >
              <div className="flex items-center justify-between">
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <span className="font-bold text-base">{server.name}</span>
                    {activeServerId === server.id && (
                      <Chip size="sm" color="success" variant="soft" className="h-5 px-1">
                        <span className="flex items-center gap-1 text-[10px] font-bold uppercase">
                          <IconCheck className="w-3 h-3" /> Active
                        </span>
                      </Chip>
                    )}
                  </div>
                  <span className="text-xs font-mono text-muted truncate max-w-[300px]">
                    {server.serverUrl}
                  </span>
                </div>

                <div className="flex items-center gap-2">
                  {activeServerId !== server.id && (
                    <Button
                      size="sm"
                      variant="ghost"
                      onPress={() => setActiveServer(server.id)}
                      className="font-bold text-xs"
                    >
                      Connect
                    </Button>
                  )}

                  <div className="flex items-center gap-1 border-l border-border/50 pl-2 ml-2">
                    <Button isIconOnly size="sm" variant="ghost" onPress={() => startEdit(server)}>
                      <IconPencil className="w-4 h-4 text-muted-foreground" />
                    </Button>
                    <Button
                      isIconOnly
                      size="sm"
                      variant="ghost"
                      onPress={() => handleDelete(server.id)}
                      isDisabled={servers.length <= 1}
                    >
                      <IconTrashBin className="w-4 h-4 text-danger" />
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};