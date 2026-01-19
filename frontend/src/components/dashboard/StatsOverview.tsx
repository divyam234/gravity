import { Card } from "@heroui/react";
import type React from "react";
import IconArrowDown from "~icons/gravity-ui/arrow-down";
import IconArrowUp from "~icons/gravity-ui/arrow-up";
import IconCloud from "~icons/gravity-ui/cloud-arrow-up-in";
import IconPulse from "~icons/gravity-ui/pulse";
import { useStats } from "../../hooks/useStats";
import { useSpeedHistory } from "../../hooks/useSpeedHistory";
import { formatBytes } from "../../lib/utils";
import { SpeedGraph } from "../ui/SpeedGraph";

export const StatsOverview: React.FC = () => {
  const { data: stats } = useStats();
  const { downloadHistory, uploadHistory } = useSpeedHistory();

  const numUploading = stats?.active?.uploads ?? 0;
  const cloudUploadSpeed = stats?.active?.uploadSpeed ?? 0;

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
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
          <SpeedGraph
            data={downloadHistory}
            color="var(--success)"
            height={40}
            className="mt-2"
          />
        </Card.Content>
      </Card>

      <Card className="overflow-hidden shadow-sm border-border">
        <Card.Content className="p-4 flex flex-col gap-2">
          <div className="flex items-center gap-4">
            <div className="p-3 rounded-full bg-accent/10 text-accent">
              <IconArrowUp className="w-6 h-6" />
            </div>
            <div>
              <p className="text-sm text-muted font-medium">Local Upload</p>
              <h4 className="text-2xl font-bold">
                {formatBytes(0)}/s
              </h4>
            </div>
          </div>
          <SpeedGraph
            data={uploadHistory}
            color="var(--accent)"
            height={40}
            className="mt-2"
          />
        </Card.Content>
      </Card>

      <Card className="overflow-hidden shadow-sm border-border">
        <Card.Content className="p-4 flex flex-col gap-2">
          <div className="flex items-center gap-4">
            <div className="p-3 rounded-full bg-cyan-500/10 text-cyan-500">
              <IconCloud className="w-6 h-6" />
            </div>
            <div>
              <p className="text-sm text-muted font-medium">Cloud Upload</p>
              <h4 className="text-2xl font-bold">
                {formatBytes(cloudUploadSpeed)}/s
              </h4>
            </div>
          </div>
          <div className="mt-2 flex items-center justify-between text-xs text-muted">
            <span>{numUploading} uploading</span>
            <span>
              {formatBytes(stats?.totals?.total_uploaded ?? 0)} total
            </span>
          </div>
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
            <div className="text-xs text-muted mt-1">
              {formatBytes(stats?.totals?.total_downloaded ?? 0)} downloaded
            </div>
          </div>
        </Card.Content>
      </Card>
    </div>
  );
};
