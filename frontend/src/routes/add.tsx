import { useState, useEffect, useCallback, useId } from "react";
import { Button, Chip, Label, Input, TextArea } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconMagnet from "~icons/gravity-ui/magnet";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import { useDownloadActions } from "../hooks/useDownloads";
import { api } from "../lib/api";
import { useSettingsStore } from "../store/useSettingsStore";
import { tasksLinkOptions } from "./tasks";
import {
  FileTree,
  getAllFileIds,
  getSelectedSize,
} from "../components/ui/FileTree";
import { formatBytes } from "../lib/utils";
import type { MagnetInfo, MagnetFile } from "../lib/types";

export const Route = createFileRoute("/add")({
  component: AddDownloadPage,
});

function AddDownloadPage() {
  const navigate = useNavigate();
  const fileInputId = useId();
  const { defaultRemote, setDefaultRemote } = useSettingsStore();

  const [uris, setUris] = useState("");
  const [filename, setFilename] = useState("");
  const [resolution, setResolution] = useState<{
    provider: string;
    supported: boolean;
  } | null>(null);

  // Magnet/Torrent state
  const [isMagnet, setIsMagnet] = useState(false);
  const [isTorrent, setIsTorrent] = useState(false);
  const [torrentBase64, setTorrentBase64] = useState<string | null>(null);
  const [isCheckingMagnet, setIsCheckingMagnet] = useState(false);
  const [checkStatus, setCheckStatus] = useState("Checking cache...");
  const [magnetInfo, setMagnetInfo] = useState<MagnetInfo | null>(null);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [magnetError, setMagnetError] = useState<string | null>(null);

  const { create } = useDownloadActions();

  const checkMagnet = useCallback(async (magnet: string) => {
    setIsCheckingMagnet(true);
    setCheckStatus("Checking AllDebrid cache...");
    setMagnetError(null);
    setMagnetInfo(null);
    setIsTorrent(false);
    setTorrentBase64(null);

    // Timer to update status if it takes long
    const statusTimer = setTimeout(() => {
      setCheckStatus("Searching for peers (fetching metadata)...");
    }, 3000);

    try {
      const info = await api.checkMagnet(magnet);
      setMagnetInfo(info);

      // Pre-select all files
      const allIds = getAllFileIds(info.files);
      setSelectedFiles(new Set(allIds));
    } catch (err: any) {
      setMagnetError(err.message || "Failed to check magnet");
    } finally {
      clearTimeout(statusTimer);
      setIsCheckingMagnet(false);
    }
  }, []);

  const handleTorrentUpload = async (file: File) => {
    setIsCheckingMagnet(true);
    setCheckStatus("Parsing torrent file...");
    setMagnetError(null);
    setMagnetInfo(null);
    setIsMagnet(false);
    setIsTorrent(true);
    setUris(""); // Clear URIs if a torrent is uploaded

    const reader = new FileReader();
    reader.onload = async (e) => {
      const base64 = (e.target?.result as string).split(",")[1];
      setTorrentBase64(base64);
      try {
        const info = await api.checkTorrent(base64);
        setMagnetInfo(info);
        const allIds = getAllFileIds(info.files);
        setSelectedFiles(new Set(allIds));
      } catch (err: any) {
        setMagnetError(err.message || "Failed to parse torrent");
      } finally {
        setIsCheckingMagnet(false);
      }
    };
    reader.readAsDataURL(file);
  };

  // Detect magnet and check
  useEffect(() => {
    if (isTorrent) return; // Don't check URIs if we're doing a torrent upload

    const url = uris.trim().split("\n")[0]?.trim();

    if (url && url.startsWith("magnet:")) {
      setIsMagnet(true);
      setResolution(null);
      const timer = setTimeout(() => {
        checkMagnet(url);
      }, 500);
      return () => clearTimeout(timer);
    } else {
      setIsMagnet(false);
      if (!isTorrent) {
        setMagnetInfo(null);
        setMagnetError(null);
        setSelectedFiles(new Set());
      }

      // Regular URL resolution
      if (url && url.startsWith("http")) {
        const timer = setTimeout(async () => {
          try {
            const res = await api.resolveUrl(url);
            setResolution(res);
          } catch (err) {
            setResolution(null);
          }
        }, 500);
        return () => clearTimeout(timer);
      } else {
        setResolution(null);
      }
    }
  }, [uris, checkMagnet, isTorrent]);

  const handleSubmit = async () => {
    if ((isMagnet || isTorrent) && magnetInfo) {
      // Magnet/Torrent download
      try {
        await api.downloadMagnet({
          magnet: isMagnet ? uris.trim().split("\n")[0] : "",
          torrentBase64: torrentBase64 || "",
          source: magnetInfo.source,
          magnetId: magnetInfo.magnetId,
          name: magnetInfo.name,
          selectedFiles: Array.from(selectedFiles),
          destination: defaultRemote || undefined,
          files: flattenFiles(magnetInfo.files),
        });
        toast.success(isTorrent ? "Torrent download started" : "Magnet download started");
        navigate(tasksLinkOptions("active"));
      } catch (err: any) {
        toast.error(`Failed to start download: ${err.message}`);
      }
    } else {
      // Regular download
      const uriList = uris.split("\n").filter((u) => u.trim());
      if (uriList.length === 0) return;

      create.mutate(
        {
          url: uriList[0],
          filename: filename || undefined,
          destination: defaultRemote || undefined,
        },
        {
          onSuccess: () => navigate(tasksLinkOptions("active")),
        },
      );
    }
  };

  const selectAllFiles = () => {
    if (magnetInfo) {
      setSelectedFiles(new Set(getAllFileIds(magnetInfo.files)));
    }
  };

  const deselectAllFiles = () => {
    setSelectedFiles(new Set());
  };

  const selectedSize = magnetInfo
    ? getSelectedSize(magnetInfo.files, selectedFiles)
    : 0;

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
            onPress={handleSubmit}
            isDisabled={
              !uris.trim() && !isTorrent ||
              create.isPending ||
              isCheckingMagnet ||
              ((isMagnet || isTorrent) && selectedFiles.size === 0)
            }
            isPending={create.isPending || isCheckingMagnet}
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
              <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                Download URL
              </Label>
              <div className="relative group">
                <TextArea
                  placeholder="Paste HTTP, FTP or Magnet links here..."
                  value={uris}
                  onChange={(e) => setUris(e.target.value)}
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
                        setMagnetInfo(null);
                      }}
                    >
                      Clear
                    </Button>
                  )}
                </div>
              )}

              {/* Regular URL resolution */}
              {!isMagnet && !isTorrent && resolution && (
                <div
                  className={`mt-2 p-4 rounded-2xl flex items-center gap-3 border ${resolution.supported ? "bg-success/5 border-success/20 text-success" : "bg-warning/5 border-warning/20 text-warning"}`}
                >
                  <IconNodesDown className="w-5 h-5" />
                  <div className="flex-1">
                    <p className="text-xs font-bold">
                      {resolution.supported
                        ? `Supported by ${resolution.provider}`
                        : "No specific provider support, will try direct download"}
                    </p>
                  </div>
                </div>
              )}

              {/* Magnet error */}
              {magnetError && (
                <div className="mt-2 p-4 rounded-2xl bg-danger/5 border border-danger/20 text-danger">
                  <p className="text-xs font-bold">{magnetError}</p>
                </div>
              )}
            </div>
          </div>

          {/* File Selection (Magnet or Torrent) */}
          {(isMagnet || isTorrent) && magnetInfo && (
            <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center justify-between mb-6">
                <div className="flex-1 min-w-0 mr-4">
                  <h3
                    className="font-bold text-lg truncate"
                    title={magnetInfo.name}
                  >
                    {magnetInfo.name}
                  </h3>
                  <div className="flex items-center gap-2 text-xs text-muted font-black uppercase tracking-widest mt-1">
                    <span>{formatBytes(magnetInfo.size)}</span>
                    <span>•</span>
                    <span>{getAllFileIds(magnetInfo.files).length} files</span>
                  </div>
                </div>
                <Chip
                  color={magnetInfo.cached ? "success" : "default"}
                  variant="soft"
                  size="sm"
                  className="font-black uppercase tracking-widest"
                >
                  {magnetInfo.cached
                    ? "Cached"
                    : magnetInfo.source === "aria2"
                      ? "P2P"
                      : "Not Cached"}
                </Chip>
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
              <div className="max-h-[500px] overflow-y-auto rounded-2xl border border-border bg-default/5 custom-scrollbar">
                <FileTree
                  files={magnetInfo.files}
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

          {/* Loading state for magnet/torrent check */}
          {(isMagnet || isTorrent) && isCheckingMagnet && (
            <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center justify-center gap-3 py-8 flex-col">
                <div className="animate-spin rounded-full h-8 w-8 border-3 border-accent border-t-transparent" />
                <span className="text-sm text-muted font-bold uppercase tracking-widest">
                  {checkStatus}
                </span>
              </div>
            </div>
          )}
        </div>

        {/* Right Column - Options */}
        <div className="lg:col-span-5 space-y-6">
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm space-y-6">
            {/* Filename (only for non-magnet/torrent) */}
            {!isMagnet && !isTorrent && (
              <div className="flex flex-col gap-2">
                <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                  Filename (Optional)
                </Label>
                <Input
                  placeholder="original-name.zip"
                  value={filename}
                  onChange={(e) => setFilename(e.target.value)}
                  className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
                />
              </div>
            )}

            <div className="flex flex-col gap-2">
              <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                Upload Target
              </Label>
              <Input
                placeholder="e.g. gdrive:/downloads"
                value={defaultRemote}
                onChange={(e) => setDefaultRemote(e.target.value)}
                className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
              />
              <p className="text-[10px] text-muted font-medium px-1 leading-relaxed">
                Enter a remote path to automatically offload files to the cloud
                after download completes.
              </p>
            </div>
          </div>

          {/* Magnet/Torrent source info */}
          {(isMagnet || isTorrent) && magnetInfo && (
            <div className="bg-background p-6 rounded-[32px] border border-border shadow-sm">
              <div className="flex items-center gap-4">
                <div
                  className={`w-12 h-12 rounded-2xl flex items-center justify-center shadow-lg ${
                    magnetInfo.source === "alldebrid"
                      ? "bg-success/10 text-success shadow-success/10"
                      : "bg-accent/10 text-accent shadow-accent/10"
                  }`}
                >
                  <span className="font-black text-lg">
                    {magnetInfo.source === "alldebrid" ? "AD" : "P2P"}
                  </span>
                </div>
                <div>
                  <p className="font-bold text-sm">
                    {magnetInfo.source === "alldebrid"
                      ? "AllDebrid High-Speed"
                      : "BitTorrent Network"}
                  </p>
                  <p className="text-xs text-muted font-medium mt-0.5">
                    {magnetInfo.source === "alldebrid"
                      ? "Downloading from AllDebrid cache via high-speed HTTP."
                      : "Downloading from BitTorrent swarm using peers and seeds."}
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

// Helper to flatten nested files for API request
function flattenFiles(files: MagnetFile[]): any[] {
  const result: any[] = [];

  function traverse(items: MagnetFile[]) {
    for (const file of items) {
      if (!file.isFolder) {
        result.push({
          id: file.id,
          name: file.name,
          path: file.path,
          size: file.size,
          link: file.link,
          index: file.index,
        });
      }
      if (file.children) {
        traverse(file.children);
      }
    }
  }

  traverse(files);
  return result;
}
