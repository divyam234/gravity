import { createFileRoute, linkOptions } from "@tanstack/react-router";
import { z } from "zod";
import { TaskList } from "../components/dashboard/TaskList";
import { TaskPageHeader } from "../components/dashboard/TaskPageHeader";

const taskSearchSchema = z.object({
  status: z
    .enum(["active", "waiting", "stopped", "uploading", "failed"])
    .default("active"),
});

export type TaskStatus = z.infer<typeof taskSearchSchema>["status"];

export const tasksLinkOptions = (status: TaskStatus) =>
  linkOptions({
    to: "/tasks",
    search: { status },
  });

export const Route = createFileRoute("/tasks")({
  validateSearch: (search) => taskSearchSchema.parse(search),
  component: TasksPage,
});

function TasksPage() {
  const { status } = Route.useSearch();

  const titles: Record<string, string> = {
    active: "Active",
    waiting: "Queued",
    stopped: "Finished",
    uploading: "Uploading",
    failed: "Failed",
  };

  const colors: Record<string, string> = {
    active: "text-success",
    waiting: "text-warning",
    stopped: "text-accent",
    uploading: "text-cyan-500",
    failed: "text-danger",
  };

  return (
    <div className="space-y-6 px-1">
      <TaskPageHeader title={titles[status]} titleColor={colors[status]} />
      <TaskList status={status} />
    </div>
  );
}
