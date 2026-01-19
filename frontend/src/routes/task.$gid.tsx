import {
  Button,
  Card,
  Chip,
  ScrollShadow,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import {
  useTaskStatus,
} from "../hooks/useEngine";
import { cn, formatBytes } from "../lib/utils";
import { tasksLinkOptions } from "./tasks";

export const Route = createFileRoute("/task/$gid")({
  component: TaskDetailsPage,
});

function TaskDetailsPage() {
  const { gid } = Route.useParams();
  const navigate = useNavigate();
  const { data: task } = useTaskStatus(gid);

  if (!task) return <div>Loading...</div>;

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      <div className="flex items-center gap-4 px-2">
        <Button
          variant="ghost"
          isIconOnly
          onPress={() => navigate(tasksLinkOptions("active"))}
        >
          <IconChevronLeft className="w-5 h-5" />
        </Button>
        <h2 className="text-2xl font-bold tracking-tight">Task Details</h2>
        <code className="text-xs bg-default/10 border border-border px-3 py-1 rounded-full text-muted font-bold">
          {gid}
        </code>
      </div>

      <div className="flex flex-col md:flex-row gap-8">
        {/* Content Area */}
        <div className="flex-1 min-h-[600px]">
          <Card className="h-full overflow-hidden flex flex-col bg-muted-background/20 shadow-sm border border-border rounded-[40px]">
              <ScrollShadow className="flex-1 p-8">
                <div className="space-y-10 text-foreground">
                  <section>
                    <h3 className="text-base font-black uppercase tracking-widest text-muted mb-6 flex items-center gap-3">
                      <div className="w-2 h-2 rounded-full bg-accent" />
                      Identity
                    </h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-8 bg-background/50 p-8 rounded-[32px] border border-border shadow-sm">
                      <div className="space-y-1.5">
                        <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                          Filename
                        </p>
                        <p className="text-lg font-bold break-all tracking-tight leading-tight">
                          {task.filename || gid}
                        </p>
                      </div>
                      <div className="space-y-1.5">
                        <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                          ID
                        </p>
                        <p className="font-mono text-sm bg-default/10 px-3 py-1 rounded-full border border-border inline-block font-bold">
                          {task.id}
                        </p>
                      </div>
                    </div>
                  </section>

                  <section>
                    <h3 className="text-base font-black uppercase tracking-widest text-muted mb-6 flex items-center gap-3">
                      <div className="w-2 h-2 rounded-full bg-accent" />
                      Location
                    </h3>
                    <div className="bg-background/50 p-8 rounded-[32px] border border-border shadow-sm">
                      <div className="space-y-1.5">
                        <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                          Destination
                        </p>
                        <p className="text-sm font-bold break-all leading-relaxed">
                          {task.destination || "Default"}
                        </p>
                      </div>
                    </div>
                  </section>

                  <section>
                    <h3 className="text-base font-black uppercase tracking-widest text-muted mb-6 flex items-center gap-3">
                      <div className="w-2 h-2 rounded-full bg-accent" />
                      Status
                    </h3>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
                      {[
                        {
                          label: "State",
                          value: task.status,
                          color: "success" as const,
                          isChip: true,
                        },
                        {
                          label: "Size",
                          value: formatBytes(task.size || 0),
                        },
                        {
                          label: "Downloaded",
                          value: formatBytes(task.downloaded || 0),
                        },
                        {
                          label: "Speed",
                          value: formatBytes(task.speed || 0) + "/s",
                        },
                      ].map((stat) => (
                        <div
                          key={stat.label}
                          className="p-6 bg-background/50 rounded-[32px] border border-border shadow-sm flex flex-col items-center justify-center text-center gap-2"
                        >
                          <p className="text-[10px] text-muted uppercase font-black tracking-widest">
                            {stat.label}
                          </p>
                          {stat.isChip ? (
                            <Chip
                              size="sm"
                              variant="soft"
                              color={stat.color}
                              className="uppercase font-black text-[10px] tracking-widest h-6 px-3"
                            >
                              {stat.value}
                            </Chip>
                          ) : (
                            <p
                              className={cn(
                                "text-lg font-black tracking-tight",
                              )}
                            >
                              {stat.value}
                            </p>
                          )}
                        </div>
                      ))}
                    </div>
                  </section>
                </div>
              </ScrollShadow>
          </Card>
        </div>
      </div>
    </div>
  );
}