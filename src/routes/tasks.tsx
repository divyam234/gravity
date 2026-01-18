import { createFileRoute, linkOptions } from "@tanstack/react-router";
import { z } from "zod";
import { TaskList } from "../components/dashboard/TaskList";
import { TaskPageHeader } from "../components/dashboard/TaskPageHeader";
import {
  activeTasksOptions,
  stoppedTasksOptions,
  waitingTasksOptions,
} from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";
import { useSettingsStore } from "../store/useSettingsStore";

const taskSearchSchema = z.object({
  status: z.enum(["all", "active", "waiting", "stopped"]).default("all"),
});

export type TaskStatus = z.infer<typeof taskSearchSchema>["status"];

export const tasksLinkOptions = (status: TaskStatus) =>
  linkOptions({
    to: "/tasks",
    search: { status },
  });

export const Route = createFileRoute("/tasks")({
  validateSearch: (search) => taskSearchSchema.parse(search),
  loaderDeps: ({ search: { status } }) => ({ status }),
  loader: async ({ context: { queryClient }, deps: { status } }) => {
    const { rpcUrl, pollingInterval } = useSettingsStore.getState();
    if (!rpcUrl) return;

    const promises = [];
    if (status === "all" || status === "active") {
      promises.push(
        queryClient.ensureQueryData(
          activeTasksOptions(rpcUrl, pollingInterval),
        ),
      );
    }
    if (status === "all" || status === "waiting") {
      promises.push(
        queryClient.ensureQueryData(
          waitingTasksOptions(rpcUrl, pollingInterval, 0, 50),
        ),
      );
    }
    if (status === "all" || status === "stopped") {
      promises.push(
        queryClient.ensureQueryData(
          stoppedTasksOptions(rpcUrl, pollingInterval, 0, 50),
        ),
      );
    }

    await Promise.all(promises);
  },
  component: TasksPage,
});

function TasksPage() {
  const { status } = Route.useSearch();
  useNotifications();

  const titles: Record<string, string> = {
    active: "Active",
    waiting: "Queued",
    stopped: "Finished",
  };

  const colors: Record<string, string> = {
    active: "text-success",
    waiting: "text-warning",
    stopped: "text-danger",
  };

  return (
    <div className="space-y-6 px-1">
      <TaskPageHeader title={titles[status]} titleColor={colors[status]} />
      <TaskList status={status} />
    </div>
  );
}
