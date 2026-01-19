import {
  Button,
  Card,
  Chip,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconCheck from "~icons/gravity-ui/check";
import IconCircleXmark from "~icons/gravity-ui/circle-xmark";
import IconClock from "~icons/gravity-ui/clock";
import IconMagnet from "~icons/gravity-ui/magnet";
import {
  useTaskStatus,
} from "../hooks/useEngine";
import { formatBytes } from "../lib/utils";
import { tasksLinkOptions } from "./tasks";
import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { ProgressBar } from "../components/ui/ProgressBar";
import type { DownloadFile } from "../lib/types";

export const Route = createFileRoute("/task/$gid")({
  component: TaskDetailsPage,
});

function TaskDetailsPage() {
  const { gid } = Route.useParams();
  const navigate = useNavigate();
  const { data: task } = useTaskStatus(gid);
  const [files, setFiles] = useState<DownloadFile[]>([]);

  useEffect(() => {
    if (task?.isMagnet) {
      api.getDownloadFiles(gid).then(res => setFiles(res.files));
      
      const interval = setInterval(() => {
        api.getDownloadFiles(gid).then(res => setFiles(res.files));
      }, 5000);
      return () => clearInterval(interval);
    }
  }, [task?.isMagnet, gid]);

  if (!task) return <div>Loading...</div>;

  return (
    <div className="max-w-6xl mx-auto space-y-6 pb-20 mt-6 px-4 md:px-0">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          isIconOnly
          onPress={() => navigate(tasksLinkOptions("active"))}
          className="h-10 w-10 rounded-xl"
        >
          <IconChevronLeft className="w-5 h-5" />
        </Button>
        <div>
          <h2 className="text-2xl font-black uppercase tracking-tight leading-none">Task Details</h2>
          <p className="text-[10px] text-muted font-black uppercase tracking-widest mt-1">
            Download GID: <span className="text-foreground/60">{gid}</span>
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
        {/* Content Area */}
        <div className="lg:col-span-8 space-y-8">
          <Card className="overflow-hidden flex flex-col bg-background shadow-sm border border-border rounded-[40px]">
              <div className="p-8 space-y-10 text-foreground">
                  <section>
                    <h3 className="text-[10px] font-black uppercase tracking-widest text-muted mb-6 flex items-center gap-3">
                      <div className="w-2 h-2 rounded-full bg-accent" />
                      Metadata
                    </h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-8 bg-default/5 p-8 rounded-[32px] border border-border/50 shadow-sm">
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
                          Source URL
                        </p>
                        <p className="text-xs font-mono break-all text-muted-foreground line-clamp-2">
                          {task.url}
                        </p>
                      </div>
                    </div>
                  </section>

                  {task.isMagnet && files.length > 0 && (
                    <section>
                      <h3 className="text-[10px] font-black uppercase tracking-widest text-muted mb-6 flex items-center gap-3">
                        <div className="w-2 h-2 rounded-full bg-accent" />
                        Files ({files.length})
                      </h3>
                      <div className="space-y-3 bg-default/5 p-6 rounded-[32px] border border-border/50 shadow-sm max-h-[600px] overflow-y-auto custom-scrollbar">
                        {files.map((file) => (
                          <div key={file.id} className="bg-background/80 p-4 rounded-2xl border border-border/50 flex flex-col gap-3">
                            <div className="flex items-center justify-between gap-4">
                              <div className="flex items-center gap-3 min-w-0">
                                <div className="w-8 h-8 bg-default/10 rounded-xl flex items-center justify-center shrink-0">
                                  {file.status === 'complete' ? <IconCheck className="w-4 h-4 text-success" /> : 
                                   file.status === 'error' ? <IconCircleXmark className="w-4 h-4 text-danger" /> :
                                   file.status === 'active' ? <div className="w-2 h-2 rounded-full bg-accent animate-pulse" /> :
                                   <IconClock className="w-4 h-4 text-muted" />}
                                </div>
                                <div className="min-w-0">
                                  <p className="text-sm font-bold truncate leading-tight">{file.name}</p>
                                  <p className="text-[10px] text-muted font-black uppercase tracking-widest mt-0.5">
                                    {formatBytes(file.downloaded)} / {formatBytes(file.size)}
                                  </p>
                                </div>
                              </div>
                              <Chip 
                                size="sm" 
                                variant="soft" 
                                color={file.status === 'complete' ? 'success' : file.status === 'error' ? 'danger' : 'default'}
                                className="font-black uppercase tracking-widest text-[9px]"
                              >
                                {file.status}
                              </Chip>
                            </div>
                            <ProgressBar 
                              value={file.progress} 
                              size="sm" 
                              color={file.status === 'complete' ? 'success' : 'accent'} 
                              className="h-1"
                            />
                          </div>
                        ))}
                      </div>
                    </section>
                  )}
              </div>
          </Card>
        </div>

        {/* Sidebar / Stats */}
        <div className="lg:col-span-4 space-y-6">
            <Card className="p-8 bg-background border border-border rounded-[40px] shadow-sm">
                <h3 className="text-[10px] font-black uppercase tracking-widest text-muted mb-8 flex items-center gap-3">
                  <div className="w-2 h-2 rounded-full bg-accent" />
                  Status Overview
                </h3>
                
                <div className="space-y-6">
                    <div className="flex flex-col gap-2">
                        <p className="text-[10px] text-muted uppercase font-black tracking-widest px-1">Progress</p>
                        <div className="bg-default/5 p-6 rounded-3xl border border-border/50">
                            <div className="flex justify-between items-end mb-4">
                                <p className="text-3xl font-black tracking-tighter leading-none">
                                    {task.size > 0 ? Math.floor((task.downloaded / task.size) * 100) : 0}%
                                </p>
                                <p className="text-xs font-bold text-muted uppercase tracking-widest">
                                    {task.status}
                                </p>
                            </div>
                            <ProgressBar 
                                value={task.size > 0 ? (task.downloaded / task.size) * 100 : 0} 
                                color={task.status === 'complete' ? 'success' : 'accent'}
                                className="h-2"
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div className="bg-default/5 p-4 rounded-3xl border border-border/50">
                            <p className="text-[10px] text-muted uppercase font-black tracking-widest mb-1">Speed</p>
                            <p className="text-sm font-bold text-success">{formatBytes(task.speed)}/s</p>
                        </div>
                        <div className="bg-default/5 p-4 rounded-3xl border border-border/50">
                            <p className="text-[10px] text-muted uppercase font-black tracking-widest mb-1">Total Size</p>
                            <p className="text-sm font-bold">{formatBytes(task.size)}</p>
                        </div>
                    </div>

                    {task.isMagnet && (
                        <div className="bg-accent/5 p-6 rounded-3xl border border-accent/20">
                            <p className="text-[10px] text-accent uppercase font-black tracking-widest mb-3 flex items-center gap-2">
                                <IconMagnet className="w-3 h-3" />
                                Magnet Stats
                            </p>
                            <div className="space-y-3">
                                <div className="flex justify-between items-center">
                                    <span className="text-xs font-medium text-muted-foreground">Files</span>
                                    <span className="text-xs font-black">{task.filesComplete || 0} / {task.totalFiles || 0}</span>
                                </div>
                                <div className="flex justify-between items-center">
                                    <span className="text-xs font-medium text-muted-foreground">Source</span>
                                    <span className="text-xs font-black uppercase tracking-widest">{task.magnetSource}</span>
                                </div>
                            </div>
                        </div>
                    )}
                </div>
            </Card>
        </div>
      </div>
    </div>
  );
}