import { createFileRoute, linkOptions } from "@tanstack/react-router";
import { z } from "zod";
import { TaskList } from "../components/dashboard/TaskList";
import { TaskPageHeader } from "../components/dashboard/TaskPageHeader";

const taskSearchSchema = z.object({
  status: z
    .enum(["active", "waiting", "paused", "uploading", "complete", "error"])
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

  const titles: Record<TaskStatus, string> = {
    active: "Active",
    waiting: "Waiting",
    paused: "Paused",
    uploading: "Uploading",
    complete: "Completed",
    error: "Failed",
  };

  const colors: Record<TaskStatus, string> = {
    active: "text-success",
    waiting: "text-muted",
    paused: "text-warning",
    uploading: "text-cyan-500",
    complete: "text-accent",
    error: "text-danger",
  };

  return (
    <div className="space-y-6 px-1">
      <TaskPageHeader
        title={titles[status]}
        titleColor={colors[status]}
        status={status}
      />
      <TaskList status={status} />
    </div>
  );
}
