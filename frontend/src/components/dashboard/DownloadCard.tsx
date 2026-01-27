import { Button, Card, Chip, Kbd, Tooltip } from "@heroui/react";
import { Link } from "@tanstack/react-router";
import type React from "react";
import IconListUl from "~icons/gravity-ui/list-ul";
import IconPause from "~icons/gravity-ui/pause";
import IconPlay from "~icons/gravity-ui/play";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useEngineActions } from "../../hooks/useEngine";
import { cn, formatBytes, formatTime } from "../../lib/utils";
import { ProgressBar } from "../ui/ProgressBar";
import { StatusChip } from "../ui/StatusChip";
import type { components } from "../../gen/api";

type Download = components["schemas"]["model.Download"];

interface DownloadCardProps {
  task: Download;
  variant?: "list" | "grid";
}

export const DownloadCard: React.FC<DownloadCardProps> = ({
  task,
  variant = "list",
}) => {
  const { pause, unpause, remove } = useEngineActions();

  const progress = (task.size || 0) > 0 ? ((task.downloaded || 0) / (task.size || 1)) * 100 : 0;

  const isPaused = task.status === "paused";
  const isActive = task.status === "active";
  const isError = task.status === "error";
  const isComplete = task.status === "complete";
  const isUploading = task.status === "uploading";

  // Calculate uploaded bytes from progress percentage
  const uploadedBytes = (task.size || 0) > 0 ? Math.floor(((task.uploadProgress || 0) / 100) * (task.size || 0)) : 0;

  // Effective display values
  const effectiveProgress = isUploading ? (task.uploadProgress || 0) : progress;
  const effectiveSpeed = isUploading ? (task.uploadSpeed || 0) : (task.speed || 0);
  const isActionable = isActive || isPaused;

  const handleRemove = () => {
    if (task.id) {
        remove.mutate({ params: { path: { id: task.id } } });
    }
  };

  const statusColor = isError
    ? "danger"
    : isUploading
      ? "default"
      : isComplete
        ? "success"
        : "accent";

  const progressClassName = isUploading ? "h-2 [&>div]:bg-cyan-500" : "h-2";

  if (variant === "list") {
    return (
      <div className="w-full flex items-center gap-6 py-4 px-6 hover:bg-default/5 transition-colors rounded-xl group/item min-h-[72px]">
        <Link
          to="/task/$gid"
          params={{ gid: task.id || "" }}
          className="flex-1 min-w-0 cursor-pointer outline-none flex items-center gap-8"
        >
          <div className="flex-1 min-w-0">
            <h3
              className="text-base font-bold truncate group-hover/item:text-accent transition-colors leading-tight"
              title={task.filename || task.id}
            >
              {task.filename || task.id}
            </h3>
            <div className="flex items-center gap-3 mt-1.5">
              <StatusChip
                status={task.status || "waiting"}
                className="h-5 text-[10px] px-2"
              />
              {task.metadataFetching && (
                <Chip
                  color="warning"
                  size="sm"
                  variant="soft"
                  className="h-5 text-[10px] px-2 font-black uppercase tracking-widest animate-pulse"
                >
                  Metadata
                </Chip>
              )}
              <span className="text-xs text-muted font-bold">
                {isUploading
                  ? `${formatBytes(uploadedBytes)} / ${formatBytes(task.size || 0)}`
                  : `${formatBytes(task.downloaded || 0)} / ${formatBytes(task.size || 0)}`}
              </span>
              {task.isMagnet && task.totalFiles && (
                <span className="text-[10px] text-muted font-black uppercase tracking-widest bg-default/10 px-1.5 py-0.5 rounded-md">
                  {task.filesComplete || 0} / {task.totalFiles} files
                </span>
              )}
            </div>
            {isError && task.error && (
              <p className="text-[10px] text-danger font-bold mt-1 uppercase tracking-tight truncate max-w-[300px]">
                {task.error}
              </p>
            )}
          </div>

          <div className="w-48 shrink-0 hidden sm:block">
            <ProgressBar
              value={effectiveProgress}
              size="sm"
              color={statusColor}
              className={progressClassName}
            />
          </div>

          <div className="w-32 shrink-0 hidden md:flex flex-col items-end gap-0.5">
            <span
              className={cn(
                "text-xs font-black",
                isUploading ? "text-cyan-500" : "text-success/80",
              )}
            >
              {isUploading ? "↑" : "↓"} {formatBytes(effectiveSpeed)}/s
            </span>
            {isActive && task.isMagnet && task.magnetSource === "aria2" && (
              <span className="text-[9px] font-black uppercase tracking-tighter text-muted">
                S: <span className="text-success">{task.seeders || 0}</span> P: <span className="text-foreground">{task.peers || 0}</span>
              </span>
            )}
            {!isUploading && isActive && (
              <span className="text-[10px] text-muted font-bold uppercase tracking-wider">
                {formatTime(task.eta || 0)}
              </span>
            )}
          </div>
        </Link>

        <div className="flex items-center gap-2 shrink-0 opacity-0 group-hover/item:opacity-100 transition-opacity ml-auto">
          {isActive && task.id && (
            <Button
              isIconOnly
              size="sm"
              variant="ghost"
              onPress={() => pause.mutate({ params: { path: { id: task.id! } } })}
              className="h-8 w-8 min-w-0"
            >
              <IconPause className="w-4 h-4 text-warning" />
            </Button>
          )}
          {isPaused && task.id && (
            <Button
              isIconOnly
              size="sm"
              variant="ghost"
              onPress={() => unpause.mutate({ params: { path: { id: task.id! } } })}
              className="h-8 w-8 min-w-0"
            >
              <IconPlay className="w-4 h-4 text-success" />
            </Button>
          )}
          <Button
            isIconOnly
            size="sm"
            variant="ghost"
            className="text-danger h-8 w-8 min-w-0"
            onPress={handleRemove}
          >
            <IconTrashBin className="w-4 h-4" />
          </Button>
        </div>
      </div>
    );
  }

  return (
    <Card className="w-full shadow-sm border-border hover:shadow-md transition-shadow">
      <Card.Content className="p-4 space-y-4">
        <div className="flex justify-between items-start gap-4">
          <Link
            to="/task/$gid"
            params={{ gid: task.id || "" }}
            className="flex-1 min-w-0 cursor-pointer group outline-none"
          >
            <h3
              className="text-lg font-bold truncate group-hover:text-accent transition-colors leading-tight"
              title={task.filename || task.id}
            >
              {task.filename || task.id}
            </h3>
            <div className="flex items-center gap-2 mt-1.5">
              <StatusChip status={task.status || "waiting"} />
              {task.metadataFetching && (
                <Chip
                  color="warning"
                  size="sm"
                  variant="soft"
                  className="h-5 text-[10px] px-2 font-black uppercase tracking-widest animate-pulse"
                >
                  Metadata
                </Chip>
              )}
              <span className="text-sm text-muted font-medium">
                {isUploading
                  ? `${formatBytes(uploadedBytes)} / ${formatBytes(task.size || 0)}`
                  : `${formatBytes(task.downloaded || 0)} / ${formatBytes(task.size || 0)}`}
              </span>
              {task.isMagnet && task.totalFiles && (
                <span className="text-[10px] text-muted font-black uppercase tracking-widest bg-default/10 px-1.5 py-0.5 rounded-md ml-auto">
                  {task.filesComplete || 0} / {task.totalFiles} files
                </span>
              )}
            </div>
            {isError && task.error && (
              <p className="text-[10px] text-danger font-bold mt-1 uppercase tracking-tight">
                {task.error}
              </p>
            )}
          </Link>

          <div className="flex items-center gap-1">
            <Tooltip>
              <Tooltip.Trigger>
                <Link to="/task/$gid" params={{ gid: task.id || "" }}>
                  <Button isIconOnly size="sm" variant="ghost">
                    <IconListUl className="w-4.5 h-4.5" />
                  </Button>
                </Link>
              </Tooltip.Trigger>
              <Tooltip.Content className="p-2 text-xs">
                Files & Details
              </Tooltip.Content>
            </Tooltip>

            {isActionable && task.id && (
              <Tooltip>
                <Tooltip.Trigger>
                  <Button
                    isIconOnly
                    size="sm"
                    variant="ghost"
                    onPress={() =>
                      isActive ? pause.mutate({ params: { path: { id: task.id! } } }) : unpause.mutate({ params: { path: { id: task.id! } } })
                    }
                  >
                    {isActive ? (
                      <IconPause className="w-4.5 h-4.5 text-warning" />
                    ) : (
                      <IconPlay className="w-4.5 h-4.5 text-success" />
                    )}
                  </Button>
                </Tooltip.Trigger>
                <Tooltip.Content className="p-2 text-xs flex items-center gap-2">
                  {isActive ? "Pause" : "Resume"}{" "}
                  <Kbd>{isActive ? "P" : "R"}</Kbd>
                </Tooltip.Content>
              </Tooltip>
            )}

            <Tooltip>
              <Tooltip.Trigger>
                <Button
                  isIconOnly
                  size="sm"
                  variant="ghost"
                  className="text-danger"
                  onPress={handleRemove}
                >
                  <IconTrashBin className="w-4.5 h-4.5" />
                </Button>
              </Tooltip.Trigger>
              <Tooltip.Content className="p-2 text-xs">Remove</Tooltip.Content>
            </Tooltip>
          </div>
        </div>

        <ProgressBar
          value={effectiveProgress}
          size="md"
          color={statusColor}
          className={progressClassName}
          showValueLabel={!!effectiveProgress && effectiveProgress > 0}
        />

        <div className="flex justify-between items-center text-sm text-muted bg-muted-background/50 p-2 rounded-xl border border-border/50">
          <div className="flex gap-4">
            <span
              className={cn(
                "flex items-center gap-1.5 font-bold",
                isUploading ? "text-cyan-500" : "text-success/80",
              )}
            >
              <div
                className={cn(
                  "w-1.5 h-1.5 rounded-full",
                  isUploading ? "bg-cyan-500" : "bg-success",
                )}
              />
              {isUploading ? "↑" : "↓"} {formatBytes(effectiveSpeed)}/s
            </span>
            {isActive && task.isMagnet && task.magnetSource === "aria2" && (
              <span className="flex items-center gap-2 text-[10px] font-black uppercase tracking-tighter text-muted">
                S: <span className="text-success">{task.seeders || 0}</span> 
                P: <span className="text-foreground">{task.peers || 0}</span>
              </span>
            )}
          </div>

          <div className="flex gap-4 font-medium items-center">
            {!isUploading && isActive && (
              <span>ETA: {formatTime(task.eta || 0)}</span>
            )}
          </div>
        </div>
      </Card.Content>
    </Card>
  );
};