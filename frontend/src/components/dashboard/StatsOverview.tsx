import { Card } from "@heroui/react";
import type React from "react";
import IconArrowDown from "~icons/gravity-ui/arrow-down";
import IconArrowUp from "~icons/gravity-ui/arrow-up";
import IconPulse from "~icons/gravity-ui/pulse";
import { useGlobalStat } from "../../hooks/useEngine";
import { useSpeedHistory } from "../../hooks/useSpeedHistory";
import { formatBytes } from "../../lib/utils";
import { SpeedGraph } from "../ui/SpeedGraph";

export const StatsOverview: React.FC = () => {
  const { data: stats } = useGlobalStat();
  const { downloadHistory, uploadHistory } = useSpeedHistory();

  const uploadSpeed = stats?.active?.uploadSpeed ?? 0;

  const tasksFinished = stats?.totals?.tasksFinished ?? 0;
  const tasksFailed = stats?.totals?.tasksFailed ?? 0;

  return (
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
                {formatBytes(stats?.active?.downloadSpeed ?? 0)}/s
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
            <span>{stats?.active?.uploads ?? 0} active uploads</span>
            <span>{formatBytes(stats?.totals?.totalUploaded ?? 0)} total</span>
          </div>
          <SpeedGraph
            data={uploadHistory}
            color="var(--cyan-500)"
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
              <h4 className="text-2xl font-bold">{stats?.active?.downloads ?? 0}</h4>
              <span className="text-sm text-muted">
                ({stats?.queue?.pending ?? 0} queued)
              </span>
            </div>
            <div className="text-xs text-muted mt-1 font-medium">
              Lifetime: {formatBytes(stats?.totals?.totalDownloaded ?? 0)}
            </div>
          </div>
        </Card.Content>
      </Card>
    </div>
  );
};
