import { Checkbox, ListBox } from "@heroui/react";
import React from "react";
import IconArchive from "~icons/gravity-ui/archive";
import IconPulse from "~icons/gravity-ui/pulse";
import { useTasksByStatus } from "../../hooks/useEngine";
import { cn } from "../../lib/utils";
import { useSettingsStore } from "../../store/useSettingsStore";
import { DownloadCard } from "./DownloadCard";
import type { TaskStatus } from "../../routes/tasks";

interface TaskListProps {
  status: TaskStatus;
}

export const TaskList: React.FC<TaskListProps> = ({ status }) => {
  const { data: tasks = [], isLoading } = useTasksByStatus(status);
  const {
    viewMode,
    searchQuery,
    isSelectionMode,
    selectedGids,
    toggleGidSelection,
  } = useSettingsStore();

  const filteredTasks = React.useMemo(() => {
    let t = tasks;

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      t = t.filter((task) => {
        const fileName = task.filename || task.id;
        return fileName.toLowerCase().includes(query);
      });
    }

    return t;
  }, [tasks, searchQuery]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4">
        {isLoading && filteredTasks.length === 0 ? (
          <div className="flex justify-center py-12">
            <IconPulse className="w-8 h-8 text-muted animate-pulse" />
          </div>
        ) : filteredTasks.length === 0 ? (
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
            items={filteredTasks}
            className={cn(
              "outline-none border-none p-0 bg-transparent",
              viewMode === "grid"
                ? "grid grid-cols-1 lg:grid-cols-2 gap-4"
                : "flex flex-col border border-border rounded-[32px] bg-muted-background/20 overflow-hidden divide-y divide-border",
            )}
          >
            {(task) => (
              <ListBox.Item
                id={task.id}
                key={task.id}
                textValue={task.filename || task.id}
                className="outline-none focus:outline-none bg-transparent p-0 w-full"
              >
                <div className="flex items-center gap-4 group w-full cursor-default">
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
              </ListBox.Item>
            )}
          </ListBox>
        )}
      </div>
    </div>
  );
};
