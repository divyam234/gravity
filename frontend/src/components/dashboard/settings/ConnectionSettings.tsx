import { Button, Card, Chip, Input, Label, Form, TextField } from "@heroui/react";
import type React from "react";
import { useState } from "react";
import IconCheck from "~icons/gravity-ui/check";
import IconPencil from "~icons/gravity-ui/pencil";
import IconPlus from "~icons/gravity-ui/plus";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useSettingsStore, type ServerConfig } from "../../../store/useSettingsStore";
import { cn } from "../../../lib/utils";

export const ConnectionSettings: React.FC = () => {
  const { servers, activeServerId, addServer, updateServer, removeServer, setActiveServer } = useSettingsStore();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [isAdding, setIsAdding] = useState(false);

  // Form state
  const [formData, setFormData] = useState<Omit<ServerConfig, "id">>({
    name: "",
    serverUrl: "http://localhost:8080/api/v1",
    apiKey: "",
  });

  const resetForm = () => {
    setFormData({
      name: "",
      serverUrl: "http://localhost:8080/api/v1",
      apiKey: "",
    });
    setEditingId(null);
    setIsAdding(false);
  };

  const startAdd = () => {
    resetForm();
    setIsAdding(true);
  };

  const startEdit = (server: ServerConfig) => {
    setFormData({
      name: server.name,
      serverUrl: server.serverUrl,
      apiKey: server.apiKey,
    });
    setEditingId(server.id);
    setIsAdding(false);
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    if (editingId) {
      updateServer(editingId, formData);
    } else {
      addServer(formData);
    }
    resetForm();
  };

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this server?")) {
      removeServer(id);
    }
  };

  const isFormVisible = isAdding || editingId !== null;

  return (
    <div className="space-y-8">
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

            <Form className="space-y-4" onSubmit={handleSave}>
              <TextField isRequired value={formData.name} onChange={(val) => setFormData({...formData, name: val})}>
                <Label className="text-sm font-bold">Name</Label>
                <Input className="bg-background" placeholder="My Home Server" />
              </TextField>

              <TextField isRequired value={formData.serverUrl} onChange={(val) => setFormData({...formData, serverUrl: val})}>
                <Label className="text-sm font-bold">Server URL</Label>
                <Input className="bg-background" placeholder="http://localhost:8080/api/v1" />
              </TextField>

              <TextField value={formData.apiKey} onChange={(val) => setFormData({...formData, apiKey: val})}>
                <Label className="text-sm font-bold">API Key</Label>
                <Input className="bg-background" type="password" placeholder="Optional" />
              </TextField>

              <div className="flex justify-end gap-2 pt-2">
                <Button size="sm" variant="secondary" onPress={resetForm}>Cancel</Button>
                <Button size="sm" variant="primary" type="submit">Save Server</Button>
              </div>
            </Form>
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