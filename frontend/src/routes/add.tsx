import {
  Button,
  Input,
  Label,
  TextArea,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import { useDownloadActions } from "../hooks/useDownloads";
import { api } from "../lib/api";
import { useSettingsStore } from "../store/useSettingsStore";
import { tasksLinkOptions } from "./tasks";

export const Route = createFileRoute("/add")({
  component: AddDownloadPage,
});

function AddDownloadPage() {
  const navigate = useNavigate();
  const { rcloneTargetRemote, setRcloneTargetRemote } = useSettingsStore();

  const [uris, setUris] = useState("");
  const [filename, setFilename] = useState("");
  const [resolution, setResolution] = useState<{provider: string, supported: boolean} | null>(null);

  const { create } = useDownloadActions();

  // Resolve preview when URL changes
  useEffect(() => {
    const firstUrl = uris.split("\n")[0]?.trim();
    if (firstUrl && firstUrl.startsWith("http")) {
      const timer = setTimeout(async () => {
        try {
          const res = await api.resolveUrl(firstUrl);
          setResolution(res);
        } catch (err) {
          setResolution(null);
        }
      }, 500);
      return () => clearTimeout(timer);
    } else {
      setResolution(null);
    }
  }, [uris]);

  const handleSubmit = async () => {
    const uriList = uris.split("\n").filter((u) => u.trim());
    if (uriList.length === 0) return;

    create.mutate(
      {
        url: uriList[0], // For now, single URL
        filename: filename || undefined,
        destination: rcloneTargetRemote || undefined,
      },
      {
        onSuccess: () => navigate(tasksLinkOptions("active")),
      }
    );
  };

  return (
    <div className="max-w-5xl mx-auto space-y-6 pb-20">
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
            isDisabled={!uris.trim() || create.isPending}
            isPending={create.isPending}
          >
            Start
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <div className="lg:col-span-7 space-y-6">
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm">
            <div className="flex flex-col gap-3">
              <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                Download URL
              </Label>
              <TextArea
                placeholder="Paste HTTP, FTP or Magnet links here..."
                value={uris}
                onChange={(e) => setUris(e.target.value)}
                className="w-full p-6 bg-default/10 rounded-3xl text-sm border border-transparent focus:bg-default/15 focus:border-accent/30 transition-all outline-none min-h-[200px] leading-relaxed font-mono"
              />
              
              {resolution && (
                <div className={`mt-2 p-4 rounded-2xl flex items-center gap-3 border ${resolution.supported ? 'bg-success/5 border-success/20 text-success' : 'bg-warning/5 border-warning/20 text-warning'}`}>
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
            </div>
          </div>
        </div>

        <div className="lg:col-span-5 space-y-6">
          <div className="bg-background p-8 rounded-[32px] border border-border shadow-sm space-y-6">
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

            <div className="flex flex-col gap-2">
              <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                Upload Target
              </Label>
              <Input
                placeholder="e.g. gdrive:/downloads"
                value={rcloneTargetRemote}
                onChange={(e) => setRcloneTargetRemote(e.target.value)}
                className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
              />
              <p className="text-[10px] text-muted font-medium px-1">
                Enter an rclone remote path to automatically offload files to the cloud.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
