import { useState, useId, useDeferredValue, useMemo } from "react";
import { Button, Chip, Label, Input, TextArea, TextField, Select, ListBox } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm, useStore } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnet from "~icons/gravity-ui/magnet";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import { useDownloadActions } from "../hooks/useDownloads";
import { openapi } from "../lib/openapi";
import { tasksLinkOptions } from "./tasks";
import {
  FileTree,
  getAllFileIds,
  getSelectedSize,
} from "../components/ui/FileTree";
import { formatBytes } from "../lib/utils";
import type { components } from "../gen/api";

type DownloadFile = components["schemas"]["model.DownloadFile"];
type MagnetFile = DownloadFile & { isFolder?: boolean; children?: MagnetFile[] };

export const Route = createFileRoute("/add")({
  component: AddDownloadPage,
});

function AddDownloadPage() {
  const navigate = useNavigate();
  const fileInputId = useId();
  const { create } = useDownloadActions();

  const [isTorrent, setIsTorrent] = useState(false);
  const [torrentBase64, setTorrentBase64] = useState<string | null>(null);

  // Selection state with ID tracking to avoid useEffect for syncing
  const [selection, setSelection] = useState<{ id: string | null, keys: Set<string> }>({ 
    id: null, 
    keys: new Set() 
  });

  // Form setup
  const form = useForm({
    defaultValues: {
      uris: "",
      filename: "",
      downloadDir: "",
      destination: "",
      headersInput: "",
      split: 8,
      maxTries: 5,
      userAgent: "",
      proxyUrl: "",
      removeLocal: undefined as boolean | undefined,
    },
    onSubmit: async ({ value }) => {
      const currentUrlValue = value.uris.trim().split("\n")[0]?.trim();
      const headers: Record<string, string> = {};
      
      if (value.headersInput.trim()) {
        value.headersInput.split('\n').forEach(line => {
          const parts = line.split(':');
          if (parts.length >= 2) {
            const key = parts[0].trim();
            const val = parts.slice(1).join(':').trim();
            if (key && val) headers[key] = val;
          }
        });
      }

      const flatOptions = {
        downloadDir: value.downloadDir || undefined,
        destination: value.destination || undefined,
        split: value.split,
        maxTries: value.maxTries,
        userAgent: value.userAgent || undefined,
        proxyUrl: value.proxyUrl || undefined,
        removeLocal: value.removeLocal,
        headers: Object.keys(headers).length > 0 ? headers : undefined,
      };

      const payload: any = {
        ...flatOptions,
      };

      if (isTorrent && torrentBase64) {
        payload.url = ""; 
        payload.torrentData = torrentBase64;
      } else if (currentUrlValue) {
        payload.url = currentUrlValue;
      } else {
        return;
      }

      if (resolveData?.data?.result) {
        const result = resolveData.data.result;
        payload.provider = resolveData.data.provider;
        payload.filename = result.name || value.filename || undefined;
        
        if (result.isMagnet || isTorrent) {
            const allFiles = (result.files || []) as DownloadFile[];
            const selectedIndexes = allFiles
                .map((f, i) => selectedFiles.has(f.path || "") ? i : -1)
                .filter(i => i !== -1);
            
            payload.selectedFiles = selectedIndexes;
        }
      } else {
        payload.filename = value.filename || undefined;
      }

      create.mutate(
        {
          body: payload,
        },
        {
          onSuccess: () => navigate(tasksLinkOptions("active")),
        },
      );
    },
  });

  // Reactive uris value for queries
  const uris = useStore(form.store, (s) => s.values.uris);
  const deferredUris = useDeferredValue(uris);
  const currentUrl = deferredUris.trim().split("\n")[0]?.trim();

  // Unified Resolve Query
  const { data: resolveInfo, isLoading: isResolving, error: resolveError } = openapi.useQuery(
    "post",
    "/providers/resolve",
    {
        body: {
            url: isTorrent ? "" : currentUrl!,
            torrentBase64: torrentBase64 || undefined,
        }
    },
    {
        enabled: (!!currentUrl && currentUrl.startsWith("magnet:")) || (isTorrent && !!torrentBase64),
        staleTime: 60000,
        select: (response) => {
            const result = response.data?.result;
            if (!result || !result.files) return { ...response, tree: [] as MagnetFile[] };

            const files = (result.files as DownloadFile[]).map(f => ({
                ...f,
                id: f.path || f.id || "",
            } as MagnetFile));

            return {
                ...response,
                tree: buildTreeFromFlatFiles(files)
            };
        }
    }
  );

  const resolveData = resolveInfo; // Alias for compatibility with rest of code
  const resolveResult = resolveInfo?.data?.result;
  const fileTree = resolveInfo?.tree || [];
  const isMagnet = !!currentUrl && currentUrl.startsWith("magnet:");
  const isComplex = isMagnet || isTorrent || (resolveResult?.files && resolveResult.files.length > 1);

  const currentResolveId = resolveResult?.hash || currentUrl || (isTorrent ? "torrent-upload" : null);

  // Sync selection when ID changes (Standard React pattern to replace useEffect for syncing state with props/data)
  if (currentResolveId !== selection.id && fileTree.length > 0) {
    setSelection({
      id: currentResolveId,
      keys: new Set(getAllFileIds(fileTree))
    });
  }

  const selectedFiles = selection.keys;
  const setSelectedFiles = (keys: Set<string>) => setSelection({ id: currentResolveId, keys });

  const handleTorrentUpload = async (file: File) => {
    setIsTorrent(true);
    form.setFieldValue("uris", "");

    const reader = new FileReader();
    reader.onload = async (e) => {
      const base64 = (e.target?.result as string).split(",")[1];
      setTorrentBase64(base64);
    };
    reader.readAsDataURL(file);
  };

  const selectAllFiles = () => {
    if (fileTree.length > 0) {
      setSelectedFiles(new Set(getAllFileIds(fileTree)));
    }
  };

  const deselectAllFiles = () => {
    setSelectedFiles(new Set());
  };

  const selectedSize = useMemo(() => {
    if (fileTree.length === 0) return 0;
    return getSelectedSize(fileTree, selectedFiles);
  }, [fileTree, selectedFiles]);

  return (
    <div className="max-w-5xl mx-auto space-y-6 pb-20 px-4 md:px-0 mt-6">
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
            onPress={() => form.handleSubmit()}
            isDisabled={
              (!currentUrl && !isTorrent) ||
              create.isPending ||
              isResolving ||
              (isComplex && selectedFiles.size === 0)
            }
            isPending={create.isPending || isResolving}
          >
            Start
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column - URL Input */}
        <div className="lg:col-span-7 space-y-6">
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
            <div className="flex flex-col gap-3">
              <form.Field
                name="uris"
                children={(field: any) => (
                  <TextField className="flex flex-col gap-3">
                    <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                      Download URL
                    </Label>
                    <div className="relative group">
                      <TextArea
                        placeholder="Paste HTTP, FTP or Magnet links here..."
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        className="w-full p-6 bg-default/10 rounded-3xl text-sm border border-transparent focus:bg-default/15 focus:border-accent/30 transition-all outline-none min-h-[120px] leading-relaxed font-mono"
                      />
                      <div className="absolute right-4 bottom-4 flex gap-2">
                        <input
                          type="file"
                          id={fileInputId}
                          className="hidden"
                          accept=".torrent"
                          onChange={(e) => {
                            const file = e.target.files?.[0];
                            if (file) handleTorrentUpload(file);
                          }}
                        />
                        <Button
                          size="sm"
                          variant="secondary"
                          className="rounded-xl font-bold text-[10px] uppercase tracking-widest px-3 h-8"
                          onPress={() => document.getElementById(fileInputId)?.click()}
                        >
                          Upload .torrent
                        </Button>
                      </div>
                    </div>
                  </TextField>
                )}
              />

              {/* Magnet/Torrent indicator */}
              {(isMagnet || isTorrent) && (
                <div className="flex items-center gap-2 px-1">
                  <IconMagnet className="w-4 h-4 text-accent" />
                  <span className="text-xs font-bold text-accent uppercase tracking-widest">
                    {isTorrent ? "Torrent file uploaded" : "Magnet link detected"}
                  </span>
                  {isTorrent && (
                    <Button 
                      variant="ghost" 
                      size="sm" 
                      className="h-6 px-2 min-w-0 text-[10px] uppercase font-black ml-2"
                      onPress={() => {
                        setIsTorrent(false);
                        setTorrentBase64(null);
                      }}
                    >
                      Clear
                    </Button>
                  )}
                </div>
              )}

              {/* Error display */}
              {resolveError && (
                <div className="mt-2 p-4 rounded-2xl bg-danger/5 border border-danger/20 text-danger text-xs font-bold">
                  {(resolveError as any).error || "Failed to process link"}
                </div>
              )}
            </div>
          </div>

          {/* File Selection */}
          {isComplex && resolveResult && (
            <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center justify-between mb-6">
                <div className="flex-1 min-w-0 mr-4">
                  <h3
                    className="font-bold text-lg truncate"
                    title={resolveResult.name}
                  >
                    {resolveResult.name}
                  </h3>
                  <div className="flex items-center gap-2 text-xs text-muted font-black uppercase tracking-widest mt-1">
                    <span>{formatBytes(resolveResult.size || 0)}</span>
                    <span>•</span>
                    <span>{resolveResult.files?.length || 0} files</span>
                  </div>
                </div>
                {resolveData?.data?.provider && (
                    <Chip
                        variant="soft"
                        size="sm"
                        className="font-black uppercase tracking-widest"
                    >
                        {resolveData.data.provider}
                    </Chip>
                )}
              </div>

              {/* Select/Deselect buttons */}
              <div className="flex gap-2 mb-4">
                <Button
                  size="sm"
                  variant="secondary"
                  onPress={selectAllFiles}
                  className="rounded-xl font-bold"
                >
                  Select All
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onPress={deselectAllFiles}
                  className="rounded-xl font-bold"
                >
                  Deselect All
                </Button>
              </div>

              {/* File Tree */}
              <div className="max-h-[500px] overflow-y-auto rounded-2xl border border-border bg-default/5 custom-scrollbar overscroll-y-contain relative">
                <FileTree
                  files={fileTree as any}
                  selectedKeys={selectedFiles}
                  onSelectionChange={setSelectedFiles}
                />
              </div>

              {/* Selection summary */}
              <div className="mt-4 flex items-center justify-between">
                <span className="text-xs font-black uppercase tracking-widest text-muted">
                  {selectedFiles.size} files selected
                </span>
                <span className="text-sm font-bold text-accent">
                  {formatBytes(selectedSize)}
                </span>
              </div>
            </div>
          )}

          {/* Loading state for resolution */}
          {isResolving && (
            <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center justify-center gap-3 py-8 flex-col">
                <div className="animate-spin rounded-full h-8 w-8 border-3 border-accent border-t-transparent" />
                <span className="text-sm text-muted font-bold uppercase tracking-widest">
                  {isTorrent ? "Parsing torrent file..." : "Resolving link..."}
                </span>
              </div>
            </div>
          )}
        </div>

        {/* Right Column - Options */}
        <div className="lg:col-span-5 space-y-6">
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm space-y-6">
            {/* Filename (only for non-complex) */}
            {!isComplex && (
              <form.Field
                name="filename"
                children={(field: any) => (
                  <TextField className="flex flex-col gap-2">
                    <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                      Filename (Optional)
                    </Label>
                    <Input
                      placeholder={resolveResult?.name || "original-name.zip"}
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
                    />
                  </TextField>
                )}
              />
            )}

            <form.Field
              name="downloadDir"
              children={(field: any) => (
                <TextField className="flex flex-col gap-2">
                  <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                    Download Directory (Optional)
                  </Label>
                  <Input
                    placeholder="e.g. /downloads/movies"
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
                  />
                  <p className="text-[10px] text-muted font-medium px-1 leading-relaxed">
                    Local directory where files will be downloaded.
                  </p>
                </TextField>
              )}
            />

            <form.Field
              name="destination"
              children={(field: any) => (
                <TextField className="flex flex-col gap-2">
                  <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                    Final Destination (Remote)
                  </Label>
                  <Input
                    placeholder="e.g. gdrive:/downloads"
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
                  />
                  <p className="text-[10px] text-muted font-medium px-1 leading-relaxed">
                    Enter a remote path to automatically offload files to the cloud
                    after download completes.
                  </p>
                </TextField>
              )}
            />
          </div>

          {/* Download Options */}
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm space-y-6">
            <h3 className="text-sm font-black uppercase tracking-widest text-muted">
              Download Options
            </h3>

            <div className="grid grid-cols-2 gap-4">
              <form.Field
                name="split"
                children={(field: any) => (
                  <TextField className="flex flex-col gap-2">
                    <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                      Split
                    </Label>
                    <Select
                      selectedKey={String(field.state.value)}
                      onSelectionChange={(key) => field.handleChange(Number(key))}
                      className="w-full"
                    >
                      <Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl border-none">
                        <Select.Value className="text-sm font-bold" />
                        <Select.Indicator className="text-muted">
                          <IconChevronDown className="w-4 h-4" />
                        </Select.Indicator>
                      </Select.Trigger>
                      <Select.Popover className="min-w-[100px] p-2 bg-background border border-border rounded-2xl shadow-xl">
                        <ListBox
                          items={[1, 2, 4, 8, 16].map((n) => ({
                            id: String(n),
                            name: String(n),
                          }))}
                        >
                          {(item) => (
                            <ListBox.Item
                              id={item.id}
                              textValue={item.name}
                              className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 text-sm cursor-pointer outline-none"
                            >
                              <Label>{item.name}</Label>
                            </ListBox.Item>
                          )}
                        </ListBox>
                      </Select.Popover>
                    </Select>
                  </TextField>
                )}
              />

              <form.Field
                name="maxTries"
                children={(field: any) => (
                  <TextField className="flex flex-col gap-2">
                    <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                      Max Retries
                    </Label>
                    <Select
                      selectedKey={String(field.state.value)}
                      onSelectionChange={(key) => field.handleChange(Number(key))}
                      className="w-full"
                    >
                      <Select.Trigger className="h-10 px-4 bg-default/10 rounded-xl border-none">
                        <Select.Value className="text-sm font-bold" />
                        <Select.Indicator className="text-muted">
                          <IconChevronDown className="w-4 h-4" />
                        </Select.Indicator>
                      </Select.Trigger>
                      <Select.Popover className="min-w-[100px] p-2 bg-background border border-border rounded-2xl shadow-xl">
                        <ListBox
                          items={[0, 1, 3, 5, 10].map((n) => ({
                            id: String(n),
                            name: String(n),
                          }))}
                        >
                          {(item) => (
                            <ListBox.Item
                              id={item.id}
                              textValue={item.name}
                              className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 text-sm cursor-pointer outline-none"
                            >
                              <Label>{item.name}</Label>
                            </ListBox.Item>
                          )}
                        </ListBox>
                      </Select.Popover>
                    </Select>
                  </TextField>
                )}
              />
            </div>

            <form.Field
              name="userAgent"
              children={(field: any) => (
                <TextField className="flex flex-col gap-2">
                  <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                    User Agent (Optional)
                  </Label>
                  <Input
                    placeholder="Mozilla/5.0..."
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
                  />
                </TextField>
              )}
            />

            <form.Field
              name="proxyUrl"
              children={(field: any) => (
                <TextField className="flex flex-col gap-2">
                  <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                    Proxy URL (Optional)
                  </Label>
                  <Input
                    placeholder="http://proxy:port"
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
                  />
                </TextField>
              )}
            />

            <form.Field
              name="headersInput"
              children={(field: any) => (
                <TextField className="flex flex-col gap-2">
                  <Label className="text-[10px] font-black uppercase tracking-widest text-muted px-1">
                    Custom Headers (Optional)
                  </Label>
                  <TextArea
                    placeholder="User-Agent: MyAgent&#10;Cookie: key=value"
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className="w-full p-4 bg-default/10 rounded-2xl text-xs border-none focus:bg-default/15 transition-all outline-none font-mono min-h-[80px]"
                  />
                </TextField>
              )}
            />
          </div>

          {/* Resolve info */}
          {resolveData?.data?.provider && (
            <div className="bg-background p-6 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center gap-4">
                <div
                  className="w-12 h-12 rounded-2xl flex items-center justify-center shadow-lg bg-accent/10 text-accent shadow-accent/10"
                >
                  <span className="font-black text-lg">
                    {resolveData.data.provider.slice(0, 2).toUpperCase()}
                  </span>
                </div>
                <div>
                  <p className="font-bold text-sm">
                    {resolveData.data.provider}
                  </p>
                  <p className="text-xs text-muted font-medium mt-0.5">
                    Link resolved via {resolveData.data.provider}.
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function buildTreeFromFlatFiles(files: MagnetFile[]): MagnetFile[] {
  // 1. Check if already nested
  if (files.some((f) => f.children && f.children.length > 0)) {
    return files;
  }

  if (files.length === 0) return [];

  const root: MagnetFile[] = [];

  files.forEach((file) => {
    const parts = file.path?.split("/") || [file.name || "unknown"];
    let currentLevel = root;
    let currentPath = "";

    parts.forEach((part: string, index: number) => {
      currentPath = currentPath ? `${currentPath}/${part}` : part;
      const isFile = index === parts.length - 1;

      // Find existing node at this level
      const existing = currentLevel.find((n) => n.name === part);

      if (existing) {
        if (!isFile) {
          if (!existing.children) existing.children = [];
          currentLevel = existing.children as MagnetFile[];
          existing.size = (existing.size || 0) + (file.size || 0); // Aggregate size
        }
      } else {
        if (isFile) {
          // File node - preserve original ID
          const node = { ...file, name: part };
          currentLevel.push(node);
        } else {
          // Folder node
          const folder: MagnetFile = {
            id: currentPath, // Use path as ID for folders
            name: part,
            path: currentPath,
            size: file.size || 0,
            isFolder: true,
            children: [],
          };
          currentLevel.push(folder);
          // Move into new folder
          currentLevel = folder.children! as MagnetFile[];
        }
      }
    });
  });

  return root;
}