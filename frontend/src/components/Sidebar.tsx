import { Button, Header, ListBox, ScrollShadow } from "@heroui/react";
import { useLocation, useNavigate } from "@tanstack/react-router";
import React from "react";
import IconArrowDown from "~icons/gravity-ui/arrow-down";
import IconCheck from "~icons/gravity-ui/check";
import IconCircleXmark from "~icons/gravity-ui/circle-xmark";
import IconClock from "~icons/gravity-ui/clock";
import IconCloudArrowUpIn from "~icons/gravity-ui/cloud-arrow-up-in";
import IconDisplay from "~icons/gravity-ui/display";
import IconGear from "~icons/gravity-ui/gear";
import IconLayoutHeaderCellsLarge from "~icons/gravity-ui/layout-header-cells-large";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import IconServer from "~icons/gravity-ui/server";
import IconXmark from "~icons/gravity-ui/xmark";
import { useStats } from "../hooks/useStats";
import { cn, formatBytes } from "../lib/utils";
import { tasksLinkOptions } from "../routes/tasks";

interface SidebarContentProps {
  onClose?: () => void;
}

export const SidebarContent: React.FC<SidebarContentProps> = ({ onClose }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: stats } = useStats();

  const activeCount = stats?.active?.downloads ?? 0;
  const waitingCount = stats?.queue?.pending ?? 0;
  const stoppedCount = (stats?.totals?.downloads_completed ?? 0) + (stats?.totals?.uploads_completed ?? 0);
  const uploadingCount = stats?.active?.uploads ?? 0;
  const failedCount = stats?.totals?.downloads_failed ?? 0;

  const mainNavItems = React.useMemo(
    () => [
      {
        key: "dashboard",
        label: "Overview",
        icon: <IconLayoutHeaderCellsLarge className="w-5 h-5" />,
        to: "/",
        count: null,
      },
      {
        key: "active",
        label: "Active",
        icon: <IconArrowDown className="w-5 h-5" />,
        linkOptions: tasksLinkOptions("active"),
        count: activeCount,
        color: "text-success",
      },
      {
        key: "uploading",
        label: "Uploading",
        icon: <IconCloudArrowUpIn className="w-5 h-5" />,
        linkOptions: tasksLinkOptions("uploading"),
        count: uploadingCount,
        color: "text-cyan-500",
      },
      {
        key: "waiting",
        label: "Queued",
        icon: <IconClock className="w-5 h-5" />,
        linkOptions: tasksLinkOptions("waiting"),
        count: waitingCount,
        color: "text-warning",
      },
      {
        key: "stopped",
        label: "Finished",
        icon: <IconCheck className="w-5 h-5" />,
        linkOptions: tasksLinkOptions("stopped"),
        count: stoppedCount,
        color: "text-accent",
      },
      {
        key: "failed",
        label: "Failed",
        icon: <IconCircleXmark className="w-5 h-5" />,
        linkOptions: tasksLinkOptions("failed"),
        count: failedCount,
        color: "text-danger",
      },
    ],
    [activeCount, waitingCount, stoppedCount, uploadingCount, failedCount]
  );

  const settingsNavItems = [
    {
      key: "engine",
      label: "Engine Options",
      icon: <IconGear className="w-4 h-4" />,
      to: "/settings/engine",
    },
    {
      key: "providers",
      label: "Providers",
      icon: <IconNodesDown className="w-4 h-4" />,
      to: "/settings/providers",
    },
    {
      key: "remotes",
      label: "Cloud Remotes",
      icon: <IconCloudArrowUpIn className="w-4 h-4" />,
      to: "/settings/remotes",
    },
    {
      key: "connection",
      label: "Server",
      icon: <IconServer className="w-4 h-4" />,
      to: "/settings/connection",
    },
    {
      key: "app",
      label: "Preferences",
      icon: <IconDisplay className="w-4 h-4" />,
      to: "/settings/app",
    },
  ];

  const selectedKey = React.useMemo(() => {
    const path = location.pathname;
    const search = location.search as any;

    if (path === "/tasks") {
      return search.status || "active";
    }

    const foundSetting = settingsNavItems.find((item) => item.to === path);
    if (foundSetting) return foundSetting.key;

    if (path === "/") return "dashboard";
    return null;
  }, [location.pathname, location.search]);

  return (
    <div className="flex flex-col h-full w-full">
      <div className="p-6 flex items-center justify-between gap-3 shrink-0">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-accent rounded-xl flex items-center justify-center text-accent-foreground font-bold text-xl shadow-lg shadow-accent/20">
            G
          </div>
          <div>
            <h1 className="font-bold tracking-tight">Gravity</h1>
            <p className="text-[10px] text-muted uppercase font-black tracking-widest leading-none">
              Download Engine
            </p>
          </div>
        </div>
        {onClose && (
          <Button
            isIconOnly
            variant="ghost"
            size="sm"
            onPress={onClose}
            className="md:hidden"
          >
            <IconXmark className="w-5 h-5" />
          </Button>
        )}
      </div>

      <ScrollShadow className="flex-1 px-3 mt-4 overflow-y-auto custom-scrollbar">
        <ListBox
          aria-label="Navigation"
          selectionMode="single"
          selectedKeys={selectedKey ? [selectedKey] : []}
          className="p-0 gap-1 mb-2"
        >
          {mainNavItems.map((item) => (
            <ListBox.Item
              key={item.key}
              id={item.key}
              textValue={item.label}
              onPress={() => {
                if (item.linkOptions) navigate(item.linkOptions);
                else if (item.to) navigate({ to: item.to });
                if (onClose) onClose();
              }}
              className={cn(
                "px-4 py-3 rounded-2xl data-[hover=true]:bg-default/10 transition-colors cursor-pointer outline-none",
                selectedKey === item.key &&
                  "bg-accent text-accent-foreground shadow-lg shadow-accent/20"
              )}
            >
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-3">
                  <span
                    className={cn(
                      "text-muted",
                      item.color,
                      selectedKey === item.key && "text-inherit"
                    )}
                  >
                    {item.icon}
                  </span>
                  <span className="text-sm font-bold tracking-tight">
                    {item.label}
                  </span>
                </div>
                {item.count !== null && (
                  <span
                    className={cn(
                      "text-[10px] font-black px-2 py-0.5 rounded-full bg-default/30 group-hover:bg-default/50 transition-colors",
                      selectedKey === item.key
                        ? "bg-accent-foreground/20 text-accent-foreground"
                        : ""
                    )}
                  >
                    {item.count}
                  </span>
                )}
              </div>
            </ListBox.Item>
          ))}
          
          <ListBox.Section className="mt-4">
            <Header className="px-4 py-2 text-xs font-black uppercase tracking-widest text-muted">Settings</Header>
            {settingsNavItems.map((item) => (
                <ListBox.Item
                    key={item.key}
                    id={item.key}
                    textValue={item.label}
                    onPress={() => {
                        navigate({ to: item.to });
                        if (onClose) onClose();
                    }}
                    className={cn(
                        "px-4 py-2.5 rounded-2xl data-[hover=true]:bg-default/10 transition-colors cursor-pointer outline-none",
                        selectedKey === item.key &&
                        "bg-default/20 font-bold"
                    )}
                >
                    <div className="flex items-center gap-3">
                        <span className="text-muted">{item.icon}</span>
                        <span className="text-sm font-medium">{item.label}</span>
                    </div>
                </ListBox.Item>
            ))}
          </ListBox.Section>
        </ListBox>
      </ScrollShadow>

      <div className="p-6 mt-auto shrink-0">
        <div className="p-4 rounded-3xl bg-default/10 border border-border flex flex-col gap-2">
          <p className="text-[10px] font-black uppercase text-muted tracking-widest">
            Session Speed
          </p>
          <div className="flex flex-col">
            <span className="text-xs font-bold text-success">
              DL: {formatBytes(stats?.active?.downloadSpeed || 0)}/s
            </span>
            <span className="text-xs font-bold text-accent">
              UL: {formatBytes(stats?.active?.uploadSpeed || 0)}/s
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};

export const Sidebar: React.FC = () => {
  return (
    <aside className="w-64 border-r border-border bg-muted-background/30 hidden md:flex flex-col h-full shrink-0">
      <SidebarContent />
    </aside>
  );
};

export const MobileSidebar: React.FC<{
  isOpen: boolean;
  onClose: () => void;
}> = ({ isOpen, onClose }) => {
  return (
    <>
      {isOpen && (
        <button
          type="button"
          className="fixed inset-0 bg-black/50 z-40 backdrop-blur-sm md:hidden animate-in fade-in duration-200 border-none cursor-default w-full h-full block"
          onClick={onClose}
          onKeyDown={(e) => {
            if (e.key === "Escape") onClose();
          }}
          aria-label="Close menu"
        />
      )}
      <div
        className={cn(
          "fixed inset-y-0 left-0 z-50 w-72 bg-background shadow-2xl transform transition-transform duration-300 md:hidden",
          isOpen ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <SidebarContent onClose={onClose} />
      </div>
    </>
  );
};
