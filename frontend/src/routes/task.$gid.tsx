import {
  Avatar,
  Button,
  Card,
  Chip,
  ListBox,
  ScrollShadow,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import React, { useId } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconCircleInfo from "~icons/gravity-ui/circle-info";
import IconFile from "~icons/gravity-ui/file";
import IconGear from "~icons/gravity-ui/gear";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import IconPersons from "~icons/gravity-ui/persons";
import { SettingField } from "../components/dashboard/settings/SettingField";
import {
  taskFilesOptions,
  taskStatusOptions,
  useAria2Actions,
  useTaskFiles,
  useTaskOption,
  useTaskPeers,
  useTaskServers,
  useTaskStatus,
} from "../hooks/useAria2";
import { aria2AllOptions } from "../lib/aria2-options";
import type { Aria2File } from "../lib/aria2-rpc";
import { aria2 } from "../lib/aria2-rpc";
import { cn, formatBytes } from "../lib/utils";
import { useSettingsStore } from "../store/useSettingsStore";
import { tasksLinkOptions } from "./tasks";

export const Route = createFileRoute("/task/$gid")({
  component: TaskDetailsPage,
  loader: async ({ context: { queryClient }, params: { gid } }) => {
    const { rpcUrl } = useSettingsStore.getState();
    if (!rpcUrl) return;
    await Promise.all([
      queryClient.ensureQueryData(taskStatusOptions(rpcUrl, gid)),
      queryClient.ensureQueryData(taskFilesOptions(rpcUrl, gid)),
    ]);
  },
});

function TaskDetailsPage() {
  const { gid } = Route.useParams();
  const navigate = useNavigate();
  const baseId = useId();
  const { data: task } = useTaskStatus(gid);
  const { data: files } = useTaskFiles(gid);
  const { data: taskOptions } = useTaskOption(gid);
  const { changeOption } = useAria2Actions();

  const [selectedTab, setSelectedTab] = React.useState<React.Key>(
    `${baseId}-overview`,
  );

  const { data: peers } = useTaskPeers(gid, selectedTab === `${baseId}-peers`);
  const { data: servers } = useTaskServers(
    gid,
    selectedTab === `${baseId}-servers`,
  );

  const [selectedKeys, setSelectedKeys] = React.useState<any>(new Set());

  React.useEffect(() => {
    if (files) {
      const selected = new Set(
        files
          .filter((f: Aria2File) => f.selected === "true")
          .map((f: Aria2File) => f.index),
      );
      setSelectedKeys(selected);
    }
  }, [files]);

  const handleSelectionChange = (keys: any) => {
    setSelectedKeys(keys);
    if (keys === "all") return;

    const selectFileStr = Array.from(keys).sort().join(",");
    aria2.changeOption(gid, { "select-file": selectFileStr });
  };

  const handleOptionUpdate = (name: string, value: string) => {
    changeOption.mutate({ gid, options: { [name]: value } });
  };

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      <div className="flex items-center gap-4 px-2">
        <Button
          variant="ghost"
          isIconOnly
          onPress={() => navigate(tasksLinkOptions("active"))}
        >
          <IconChevronLeft className="w-5 h-5" />
        </Button>
        <h2 className="text-2xl font-bold tracking-tight">Task Details</h2>
        <code className="text-xs bg-default/10 border border-border px-3 py-1 rounded-full text-muted font-bold">
          {gid}
        </code>
      </div>

      <div className="flex flex-col md:flex-row gap-8">
        {/* Sidebar / ListBox */}
        <div className="w-full md:w-64 shrink-0">
          <Card className="p-3 shadow-sm border border-border bg-muted-background/20 rounded-[32px]">
            <ListBox
              aria-label="Task Details Sections"
              selectionMode="single"
              selectedKeys={[selectedTab as string]}
              onSelectionChange={(keys: any) => {
                const key = Array.from(keys)[0];
                if (key) setSelectedTab(key as React.Key);
              }}
              className="w-full gap-1"
            >
              <ListBox.Item
                id={`${baseId}-overview`}
                textValue="Overview"
                className={cn(
                  "flex items-center gap-3 px-4 py-3 rounded-2xl cursor-pointer outline-none transition-colors",
                  selectedTab === `${baseId}-overview`
                    ? "bg-accent text-accent-foreground"
                    : "hover:bg-default/10",
                )}
              >
                <div className="flex items-center gap-3">
                  <IconCircleInfo className="w-5 h-5" />
                  <span className="font-bold text-sm">Overview</span>
                </div>
              </ListBox.Item>
              <ListBox.Item
                id={`${baseId}-files`}
                textValue="Files"
                className={cn(
                  "flex items-center gap-3 px-4 py-3 rounded-2xl cursor-pointer outline-none transition-colors",
                  selectedTab === `${baseId}-files`
                    ? "bg-accent text-accent-foreground"
                    : "hover:bg-default/10",
                )}
              >
                <div className="flex items-center gap-3">
                  <IconFile className="w-5 h-5" />
                  <span className="font-bold text-sm">Files</span>
                </div>
              </ListBox.Item>
              <ListBox.Item
                id={`${baseId}-peers`}
                textValue="Peers"
                className={cn(
                  "flex items-center gap-3 px-4 py-3 rounded-2xl cursor-pointer outline-none transition-colors",
                  selectedTab === `${baseId}-peers`
                    ? "bg-accent text-accent-foreground"
                    : "hover:bg-default/10",
                )}
              >
                <div className="flex items-center gap-3">
                  <IconPersons className="w-5 h-5" />
                  <span className="font-bold text-sm">
                    Peers ({peers?.length || 0})
                  </span>
                </div>
              </ListBox.Item>
              <ListBox.Item
                id={`${baseId}-servers`}
                textValue="Servers"
                className={cn(
                  "flex items-center gap-3 px-4 py-3 rounded-2xl cursor-pointer outline-none transition-colors",
                  selectedTab === `${baseId}-servers`
                    ? "bg-accent text-accent-foreground"
                    : "hover:bg-default/10",
                )}
              >
                <div className="flex items-center gap-3">
                  <IconNodesDown className="w-5 h-5" />
                  <span className="font-bold text-sm">Servers</span>
                </div>
              </ListBox.Item>
              <ListBox.Item
                id={`${baseId}-options`}
                textValue="Options"
                className={cn(
                  "flex items-center gap-3 px-4 py-3 rounded-2xl cursor-pointer outline-none transition-colors",
                  selectedTab === `${baseId}-options`
                    ? "bg-accent text-accent-foreground"
                    : "hover:bg-default/10",
                )}
              >
                <div className="flex items-center gap-3">
                  <IconGear className="w-5 h-5" />
                  <span className="font-bold text-sm">Options</span>
                </div>
              </ListBox.Item>
            </ListBox>
          </Card>
        </div>

        {/* Content Area */}
        <div className="flex-1 min-h-[600px]">
          <Card className="h-full overflow-hidden flex flex-col bg-muted-background/20 shadow-sm border border-border rounded-[40px]">
            {selectedTab === `${baseId}-overview` && task && (
              <ScrollShadow className="flex-1 p-8">
                <div className="space-y-10 text-foreground">
                  <section>
                    <h3 className="text-base font-black uppercase tracking-widest text-muted mb-6 flex items-center gap-3">
                      <div className="w-2 h-2 rounded-full bg-accent" />
                      Identity
                    </h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-8 bg-background/50 p-8 rounded-[32px] border border-border shadow-sm">
                      <div className="space-y-1.5">
                        <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                          Filename
                        </p>
                        <p className="text-lg font-bold break-all tracking-tight leading-tight">
                          {task.bittorrent?.info?.name ||
                            task.files[0]?.path?.split("/").pop() ||
                            gid}
                        </p>
                      </div>
                      <div className="space-y-1.5">
                        <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                          GID
                        </p>
                        <p className="font-mono text-sm bg-default/10 px-3 py-1 rounded-full border border-border inline-block font-bold">
                          {gid}
                        </p>
                      </div>
                      {task.infoHash && (
                        <div className="space-y-1.5 md:col-span-2 pt-4 border-t border-border/50">
                          <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                            Info Hash
                          </p>
                          <p className="font-mono text-sm break-all text-accent font-bold">
                            {task.infoHash}
                          </p>
                        </div>
                      )}
                    </div>
                  </section>

                  <section>
                    <h3 className="text-base font-black uppercase tracking-widest text-muted mb-6 flex items-center gap-3">
                      <div className="w-2 h-2 rounded-full bg-accent" />
                      Location
                    </h3>
                    <div className="bg-background/50 p-8 rounded-[32px] border border-border shadow-sm">
                      <div className="space-y-1.5">
                        <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                          Download Directory
                        </p>
                        <p className="text-sm font-bold break-all leading-relaxed">
                          {task.dir}
                        </p>
                      </div>
                    </div>
                  </section>

                  <section>
                    <h3 className="text-base font-black uppercase tracking-widest text-muted mb-6 flex items-center gap-3">
                      <div className="w-2 h-2 rounded-full bg-accent" />
                      Status
                    </h3>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
                      {[
                        {
                          label: "State",
                          value: task.status,
                          color: "success" as const,
                          isChip: true,
                        },
                        { label: "Connections", value: task.connections },
                        {
                          label: "Total Size",
                          value: formatBytes(task.totalLength),
                        },
                        {
                          label: "Uploaded",
                          value: formatBytes(task.uploadLength),
                          isAccent: true,
                        },
                      ].map((stat) => (
                        <div
                          key={stat.label}
                          className="p-6 bg-background/50 rounded-[32px] border border-border shadow-sm flex flex-col items-center justify-center text-center gap-2"
                        >
                          <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                            {stat.label}
                          </p>
                          {stat.isChip ? (
                            <Chip
                              size="sm"
                              variant="soft"
                              color={stat.color}
                              className="uppercase font-black text-[10px] tracking-widest h-6 px-3"
                            >
                              {stat.value}
                            </Chip>
                          ) : (
                            <p
                              className={cn(
                                "text-lg font-black tracking-tight",
                                stat.isAccent && "text-accent",
                              )}
                            >
                              {stat.value}
                            </p>
                          )}
                        </div>
                      ))}
                    </div>
                  </section>
                </div>
              </ScrollShadow>
            )}

            {selectedTab === `${baseId}-files` && (
              <div className="flex flex-col h-full">
                <div className="p-8 border-b border-border flex justify-between items-center bg-background/50">
                  <div className="space-y-1">
                    <span className="text-2xl font-bold block tracking-tight">
                      Files
                    </span>
                    <span className="text-xs text-muted uppercase font-black tracking-widest">
                      {files?.length} items • Multi-selection enabled
                    </span>
                  </div>
                  <Chip
                    variant="soft"
                    color="accent"
                    size="sm"
                    className="font-black text-[10px] tracking-widest px-4 h-7"
                  >
                    SELECTIVE DOWNLOAD
                  </Chip>
                </div>

                <ScrollShadow className="flex-1 p-6">
                  <ListBox
                    aria-label="Files list"
                    selectionMode="multiple"
                    selectedKeys={selectedKeys}
                    onSelectionChange={handleSelectionChange}
                    className="gap-3"
                    items={files || []}
                  >
                    {(file: Aria2File) => (
                      <ListBox.Item
                        key={file.index}
                        id={file.index}
                        textValue={file.path}
                        className="p-5 rounded-[24px] border border-border bg-background/80 hover:border-accent/40 transition-all group data-[selected=true]:border-accent/50 data-[selected=true]:bg-accent/5"
                      >
                        <div className="flex items-start gap-5 w-full">
                          <ListBox.ItemIndicator className="mt-1" />

                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-3">
                              <div className="w-9 h-9 rounded-2xl bg-default/10 flex items-center justify-center group-hover:bg-accent/10 transition-colors">
                                <IconFile className="w-5 h-5 text-muted group-hover:text-accent transition-colors shrink-0" />
                              </div>
                              <span
                                className="text-base font-bold truncate tracking-tight"
                                title={file.path}
                              >
                                {file.path.split("/").pop() || "Unknown File"}
                              </span>
                            </div>
                            <div className="flex gap-6 mt-4 text-xs">
                              <span className="flex items-center gap-2 font-bold">
                                <div className="w-1.5 h-1.5 rounded-full bg-muted/40" />
                                {formatBytes(file.length)}
                              </span>
                              <span className="flex items-center gap-2 text-success font-black uppercase tracking-widest">
                                <div className="w-1.5 h-1.5 rounded-full bg-success" />
                                {(
                                  (Number(file.completedLength) /
                                    Number(file.length)) *
                                  100
                                ).toFixed(1)}
                                % complete
                              </span>
                            </div>
                          </div>
                        </div>
                      </ListBox.Item>
                    )}
                  </ListBox>
                </ScrollShadow>
              </div>
            )}

            {selectedTab === `${baseId}-peers` && (
              <ScrollShadow className="flex-1 p-6">
                {!peers || peers.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-full text-muted gap-6 py-20 opacity-60">
                    <div className="w-24 h-24 rounded-[32px] bg-default/10 flex items-center justify-center">
                      <IconPersons className="w-12 h-12" />
                    </div>
                    <p className="font-bold uppercase tracking-widest text-xs">
                      No peers connected
                    </p>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 gap-4">
                    {peers.map((peer: any) => (
                      <div
                        key={`${peer.ip}-${peer.port}`}
                        className="flex items-center justify-between p-6 rounded-[28px] border border-border bg-background shadow-sm hover:border-accent/40 transition-colors"
                      >
                        <div className="flex items-center gap-5">
                          <Avatar className="w-12 h-12 font-black bg-accent/10 text-accent rounded-2xl">
                            <Avatar.Fallback>
                              {peer.peerId.slice(1, 3).toUpperCase()}
                            </Avatar.Fallback>
                          </Avatar>
                          <div className="flex flex-col gap-1">
                            <div className="flex items-center gap-2">
                              <span className="text-base font-bold tracking-tight">
                                {peer.ip}
                              </span>
                              <span className="text-[10px] bg-default/10 px-2 py-0.5 rounded-lg text-muted font-black border border-border/50">
                                :{peer.port}
                              </span>
                            </div>
                            <span className="text-[10px] font-mono text-muted truncate max-w-[180px] uppercase tracking-tighter">
                              {peer.peerId}
                            </span>
                          </div>
                        </div>
                        <div className="flex gap-10">
                          <div className="flex flex-col items-end gap-1">
                            <span className="text-[9px] text-muted uppercase font-black tracking-widest">
                              Down
                            </span>
                            <span className="text-base text-success font-black">
                              {formatBytes(peer.downloadSpeed)}/s
                            </span>
                          </div>
                          <div className="flex flex-col items-end border-l border-border pl-10 gap-1">
                            <span className="text-[9px] text-muted uppercase font-black tracking-widest">
                              Up
                            </span>
                            <span className="text-base text-accent font-black">
                              {formatBytes(peer.uploadSpeed)}/s
                            </span>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </ScrollShadow>
            )}

            {selectedTab === `${baseId}-servers` && (
              <ScrollShadow className="flex-1 p-6">
                {!servers || servers.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-full text-muted gap-6 py-20 opacity-60">
                    <div className="w-24 h-24 rounded-[32px] bg-default/10 flex items-center justify-center">
                      <IconNodesDown className="w-12 h-12" />
                    </div>
                    <p className="font-bold uppercase tracking-widest text-xs">
                      No server information
                    </p>
                  </div>
                ) : (
                  <div className="space-y-8">
                    {servers.map((srv: any) => (
                      <div key={srv.index} className="space-y-4">
                        <div className="flex items-center gap-3 px-4">
                          <div className="w-2 h-4 rounded-full bg-accent/40" />
                          <div className="text-[10px] font-black text-muted uppercase tracking-[0.2em]">
                            File Index: {srv.index}
                          </div>
                        </div>
                        <div className="space-y-3">
                          {srv.servers.map((s: any) => (
                            <div
                              key={s.uri}
                              className="p-6 rounded-[28px] border border-border bg-background flex justify-between items-center shadow-sm hover:border-accent/40 transition-colors"
                            >
                              <div className="flex flex-col min-w-0 gap-2">
                                <span className="text-sm font-bold truncate text-accent tracking-tight">
                                  {s.uri}
                                </span>
                                <div className="flex items-center gap-3">
                                  {s.uri === srv.currentUri && (
                                    <Chip
                                      size="sm"
                                      variant="soft"
                                      color="success"
                                      className="h-5 text-[9px] font-black tracking-widest px-3"
                                    >
                                      CURRENT
                                    </Chip>
                                  )}
                                  <span className="text-[10px] text-muted font-black uppercase tracking-widest">
                                    Priority: {s.currentPriority}
                                  </span>
                                </div>
                              </div>
                              <div className="flex flex-col items-end shrink-0 ml-6 gap-1">
                                <span className="text-[9px] text-muted uppercase font-black tracking-widest">
                                  Speed
                                </span>
                                <span className="text-base text-success font-black">
                                  {formatBytes(s.downloadSpeed)}/s
                                </span>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </ScrollShadow>
            )}

            {selectedTab === `${baseId}-options` && taskOptions && (
              <ScrollShadow className="flex-1 p-8">
                <div className="space-y-2">
                  <div className="flex items-center justify-between mb-8">
                    <div className="space-y-1">
                      <h3 className="text-2xl font-bold tracking-tight">
                        Task Options
                      </h3>
                      <p className="text-xs text-muted font-black uppercase tracking-widest">
                        Modify settings for this specific download
                      </p>
                    </div>
                    <Chip
                      size="sm"
                      variant="soft"
                      className="font-black text-[10px] tracking-widest px-4"
                    >
                      {Object.keys(taskOptions).length} OPTIONS
                    </Chip>
                  </div>
                  <div className="flex flex-col">
                    {Object.keys(aria2AllOptions)
                      .filter((name) => taskOptions[name] !== undefined)
                      .map((name) => (
                        <SettingField
                          key={name}
                          opt={{ ...aria2AllOptions[name], name } as any}
                          value={taskOptions[name]}
                          onUpdate={handleOptionUpdate}
                        />
                      ))}
                  </div>
                </div>
              </ScrollShadow>
            )}
          </Card>
        </div>
      </div>
    </div>
  );
}
