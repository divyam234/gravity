import { Checkbox, Dropdown, ListBox } from "@heroui/react";
import React, { useId } from "react";
import IconArchive from "~icons/gravity-ui/archive";
import IconCopy from "~icons/gravity-ui/copy";
import IconPause from "~icons/gravity-ui/pause";
import IconPlay from "~icons/gravity-ui/play";
import IconPulse from "~icons/gravity-ui/pulse";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useDownloadActions, useDownloads } from "../../hooks/useDownloads";
import { cn } from "../../lib/utils";
import { useSettingsStore } from "../../store/useSettingsStore";
import { DownloadCard } from "./DownloadCard";

interface TaskListProps {
  status: string;
}

export const TaskList: React.FC<TaskListProps> = ({ status }) => {
  const { data, isLoading } = useDownloads({ status: [status] });
  const { pause, resume, remove } = useDownloadActions();
  const baseId = useId();
  const {
    viewMode,
    searchQuery,
    isSelectionMode,
    selectedGids,
    toggleGidSelection,
  } = useSettingsStore();

  const tasks = React.useMemo(() => {
    let t = data?.data || [];

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      t = t.filter((task) => {
        const fileName = task.filename || task.id;
        return fileName.toLowerCase().includes(query);
      });
    }

    return t;
  }, [data, searchQuery]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4">
        {isLoading && tasks.length === 0 ? (
          <div className="flex justify-center py-12">
            <IconPulse className="w-8 h-8 text-muted animate-pulse" />
          </div>
        ) : tasks.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted">
            <IconArchive className="w-12 h-12 mb-4 opacity-50" />
            <p className="text-lg font-medium">No tasks found</p>
            <p className="text-sm">
              {searchQuery
                ? "Try a different search query"
                : "Add a new download to get started"}
            </p>
          </div>
        ) : (
          <ListBox
            aria-label="Tasks"
            className={cn(
              "outline-none border-none p-0 bg-transparent",
              viewMode === "grid"
                ? "grid grid-cols-1 lg:grid-cols-2 gap-4"
                : "flex flex-col border border-border rounded-[32px] bg-muted-background/20 overflow-hidden divide-y divide-border",
            )}
          >
            {tasks.map((task) => (
              <ListBox.Item
                key={task.id}
                id={task.id}
                textValue={task.filename || task.id}
                className="outline-none focus:outline-none bg-transparent p-0 w-full"
              >
                {/* @ts-ignore - contextMenu is valid at runtime */}
                <Dropdown trigger="contextMenu">
                  <div className="flex items-center gap-4 group w-full">
                    {isSelectionMode && (
                      <div className="pl-6">
                        <Checkbox
                          isSelected={selectedGids.has(task.id)}
                          onChange={() => toggleGidSelection(task.id)}
                        />
                      </div>
                    )}
                    <div className="flex-1 min-w-0 w-full">
                      <DownloadCard task={task} variant={viewMode} />
                    </div>
                  </div>

                  <Dropdown.Popover className="min-w-[200px] p-1 bg-background border border-border rounded-xl shadow-xl">
                    <Dropdown.Menu
                      onAction={(key) => {
                        const action = String(key).replace(`${baseId}-`, "");
                        if (action === "pause") pause.mutate(task.id);
                        if (action === "resume") resume.mutate(task.id);
                        if (action === "remove") remove.mutate({ id: task.id });
                        if (action === "copy") {
                          navigator.clipboard.writeText(task.url);
                        }
                      }}
                    >
                      <Dropdown.Item
                        id={
                          task.status === "paused"
                            ? `${baseId}-resume`
                            : `${baseId}-pause`
                        }
                        textValue={
                          task.status === "paused" ? "Resume" : "Pause"
                        }
                        className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 cursor-pointer outline-none flex items-center gap-2"
                      >
                        {task.status === "paused" ? (
                          <IconPlay className="w-4 h-4 text-success" />
                        ) : (
                          <IconPause className="w-4 h-4 text-warning" />
                        )}
                        <span className="text-sm font-medium">
                          {task.status === "paused" ? "Resume" : "Pause"}
                        </span>
                      </Dropdown.Item>

                      <Dropdown.Item
                        id={`${baseId}-copy`}
                        textValue="Copy Link"
                        className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 cursor-pointer outline-none flex items-center gap-2"
                      >
                        <IconCopy className="w-4 h-4 text-accent" />
                        <span className="text-sm font-medium">Copy Link</span>
                      </Dropdown.Item>

                      <Dropdown.Item
                        id={`${baseId}-remove`}
                        textValue="Remove"
                        variant="danger"
                        className="px-3 py-2 rounded-lg data-[hover=true]:bg-danger/10 text-danger cursor-pointer outline-none flex items-center gap-2"
                      >
                        <IconTrashBin className="w-4 h-4" />
                        <span className="text-sm font-medium">Remove</span>
                      </Dropdown.Item>
                    </Dropdown.Menu>
                  </Dropdown.Popover>
                </Dropdown>
              </ListBox.Item>
            ))}
          </ListBox>
        )}
      </div>
    </div>
  );
};
