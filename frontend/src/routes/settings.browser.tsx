import {
  Button,
  Card,
  Label,
  ScrollShadow,
  Input,
  TextField,
  Select,
  ListBox,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  useEngineActions,
  globalOptionOptions,
  useGlobalOption,
} from "../hooks/useEngine";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconArrowsRotate from "~icons/gravity-ui/arrows-rotate-right";
import type { Key } from "react-aria-components";
import { toast } from "sonner";
import { api } from "@/lib/api";

export const Route = createFileRoute("/settings/browser")({
  component: BrowserSettingsPage,
  loader: async ({ context: { queryClient } }) => {
    queryClient.prefetchQuery(globalOptionOptions());
  },
});

function BrowserSettingsPage() {
  const navigate = useNavigate();
  const { data: options } = useGlobalOption();
  const { changeGlobalOption } = useEngineActions();

  // VFS States
  const [vfsCacheMode, setVfsCacheMode] = useState<Key>("off");
  const [vfsCacheMaxSize, setVfsCacheMaxSize] = useState("10G");
  const [vfsCacheMaxAge, setVfsCacheMaxAge] = useState("1h");
  const [vfsWriteBack, setVfsWriteBack] = useState("5s");
  const [vfsReadChunkSize, setVfsReadChunkSize] = useState("128M");
  const [vfsReadChunkSizeLimit, setVfsReadChunkSizeLimit] = useState("off");
  const [vfsReadAhead, setVfsReadAhead] = useState("128M");
  const [vfsDirCacheTime, setVfsDirCacheTime] = useState("5m");
  const [vfsPollInterval, setVfsPollInterval] = useState("1m");
  const [vfsReadChunkStreams, setVfsReadChunkStreams] = useState("0");

  // VFS Flags
  const [isValid, setIsValid] = useState(true);

  useEffect(() => {
    // Load VFS Options
    if (options?.vfsCacheMode) setVfsCacheMode(options.vfsCacheMode);
    if (options?.vfsCacheMaxSize) setVfsCacheMaxSize(options.vfsCacheMaxSize);
    if (options?.vfsCacheMaxAge) setVfsCacheMaxAge(options.vfsCacheMaxAge);
    if (options?.vfsWriteBack) setVfsWriteBack(options.vfsWriteBack);
    if (options?.vfsReadChunkSize)
      setVfsReadChunkSize(options.vfsReadChunkSize);
    if (options?.vfsReadChunkSizeLimit)
      setVfsReadChunkSizeLimit(options.vfsReadChunkSizeLimit);
    if (options?.vfsReadAhead) setVfsReadAhead(options.vfsReadAhead);
    if (options?.vfsDirCacheTime) setVfsDirCacheTime(options.vfsDirCacheTime);
    if (options?.vfsPollInterval) setVfsPollInterval(options.vfsPollInterval);
    if (options?.vfsReadChunkStreams)
      setVfsReadChunkStreams(options.vfsReadChunkStreams);
  }, [options]);

  const validateDuration = (val: string) => {
    if (val.toLowerCase() === "unlimited" || val === "off" || val === "0")
      return true;
    return /^\d+[smhd]$/i.test(val);
  };

  const validateSize = (val: string) => {
    if (val === "off" || val === "0") return true;
    return /^\d+[KMGTP]?$/i.test(val);
  };

  const handleVfsChange = (
    key: string,
    val: string | boolean,
    validator?: (v: string) => boolean,
  ) => {
    const stringVal = val.toString();
    const valid =
      typeof val === "boolean" ? true : validator ? validator(stringVal) : true;

    // Update Local State
    const setters: Record<string, Function> = {
      vfsCacheMode: setVfsCacheMode,
      vfsCacheMaxSize: setVfsCacheMaxSize,
      vfsCacheMaxAge: setVfsCacheMaxAge,
      vfsWriteBack: setVfsWriteBack,
      vfsReadChunkSize: setVfsReadChunkSize,
      vfsReadChunkSizeLimit: setVfsReadChunkSizeLimit,
      vfsReadAhead: setVfsReadAhead,
      vfsDirCacheTime: setVfsDirCacheTime,
      vfsPollInterval: setVfsPollInterval,
      vfsReadChunkStreams: setVfsReadChunkStreams,
    };

    if (setters[key]) setters[key](val);

    if (valid) {
      changeGlobalOption.mutate({ [key]: stringVal });
    }
  };

  const restartMutation = useMutation({
    mutationFn: () => api.restartEngine(),
    onSuccess: () => {
      toast.success("VFS Engine restarted");
    },
    onError: (err: any) => {
      toast.error("Failed to restart VFS: " + err.message);
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
          <h2 className="text-2xl font-bold tracking-tight">Browser</h2>
          <p className="text-xs text-muted">
            File browsing performance & display
          </p>
        </div>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        <ScrollShadow className="h-full">
          <div className="max-w-4xl mx-auto p-8 space-y-10">
            {/* VFS Options */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-primary rounded-full" />
                <h3 className="text-lg font-bold">VFS Cache (Streaming)</h3>
              </div>

              <div className="grid gap-4">
                <Card className="p-6 bg-background/50 border-border">
                  <div className="flex items-center justify-between">
                    <div>
                      <Label className="text-sm font-bold">Restart VFS</Label>
                      <p className="text-xs text-muted mt-0.5">
                        Manually restart the storage engine to apply deep
                        configuration changes.
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      className="rounded-xl font-bold hover:bg-accent/10"
                      onPress={() => restartMutation.mutate()}
                      isPending={restartMutation.isPending}
                    >
                      <IconArrowsRotate className="w-4 h-4 mr-2" />
                      Restart
                    </Button>
                  </div>
                </Card>

                <Card className="p-6 bg-background/50 border-border">
                  <div className="flex items-center justify-between">
                    <div className="flex-1 mr-8">
                      <Label className="text-sm font-bold">
                        VFS Cache Mode
                      </Label>
                      <p className="text-xs text-muted mt-0.5">
                        "full" is required for seeking in videos and random
                        access.
                      </p>
                    </div>

                    <Select
                      className="w-48"
                      value={vfsCacheMode}
                      onChange={(key) =>
                        handleVfsChange("vfsCacheMode", key as string)
                      }
                    >
                      <Select.Trigger>
                        <Select.Value className="text-sm font-bold" />
                        <Select.Indicator />
                      </Select.Trigger>
                      <Select.Popover>
                        <ListBox>
                          <ListBox.Item id="off" textValue="Off">
                            Off
                          </ListBox.Item>
                          <ListBox.Item id="minimal" textValue="Minimal">
                            Minimal
                          </ListBox.Item>
                          <ListBox.Item id="writes" textValue="Writes">
                            Writes
                          </ListBox.Item>
                          <ListBox.Item id="full" textValue="Full">
                            Full
                          </ListBox.Item>
                        </ListBox>
                      </Select.Popover>
                    </Select>
                  </div>
                </Card>

                {/* Sizes Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Card className="p-6 bg-background/50 border-border">
                    <Label className="text-sm font-bold">Max Cache Size</Label>
                    <p className="text-xs text-muted mb-4">
                      Max disk space for cache (e.g. 10G, 100G)
                    </p>
                    <TextField>
                      <Input
                        value={vfsCacheMaxSize}
                        onChange={(e) =>
                          handleVfsChange(
                            "vfsCacheMaxSize",
                            e.target.value,
                            validateSize,
                          )
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none font-bold px-4"
                      />
                    </TextField>
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <Label className="text-sm font-bold">Read Ahead</Label>
                    <p className="text-xs text-muted mb-4">
                      Bytes to read ahead in "full" mode (e.g. 128M)
                    </p>
                    <TextField>
                      <Input
                        value={vfsReadAhead}
                        onChange={(e) =>
                          handleVfsChange(
                            "vfsReadAhead",
                            e.target.value,
                            validateSize,
                          )
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none font-bold px-4"
                      />
                    </TextField>
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <Label className="text-sm font-bold">Read Chunk Size</Label>
                    <p className="text-xs text-muted mb-4">
                      Initial chunk size for reads (e.g. 128M)
                    </p>
                    <TextField>
                      <Input
                        value={vfsReadChunkSize}
                        onChange={(e) =>
                          handleVfsChange(
                            "vfsReadChunkSize",
                            e.target.value,
                            validateSize,
                          )
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none font-bold px-4"
                      />
                    </TextField>
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <Label className="text-sm font-bold">
                      Chunk Size Limit
                    </Label>
                    <p className="text-xs text-muted mb-4">
                      Max doubled chunk size (e.g. 512M, or "off")
                    </p>
                    <TextField>
                      <Input
                        value={vfsReadChunkSizeLimit}
                        onChange={(e) =>
                          handleVfsChange(
                            "vfsReadChunkSizeLimit",
                            e.target.value,
                            validateSize,
                          )
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none font-bold px-4"
                      />
                    </TextField>
                  </Card>
                </div>

                {/* Durations Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Card className="p-6 bg-background/50 border-border">
                    <Label className="text-sm font-bold">Dir Cache Time</Label>
                    <p className="text-xs text-muted mb-4">
                      How long to cache directory listings (e.g. 5m)
                    </p>
                    <TextField>
                      <Input
                        value={vfsDirCacheTime}
                        onChange={(e) =>
                          handleVfsChange(
                            "vfsDirCacheTime",
                            e.target.value,
                            validateDuration,
                          )
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none font-bold px-4"
                      />
                    </TextField>
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <Label className="text-sm font-bold">Max Cache Age</Label>
                    <p className="text-xs text-muted mb-4">
                      Max age of cached files (e.g. 1h, 24h)
                    </p>
                    <TextField>
                      <Input
                        value={vfsCacheMaxAge}
                        onChange={(e) =>
                          handleVfsChange(
                            "vfsCacheMaxAge",
                            e.target.value,
                            validateDuration,
                          )
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none font-bold px-4"
                      />
                    </TextField>
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <Label className="text-sm font-bold">
                      Write Back Delay
                    </Label>
                    <p className="text-xs text-muted mb-4">
                      Delay before uploading changed files (e.g. 5s)
                    </p>
                    <TextField>
                      <Input
                        value={vfsWriteBack}
                        onChange={(e) =>
                          handleVfsChange(
                            "vfsWriteBack",
                            e.target.value,
                            validateDuration,
                          )
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none font-bold px-4"
                      />
                    </TextField>
                  </Card>

                  <Card className="p-6 bg-background/50 border-border">
                    <Label className="text-sm font-bold">Poll Interval</Label>
                    <p className="text-xs text-muted mb-4">
                      How often to poll for changes (e.g. 1m)
                    </p>
                    <TextField>
                      <Input
                        value={vfsPollInterval}
                        onChange={(e) =>
                          handleVfsChange(
                            "vfsPollInterval",
                            e.target.value,
                            validateDuration,
                          )
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none font-bold px-4"
                      />
                    </TextField>
                  </Card>
                </div>

                {/* Others */}
                <Card className="p-6 bg-background/50 border-border">
                  <div className="flex items-center justify-between">
                    <div className="flex-1 mr-8">
                      <Label className="text-sm font-bold">
                        Read Chunk Streams
                      </Label>
                      <p className="text-xs text-muted mt-0.5">
                        Number of parallel streams to use for reading chunks (0
                        to disable).
                      </p>
                    </div>

                    <TextField className="w-48">
                      <Input
                        value={vfsReadChunkStreams}
                        onChange={(e) =>
                          handleVfsChange("vfsReadChunkStreams", e.target.value)
                        }
                        className="h-11 bg-default/10 rounded-2xl border-none text-center font-bold"
                      />
                    </TextField>
                  </div>
                </Card>
              </div>
            </section>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
