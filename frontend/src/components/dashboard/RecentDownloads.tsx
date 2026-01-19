import { Button, Card } from "@heroui/react";
import { Link } from "@tanstack/react-router";
import type React from "react";
import IconArrowRight from "~icons/gravity-ui/arrow-right";
import { useRecentDownloads } from "../../hooks/useEngine";
import { DownloadCard } from "./DownloadCard";

export const RecentDownloads: React.FC = () => {
  const { data: downloads, isLoading } = useRecentDownloads(5);

  if (isLoading) {
    return <div className="animate-pulse h-48 bg-default/10 rounded-3xl" />;
  }

  if (!downloads || downloads.length === 0) {
    return (
        <Card className="shadow-sm border-border bg-muted-background/30 p-8 flex flex-col items-center justify-center text-center gap-2">
            <p className="text-sm font-bold text-muted">No recent downloads</p>
            <Link to="/add">
                <Button size="sm" variant="secondary">Start Download</Button>
            </Link>
        </Card>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-bold">Recent Activity</h3>
        <Link to="/tasks" search={{ status: 'active' }}>
            <Button size="sm" variant="ghost" className="text-accent font-bold">
                View All <IconArrowRight className="ml-1 w-4 h-4" />
            </Button>
        </Link>
      </div>
      
            <div className="flex flex-col gap-3">
      
              {downloads.map((task) => (
      
                  <DownloadCard key={task.id} task={task} variant="list" />
      
              ))}
      
            </div>
      
          </div>
      
        );
      
      };
      
      