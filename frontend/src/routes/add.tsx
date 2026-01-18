import {
  Button,
  FieldError,
  Input,
  Label,
  ListBox,
  Select,
  Switch,
  Tabs,
  TextArea,
  TextField,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import React, { useId } from "react";
import { FileTrigger } from "react-aria-components";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconFileArrowUp from "~icons/gravity-ui/file-arrow-up";
import IconGear from "~icons/gravity-ui/gear";
import IconGlobe from "~icons/gravity-ui/globe";
import IconLink from "~icons/gravity-ui/link";
import IconShieldCheck from "~icons/gravity-ui/shield-check";
import { useAria2Actions } from "../hooks/useAria2";
import { useRcloneRemotes } from "../hooks/useRclone";
import { useFileStore } from "../store/useFileStore";
import { useSettingsStore } from "../store/useSettingsStore";
import { tasksLinkOptions } from "./tasks";

export const Route = createFileRoute("/add")({
  component: AddDownloadPage,
});

function AddDownloadPage() {
  const navigate = useNavigate();
  const baseId = useId();

  const { pendingFile, clearPendingFile } = useFileStore();
  const { rcloneTargetRemote, setRcloneTargetRemote } = useSettingsStore();

  const [selectedTab, setSelectedTab] = React.useState<React.Key>(
    `${baseId}-links`,
  );
  const [optionsTab, setOptionsTab] = React.useState<React.Key>(
    `${baseId}-opt-general`,
  );

  const [uris, setUris] = React.useState("");
  const [options, setOptions] = React.useState<Record<string, string>>({
    dir: "",
    split: "5",
    "max-connection-per-server": "1",
    "user-agent": "",
    referer: "",
    "seed-ratio": "1.0",
    "seed-time": "0",
  });

  const { data: remotes = [] } = useRcloneRemotes();

  const remoteOptions = remotes.map((r) => ({ id: r, name: r }));

  const { addUri, addTorrent, addMetalink } = useAria2Actions();

  const handleFileSelect = React.useCallback(
    async (files: FileList | null) => {
      if (!files) return;
      const fileList = Array.from(files);

      for (const file of fileList) {
        const reader = new FileReader();
        reader.onload = () => {
          const base64 = (reader.result as string).split(",")[1];
          const onSuccess = () => navigate(tasksLinkOptions("active"));

          if (file.name.endsWith(".torrent")) {
            addTorrent.mutate(
              {
                torrent: base64,
                options: options,
              },
              { onSuccess },
            );
          } else if (file.name.endsWith(".metalink")) {
            addMetalink.mutate(
              {
                metalink: base64,
                options: options,
              },
              { onSuccess },
            );
          }
        };
        reader.readAsDataURL(file);
      }
    },
    [addTorrent, addMetalink, options, navigate],
  );

  React.useEffect(() => {
    if (pendingFile) {
      handleFileSelect([pendingFile] as any);
      clearPendingFile();
    }
  }, [pendingFile, clearPendingFile, handleFileSelect]);

  const validateUris = (val: string) => {
    if (!val.trim()) return "Enter at least one link";
    const lines = val.split("\n").filter((l) => l.trim());
    const invalid = lines.find(
      (l) =>
        !/^(http|https|ftp|sftp|magnet):/i.test(l.trim()) &&
        !/^[a-f0-9]{40}$/i.test(l.trim()),
    );
    if (invalid) return "Invalid protocol in one of the links";
    return true;
  };

  const handleSubmit = async () => {
    const onSuccess = () => navigate(tasksLinkOptions("active"));
    const cleanOptions = Object.fromEntries(
      Object.entries(options).filter(([_, v]) => v !== ""),
    );

    if (rcloneTargetRemote) {
      cleanOptions["rclone-target"] = rcloneTargetRemote;
    }

    if (selectedTab === `${baseId}-links` && validateUris(uris) === true) {
      const uriList = uris.split("\n").filter((u) => u.trim());
      addUri.mutate(
        {
          uris: uriList,
          options: cleanOptions,
        },
        { onSuccess },
      );
    }
  };

  const updateOption = (key: string, value: string) => {
    setOptions((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="max-w-5xl mx-auto space-y-6 pb-20">
      {/* Header */}
      <div className="flex items-center justify-between bg-background p-4 rounded-3xl border border-border shadow-sm">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            isIconOnly
            onPress={() => navigate(tasksLinkOptions("active"))}
            className="h-10 w-10 rounded-xl"
          >
            <IconChevronLeft className="w-5 h-5" />
          </Button>
          <h2 className="text-xl font-black uppercase tracking-tight">
            Add Download
          </h2>
        </div>
        <div className="flex gap-2">
          <Button
            variant="ghost"
            className="px-6 h-10 rounded-xl font-bold"
            onPress={() => navigate(tasksLinkOptions("active"))}
          >
            Cancel
          </Button>
          <Button
            className="px-8 h-10 rounded-xl font-black uppercase tracking-widest shadow-lg shadow-accent/20 bg-accent text-accent-foreground"
            onPress={handleSubmit}
            isDisabled={
              selectedTab === `${baseId}-links`
                ? validateUris(uris) !== true
                : false
            }
          >
            Start
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Input Section */}
        <div className="lg:col-span-7 space-y-6">
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
            <Tabs
              aria-label="Download Type"
              selectedKey={selectedTab as string}
              onSelectionChange={setSelectedTab}
              className="w-full mb-8"
            >
              <Tabs.ListContainer className="bg-default/10 p-1 rounded-2xl">
                <Tabs.List className="w-full">
                  <Tabs.Tab id={`${baseId}-links`} className="w-full py-2">
                    <div className="flex items-center justify-center gap-2">
                      <IconLink className="w-4 h-4" />
                      <span className="font-bold text-sm">Links / Magnet</span>
                    </div>
                    <Tabs.Indicator className="bg-background rounded-xl shadow-sm" />
                  </Tabs.Tab>
                  <Tabs.Tab id={`${baseId}-torrent`} className="w-full py-2">
                    <div className="flex items-center justify-center gap-2">
                      <IconFileArrowUp className="w-4 h-4" />
                      <span className="font-bold text-sm">Torrent / File</span>
                    </div>
                    <Tabs.Indicator className="bg-background rounded-xl shadow-sm" />
                  </Tabs.Tab>
                </Tabs.List>
              </Tabs.ListContainer>
            </Tabs>

            <div className="min-h-[240px]">
              {selectedTab === `${baseId}-links` && (
                <TextField
                  className="w-full"
                  value={uris}
                  onChange={setUris}
                  validate={validateUris}
                  validationBehavior="aria"
                >
                  <div className="flex flex-col gap-3">
                    <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                      Download URLs
                    </Label>
                    <div className="relative group">
                      <TextArea
                        placeholder="Paste HTTP, FTP or Magnet links here..."
                        className="w-full p-6 bg-default/10 rounded-3xl text-sm border border-transparent focus:bg-default/15 focus:border-accent/30 transition-all outline-none min-h-[200px] leading-relaxed font-mono data-[invalid=true]:border-danger/50"
                      />
                      <FieldError className="absolute -bottom-6 right-1 text-[10px] text-danger font-black uppercase tracking-widest animate-in fade-in slide-in-from-top-1" />
                    </div>
                  </div>
                </TextField>
              )}

              {selectedTab === `${baseId}-torrent` && (
                <div className="flex flex-col gap-3 h-full">
                  <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                    Select Files
                  </Label>
                  <FileTrigger
                    acceptedFileTypes={[".torrent", ".metalink"]}
                    allowsMultiple
                    onSelect={handleFileSelect}
                  >
                    <Button
                      variant="secondary"
                      className="flex flex-col items-center justify-center p-12 border-2 border-dashed border-border rounded-[40px] text-muted gap-4 hover:border-accent hover:text-accent transition-all cursor-pointer w-full h-[200px] bg-default/5 hover:bg-accent/5 overflow-hidden group"
                    >
                      <div className="w-20 h-20 bg-background rounded-full flex items-center justify-center shadow-lg border border-border group-hover:scale-110 group-hover:rotate-6 transition-all duration-500">
                        <IconFileArrowUp className="w-10 h-10 opacity-30 text-accent group-hover:opacity-100" />
                      </div>
                      <div className="flex flex-col gap-1 items-center">
                        <p className="text-center w-full font-bold">
                          Drop .torrent or .metalink files here
                        </p>
                        <p className="text-[10px] uppercase font-black tracking-widest opacity-60">
                          or click to browse local files
                        </p>
                      </div>
                    </Button>
                  </FileTrigger>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Options Section */}
        <div className="lg:col-span-5 space-y-6">
          <div className="bg-background rounded-[32px] border border-border shadow-sm overflow-hidden flex flex-col h-full">
            <div className="p-6 border-b border-border bg-default/5 flex items-center gap-3">
              <div className="p-2 bg-accent/10 rounded-xl text-accent">
                <IconGear className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-black uppercase tracking-widest text-xs">
                  Download Options
                </h3>
                <p className="text-[10px] text-muted font-bold">
                  Configuration for this task
                </p>
              </div>
            </div>

            <Tabs
              aria-label="Option Categories"
              selectedKey={optionsTab as string}
              onSelectionChange={setOptionsTab}
              className="flex-1 flex flex-col"
            >
              <Tabs.ListContainer className="p-3 border-b border-border bg-background">
                <Tabs.List className="w-full gap-1">
                  <Tabs.Tab
                    id={`${baseId}-opt-general`}
                    className="flex-1 py-2"
                  >
                    <div className="flex flex-col items-center gap-1">
                      <IconGear className="w-4 h-4" />
                      <span className="text-[9px] font-black uppercase">
                        General
                      </span>
                    </div>
                    <Tabs.Indicator className="bg-accent/10 rounded-xl" />
                  </Tabs.Tab>
                  <Tabs.Tab id={`${baseId}-opt-http`} className="flex-1 py-2">
                    <div className="flex flex-col items-center gap-1">
                      <IconGlobe className="w-4 h-4" />
                      <span className="text-[9px] font-black uppercase">
                        HTTP
                      </span>
                    </div>
                    <Tabs.Indicator className="bg-accent/10 rounded-xl" />
                  </Tabs.Tab>
                  <Tabs.Tab id={`${baseId}-opt-bt`} className="flex-1 py-2">
                    <div className="flex flex-col items-center gap-1">
                      <IconShieldCheck className="w-4 h-4" />
                      <span className="text-[9px] font-black uppercase">
                        BT
                      </span>
                    </div>
                    <Tabs.Indicator className="bg-accent/10 rounded-xl" />
                  </Tabs.Tab>
                  <Tabs.Tab id={`${baseId}-opt-cloud`} className="flex-1 py-2">
                    <div className="flex flex-col items-center gap-1">
                      <IconGlobe className="w-4 h-4" />
                      <span className="text-[9px] font-black uppercase">
                        Cloud
                      </span>
                    </div>
                    <Tabs.Indicator className="bg-accent/10 rounded-xl" />
                  </Tabs.Tab>
                </Tabs.List>
              </Tabs.ListContainer>

              <div className="p-6 flex-1 overflow-y-auto">
                {optionsTab === `${baseId}-opt-general` && (
                  <div className="space-y-6">
                    <TextField
                      value={options.dir}
                      onChange={(v) => updateOption("dir", v)}
                    >
                      <div className="flex flex-col gap-2">
                        <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                          Download Directory
                        </Label>
                        <Input
                          placeholder="/path/to/downloads"
                          className="w-full h-10 px-4 bg-default/10 rounded-xl text-sm border-none focus:bg-default/20 transition-all outline-none"
                        />
                      </div>
                    </TextField>

                    <div className="grid grid-cols-2 gap-4">
                      <TextField
                        value={options.split}
                        onChange={(v) => updateOption("split", v)}
                      >
                        <div className="flex flex-col gap-2">
                          <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                            Split Conn.
                          </Label>
                          <Input
                            type="number"
                            className="w-full h-10 px-4 bg-default/10 rounded-xl text-sm border-none focus:bg-default/20 transition-all outline-none"
                          />
                        </div>
                      </TextField>
                      <TextField
                        value={options["max-connection-per-server"]}
                        onChange={(v) =>
                          updateOption("max-connection-per-server", v)
                        }
                      >
                        <div className="flex flex-col gap-2">
                          <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                            Conn/Serv
                          </Label>
                          <Input
                            type="number"
                            className="w-full h-10 px-4 bg-default/10 rounded-xl text-sm border-none focus:bg-default/20 transition-all outline-none"
                          />
                        </div>
                      </TextField>
                    </div>
                  </div>
                )}

                {optionsTab === `${baseId}-opt-http` && (
                  <div className="space-y-6">
                    <TextField
                      value={options["user-agent"]}
                      onChange={(v) => updateOption("user-agent", v)}
                    >
                      <div className="flex flex-col gap-2">
                        <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                          User Agent
                        </Label>
                        <Input
                          placeholder="aria2/1.x"
                          className="w-full h-10 px-4 bg-default/10 rounded-xl text-sm border-none focus:bg-default/20 transition-all outline-none"
                        />
                      </div>
                    </TextField>

                    <TextField
                      value={options.referer}
                      onChange={(v) => updateOption("referer", v)}
                    >
                      <div className="flex flex-col gap-2">
                        <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                          Referer
                        </Label>
                        <Input
                          placeholder="https://..."
                          className="w-full h-10 px-4 bg-default/10 rounded-xl text-sm border-none focus:bg-default/20 transition-all outline-none"
                        />
                      </div>
                    </TextField>

                    <div className="flex items-center justify-between p-4 rounded-2xl bg-default/5 border border-border/50">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs font-bold">Integrity</span>
                        <span className="text-[9px] text-muted font-black uppercase">
                          Verify Piece Hashes
                        </span>
                      </div>
                      <Switch
                        isSelected={options["check-integrity"] === "true"}
                        onChange={(v) =>
                          updateOption("check-integrity", String(v))
                        }
                      />
                    </div>
                  </div>
                )}

                {optionsTab === `${baseId}-opt-bt` && (
                  <div className="space-y-6">
                    <div className="grid grid-cols-2 gap-4">
                      <TextField
                        value={options["seed-ratio"]}
                        onChange={(v) => updateOption("seed-ratio", v)}
                      >
                        <div className="flex flex-col gap-2">
                          <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                            Seed Ratio
                          </Label>
                          <Input
                            type="number"
                            step="0.1"
                            className="w-full h-10 px-4 bg-default/10 rounded-xl text-sm border-none focus:bg-default/20 transition-all outline-none"
                          />
                        </div>
                      </TextField>
                      <TextField
                        value={options["seed-time"]}
                        onChange={(v) => updateOption("seed-time", v)}
                      >
                        <div className="flex flex-col gap-2">
                          <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                            Seed Time (m)
                          </Label>
                          <Input
                            type="number"
                            className="w-full h-10 px-4 bg-default/10 rounded-xl text-sm border-none focus:bg-default/20 transition-all outline-none"
                          />
                        </div>
                      </TextField>
                    </div>

                    <div className="flex items-center justify-between p-4 rounded-2xl bg-default/5 border border-border/50">
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs font-bold">Metadata</span>
                        <span className="text-[9px] text-muted font-black uppercase">
                          Metadata Only
                        </span>
                      </div>
                      <Switch
                        isSelected={options["bt-metadata-only"] === "true"}
                        onChange={(v) =>
                          updateOption("bt-metadata-only", String(v))
                        }
                      />
                    </div>
                  </div>
                )}

                {optionsTab === `${baseId}-opt-cloud` && (
                  <div className="space-y-6">
                    <div className="flex flex-col gap-2">
                      <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                        Target Remote
                      </Label>
                      <Select
                        value={rcloneTargetRemote || "none"}
                        onChange={(key) =>
                          setRcloneTargetRemote(
                            key === "none" ? "" : (key as string),
                          )
                        }
                        placeholder="Select a remote..."
                      >
                        <Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl hover:bg-default/20 transition-colors border-none outline-none">
                          <Select.Value className="text-sm font-medium" />
                          <Select.Indicator className="text-muted">
                            <IconChevronLeft className="w-4 h-4 -rotate-90" />
                          </Select.Indicator>
                        </Select.Trigger>
                        <Select.Popover className="min-w-240 p-2 bg-background border border-border rounded-2xl shadow-xl">
                          <ListBox items={remoteOptions}>
                            {(item) => (
                              <ListBox.Item
                                id={item.id}
                                textValue={item.name}
                                className="rounded-lg px-2 py-1.5 text-sm font-medium hover:bg-default/10 cursor-pointer transition-colors flex items-center justify-between"
                              >
                                <Label>{item.name}</Label>
                              </ListBox.Item>
                            )}
                          </ListBox>
                        </Select.Popover>
                      </Select>
                      <p className="text-[10px] text-muted">
                        Files will be automatically uploaded to this remote upon
                        completion.
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </Tabs>
          </div>
        </div>
      </div>
    </div>
  );
}
