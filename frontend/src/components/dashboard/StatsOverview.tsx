import { Card } from "@heroui/react";
import type React from "react";
import IconArrowDown from "~icons/gravity-ui/arrow-down";
import IconArrowUp from "~icons/gravity-ui/arrow-up";
import IconPulse from "~icons/gravity-ui/pulse";
import IconDatabase from "~icons/gravity-ui/database";
import { useGlobalStat } from "../../hooks/useEngine";
import { useSpeedHistory } from "../../hooks/useSpeedHistory";
import { formatBytes } from "../../lib/utils";
import { SpeedGraph } from "../ui/SpeedGraph";

export const StatsOverview: React.FC = () => {
  const { data: stats } = useGlobalStat();
  const { downloadHistory, uploadHistory } = useSpeedHistory();

  const uploadSpeed = stats?.speeds?.upload ?? 0;

  const tasksFinished = stats?.tasks?.completed ?? 0;
  const tasksFailed = stats?.tasks?.failed ?? 0;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="overflow-hidden shadow-sm border-border">
          <Card.Content className="p-4 flex flex-col gap-2">
            <div className="flex items-center gap-4">
              <div className="p-3 rounded-full bg-success/10 text-success">
                <IconArrowDown className="w-6 h-6" />
              </div>
              <div>
                <p className="text-sm text-muted font-medium">Download Speed</p>
                <h4 className="text-2xl font-bold">
                  {formatBytes(stats?.speeds?.download ?? 0)}/s
                </h4>
              </div>
            </div>
            <div className="mt-2 flex items-center justify-between text-xs text-muted font-medium">
              <span>{tasksFinished} tasks finished</span>
              {tasksFailed > 0 && <span className="text-danger">{tasksFailed} failed</span>}
            </div>
            <SpeedGraph
              data={downloadHistory}
              color="var(--success)"
              height={40}
              className="mt-1"
            />
          </Card.Content>
        </Card>

        <Card className="overflow-hidden shadow-sm border-border">
          <Card.Content className="p-4 flex flex-col gap-2">
            <div className="flex items-center gap-4">
              <div className="p-3 rounded-full bg-cyan-500/10 text-cyan-500">
                <IconArrowUp className="w-6 h-6" />
              </div>
              <div>
                <p className="text-sm text-muted font-medium">Upload Speed</p>
                <h4 className="text-2xl font-bold">
                  {formatBytes(uploadSpeed)}/s
                </h4>
              </div>
            </div>
            <div className="mt-2 flex items-center justify-between text-xs text-muted font-medium">
              <span>{stats?.tasks?.uploading ?? 0} active uploads</span>
              <span>{formatBytes(stats?.usage?.totalUploaded ?? 0)} total</span>
            </div>
            <SpeedGraph
              data={uploadHistory}
              color="#06b6d4"
              height={40}
              className="mt-1"
            />
          </Card.Content>
        </Card>

        <Card className="shadow-sm border-border">
          <Card.Content className="p-4 flex items-center h-full gap-4">
            <div className="p-3 rounded-full bg-warning/10 text-warning">
              <IconPulse className="w-6 h-6" />
            </div>
            <div>
              <p className="text-sm text-muted font-medium">Active Tasks</p>
              <div className="flex gap-2 items-baseline">
                <h4 className="text-2xl font-bold">{stats?.tasks?.active ?? 0}</h4>
                <span className="text-sm text-muted">
                  ({stats?.tasks?.waiting ?? 0} queued)
                </span>
              </div>
              <div className="text-xs text-muted mt-1 font-medium">
                Lifetime: {formatBytes(stats?.usage?.totalDownloaded ?? 0)}
              </div>
            </div>
          </Card.Content>
        </Card>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card className="shadow-sm border-border">
          <Card.Content className="p-4 flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <IconDatabase className="w-5 h-5 text-muted" />
                <span className="text-sm font-bold">Disk Space</span>
              </div>
              <span className="text-xs font-mono text-muted">
                {formatBytes(stats?.system?.diskFree ?? 0)} free of {formatBytes(stats?.system?.diskTotal ?? 0)}
              </span>
            </div>
            <div className="space-y-1.5">
              <div className="h-2 w-full bg-default/10 rounded-full overflow-hidden">
                <div 
                  className="h-full bg-accent transition-all duration-500" 
                  style={{ width: `${stats?.system?.diskUsage || 0}%` }}
                />
              </div>
              <div className="flex justify-end">
                <span className="text-[10px] font-black uppercase tracking-widest text-muted">
                  {stats?.system?.diskUsage?.toFixed(1)}% Used
                </span>
              </div>
            </div>
          </Card.Content>
        </Card>

        <Card className="shadow-sm border-border">
          <Card.Content className="p-4 flex items-center justify-between h-full">
            <div className="space-y-1">
              <p className="text-[10px] font-black uppercase text-muted tracking-widest">Session Usage</p>
              <div className="flex gap-4">
                <div className="flex flex-col">
                  <span className="text-xs text-muted">Downloaded</span>
                  <span className="text-sm font-bold text-success">{formatBytes(stats?.usage?.sessionDownloaded ?? 0)}</span>
                </div>
                <div className="flex flex-col">
                  <span className="text-xs text-muted">Uploaded</span>
                  <span className="text-sm font-bold text-accent">{formatBytes(stats?.usage?.sessionUploaded ?? 0)}</span>
                </div>
              </div>
            </div>
            <div className="text-right">
              <p className="text-[10px] font-black uppercase text-muted tracking-widest">Uptime</p>
              <span className="text-sm font-mono font-bold">
                {Math.floor((stats?.system?.uptime ?? 0) / 3600)}h {Math.floor(((stats?.system?.uptime ?? 0) % 3600) / 60)}m
              </span>
            </div>
          </Card.Content>
        </Card>
      </div>
    </div>
  );
};