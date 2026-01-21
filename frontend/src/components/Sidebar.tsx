import { Button, ListBox, ScrollShadow, Label, Accordion } from "@heroui/react";
import { useLocation, useSearch } from "@tanstack/react-router";
import React from "react";
import IconChevronRight from "~icons/gravity-ui/chevron-right";
import IconArrowDown from "~icons/gravity-ui/arrow-down";
import IconThunderbolt from "~icons/gravity-ui/thunderbolt";
import IconCheck from "~icons/gravity-ui/check";
import IconCircleXmark from "~icons/gravity-ui/circle-xmark";
import IconClock from "~icons/gravity-ui/clock";
import IconCloud from "~icons/gravity-ui/cloud";
import IconCloudArrowUpIn from "~icons/gravity-ui/cloud-arrow-up-in";
import IconDisplay from "~icons/gravity-ui/display";
import IconFolder from "~icons/gravity-ui/folder";
import IconGlobe from "~icons/gravity-ui/globe";
import IconLayoutHeaderCellsLarge from "~icons/gravity-ui/layout-header-cells-large";
import IconMagnet from "~icons/gravity-ui/magnet";
import IconRocket from "~icons/gravity-ui/rocket";
import IconGear from "~icons/gravity-ui/gear";
import IconServer from "~icons/gravity-ui/server";
import IconXmark from "~icons/gravity-ui/xmark";
import { useQueryClient } from "@tanstack/react-query";
import { useGlobalStat } from "../hooks/useEngine";
import { cn, formatBytes } from "../lib/utils";
import { tasksLinkOptions } from "../routes/tasks";

interface NavItem {
  key: string;
  label: string;
  icon: React.ReactNode;
  to: string;
  count: number | null;
  color?: string;
  linkOptions?: any;
}

interface SidebarContentProps {
  onClose?: () => void;
}

export const SidebarContent: React.FC<SidebarContentProps> = ({ onClose }) => {
  const location = useLocation();
  const search: any = useSearch({ from: "__root__" });
  const queryClient = useQueryClient();
  const { data: stats } = useGlobalStat();

  const activeCount = stats?.tasks?.active ?? 0;
  const pendingCount = stats?.tasks?.waiting ?? 0;
  const pausedCount = stats?.tasks?.paused ?? 0;
  const completeCount = stats?.tasks?.completed ?? 0;
  const uploadingCount = stats?.tasks?.uploading ?? 0;
  const errorCount = stats?.tasks?.failed ?? 0;

  const topNavItems = React.useMemo<NavItem[]>(
    () => [
      {
        key: "dashboard",
        label: "Overview",
        icon: <IconLayoutHeaderCellsLarge className="w-5 h-5" />,
        to: "/",
        count: null,
      },
      {
        key: "files",
        label: "Browser",
        icon: <IconFolder className="w-5 h-5" />,
        to: "/files",
        linkOptions: { search: { path: "/" } },
        count: null,
      },
    ],
    [],
  );

  const downloadNavItems = React.useMemo<NavItem[]>(
    () => [
      {
        key: "active",
        label: "Active",
        icon: <IconArrowDown className="w-5 h-5" />,
        to: "/tasks",
        linkOptions: tasksLinkOptions("active"),
        count: activeCount,
        color: "text-success",
      },
      {
        key: "uploading",
        label: "Uploading",
        icon: <IconCloudArrowUpIn className="w-5 h-5" />,
        to: "/tasks",
        linkOptions: tasksLinkOptions("uploading"),
        count: uploadingCount,
        color: "text-cyan-500",
      },
      {
        key: "paused",
        label: "Paused",
        icon: <IconClock className="w-5 h-5" />,
        to: "/tasks",
        linkOptions: tasksLinkOptions("paused"),
        count: pausedCount,
        color: "text-warning",
      },
      {
        key: "waiting",
        label: "Waiting",
        icon: <IconClock className="w-5 h-5" />,
        to: "/tasks",
        linkOptions: tasksLinkOptions("waiting"),
        count: pendingCount,
        color: "text-muted",
      },
      {
        key: "complete",
        label: "Completed",
        icon: <IconCheck className="w-5 h-5" />,
        to: "/tasks",
        linkOptions: tasksLinkOptions("complete"),
        count: completeCount,
        color: "text-accent",
      },
      {
        key: "error",
        label: "Failed",
        icon: <IconCircleXmark className="w-5 h-5" />,
        to: "/tasks",
        linkOptions: tasksLinkOptions("error"),
        count: errorCount,
        color: "text-danger",
      },
    ],
    [
      activeCount,
      pendingCount,
      pausedCount,
      completeCount,
      uploadingCount,
      errorCount,
    ],
  );

  const settingsNavItems = React.useMemo<NavItem[]>(
    () => [
      {
        key: "downloads",
        label: "Downloads",
        icon: <IconRocket className="w-4 h-4" />,
        to: "/settings/downloads",
        count: null,
      },
      {
        key: "cloud",
        label: "Cloud Storage",
        icon: <IconCloud className="w-4 h-4" />,
        to: "/settings/cloud",
        count: null,
      },
      {
        key: "premium",
        label: "Premium Services",
        icon: <IconThunderbolt className="w-4 h-4" />,
        to: "/settings/premium",
        count: null,
      },
      {
        key: "network",
        label: "Network",
        icon: <IconGlobe className="w-4 h-4" />,
        to: "/settings/network",
        count: null,
      },
      {
        key: "torrents",
        label: "Torrents",
        icon: <IconMagnet className="w-4 h-4" />,
        to: "/settings/torrents",
        count: null,
      },
      {
        key: "browser",
        label: "Browser",
        icon: <IconFolder className="w-4 h-4" />,
        to: "/settings/browser",
        count: null,
      },
      {
        key: "preferences",
        label: "Preferences",
        icon: <IconDisplay className="w-4 h-4" />,
        to: "/settings/preferences",
        count: null,
      },
      {
        key: "server",
        label: "Server",
        icon: <IconServer className="w-4 h-4" />,
        to: "/settings/server",
        count: null,
      },
    ],
    [],
  );

  const selectedKey = React.useMemo(() => {
    const path = location.pathname;

    if (path === "/tasks") {
      return search.status || "active";
    }

    if (path === "/files") {
      return "files";
    }

    const foundSetting = settingsNavItems.find((item) => item.to === path);
    if (foundSetting) return foundSetting.key;

    if (path === "/") return "dashboard";
    return null;
  }, [location.pathname, search.status, settingsNavItems]);

  const isDownloadsHeaderActive =
    location.pathname === "/tasks" && !search.status;
  const isSettingsHeaderActive =
    location.pathname === "/settings" || location.pathname === "/settings/";

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

      <ScrollShadow className="flex-1 px-3 mt-4 overflow-y-auto">
        {/* Top Items (Fixed) */}
        <ListBox
          aria-label="Top Navigation"
          selectionMode="single"
          selectedKeys={selectedKey ? [selectedKey] : []}
          className="p-0 gap-1 mb-2"
          items={topNavItems}
        >
          {(item) => (
            <ListBox.Item
              id={item.key}
              href={item.to as any}
              routerOptions={item.linkOptions}
              textValue={item.label}
              onPress={() => {
                if (onClose) onClose();
              }}
              className={cn(
                "px-4 py-2.5 rounded-2xl transition-all cursor-pointer outline-none group relative overflow-hidden",
                "hover:bg-default/10 focus-visible:bg-default/10 focus-visible:ring-2 focus-visible:ring-accent/50",
                "data-[selected=true]:bg-accent/10 data-[selected=true]:text-accent data-[selected=true]:font-bold",
              )}
            >
              <div className="flex items-center gap-3 relative z-10">
                <span
                  className={cn(
                    "text-muted transition-colors group-data-[selected=true]:text-inherit",
                  )}
                >
                  {item.icon}
                </span>
                <Label className="text-sm tracking-tight text-inherit cursor-pointer">
                  {item.label}
                </Label>
              </div>
              <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-accent rounded-r-full transform -translate-x-full transition-transform duration-200 group-data-[selected=true]:translate-x-0" />
            </ListBox.Item>
          )}
        </ListBox>

        <Accordion
          allowsMultipleExpanded
          defaultExpandedKeys={["downloads", "settings"]}
          hideSeparator
          className="p-0 flex flex-col gap-1"
        >
          <Accordion.Item id="downloads" className="border-none p-0">
            <Accordion.Heading>
              <Accordion.Trigger
                className={cn(
                  "px-4 py-2.5 rounded-2xl transition-all cursor-pointer outline-none group flex items-center justify-between w-full text-left relative overflow-hidden",
                  "hover:bg-default/10 focus-visible:bg-default/10 focus-visible:ring-2 focus-visible:ring-accent/50",
                  isDownloadsHeaderActive
                    ? "bg-accent/10 text-accent font-bold"
                    : "text-foreground",
                )}
              >
                <div className="flex items-center gap-3 relative z-10">
                  <span
                    className={cn(
                      isDownloadsHeaderActive ? "text-inherit" : "text-muted",
                    )}
                  >
                    <IconRocket className="w-5 h-5" />
                  </span>
                  <Label className="text-sm font-bold tracking-tight text-inherit cursor-pointer">
                    Downloads
                  </Label>
                </div>
                <Accordion.Indicator>
                  <IconChevronRight
                    className={cn(
                      "w-3.5 h-3.5 transition-transform group-aria-expanded:rotate-90 opacity-50 group-hover:opacity-100",
                    )}
                  />
                </Accordion.Indicator>
                <div
                  className={cn(
                    "absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-accent rounded-r-full transform transition-transform duration-200",
                    isDownloadsHeaderActive ? "translate-x-0" : "-translate-x-full",
                  )}
                />
              </Accordion.Trigger>
            </Accordion.Heading>
            <Accordion.Panel>
              <Accordion.Body className="p-0 pt-1">
                <ListBox
                  aria-label="Downloads Navigation"
                  selectionMode="single"
                  selectedKeys={selectedKey ? [selectedKey] : []}
                  className="p-0 gap-1 mb-2"
                  items={downloadNavItems}
                >
                  {(item) => (
                    <ListBox.Item
                      id={item.key}
                      href={item.to as any}
                      routerOptions={item.linkOptions}
                      textValue={item.label}
                      onPress={() => {
                        queryClient.refetchQueries({
                          queryKey: ["gravity", "downloads", item.key],
                        });
                        if (onClose) onClose();
                      }}
                      className={cn(
                        "pl-11 pr-4 py-2.5 rounded-2xl transition-all cursor-pointer outline-none group relative overflow-hidden",
                        "hover:bg-default/10 focus-visible:bg-default/10 focus-visible:ring-2 focus-visible:ring-accent/50",
                        "data-[selected=true]:bg-accent/10 data-[selected=true]:text-accent data-[selected=true]:font-bold",
                      )}
                    >
                      <div className="flex items-center justify-between w-full relative z-10">
                        <div className="flex items-center gap-3">
                          <span
                            className={cn(
                              "text-muted transition-colors group-data-[selected=true]:text-inherit",
                              item.color,
                            )}
                          >
                            {item.icon}
                          </span>
                          <Label className="text-sm tracking-tight text-inherit cursor-pointer">
                            {item.label}
                          </Label>
                        </div>
                        {item.count !== null && (
                          <span
                            className={cn(
                              "text-[10px] font-black px-2 py-0.5 rounded-full bg-default/30 group-hover:bg-default/50 transition-colors",
                              "group-data-[selected=true]:bg-accent/20 group-data-[selected=true]:text-accent",
                            )}
                          >
                            {item.count}
                          </span>
                        )}
                      </div>
                      <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-accent rounded-r-full transform -translate-x-full transition-transform duration-200 group-data-[selected=true]:translate-x-0" />
                    </ListBox.Item>
                  )}
                </ListBox>
              </Accordion.Body>
            </Accordion.Panel>
          </Accordion.Item>

          <Accordion.Item id="settings" className="border-none p-0">
            <Accordion.Heading>
              <Accordion.Trigger
                className={cn(
                  "px-4 py-2.5 rounded-2xl transition-all cursor-pointer outline-none group flex items-center justify-between w-full text-left relative overflow-hidden",
                  "hover:bg-default/10 focus-visible:bg-default/10 focus-visible:ring-2 focus-visible:ring-accent/50",
                  isSettingsHeaderActive
                    ? "bg-accent/10 text-accent font-bold"
                    : "text-foreground",
                )}
              >
                <div className="flex items-center gap-3 relative z-10">
                  <span
                    className={cn(
                      isSettingsHeaderActive ? "text-inherit" : "text-muted",
                    )}
                  >
                    <IconGear className="w-5 h-5" />
                  </span>
                  <Label className="text-sm font-bold tracking-tight text-inherit cursor-pointer">
                    Settings
                  </Label>
                </div>
                <Accordion.Indicator>
                  <IconChevronRight
                    className={cn(
                      "w-3.5 h-3.5 transition-transform group-aria-expanded:rotate-90 opacity-50 group-hover:opacity-100",
                    )}
                  />
                </Accordion.Indicator>
                <div
                  className={cn(
                    "absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-accent rounded-r-full transform transition-transform duration-200",
                    isSettingsHeaderActive ? "translate-x-0" : "-translate-x-full",
                  )}
                />
              </Accordion.Trigger>
            </Accordion.Heading>
            <Accordion.Panel>
              <Accordion.Body className="p-0 pt-1">
                <ListBox
                  aria-label="Settings Navigation"
                  selectionMode="single"
                  selectedKeys={selectedKey ? [selectedKey] : []}
                  className="p-0 gap-1"
                  items={settingsNavItems}
                >
                  {(item) => (
                    <ListBox.Item
                      id={item.key}
                      href={item.to as any}
                      routerOptions={item.linkOptions}
                      textValue={item.label}
                      onPress={() => {
                        if (onClose) onClose();
                      }}
                      className={cn(
                        "pl-11 pr-4 py-2.5 rounded-2xl transition-all cursor-pointer outline-none group relative overflow-hidden",
                        "hover:bg-default/10 focus-visible:bg-default/10 focus-visible:ring-2 focus-visible:ring-accent/50",
                        "data-[selected=true]:bg-accent/10 data-[selected=true]:text-accent data-[selected=true]:font-bold",
                      )}
                    >
                      <div className="flex items-center gap-3 relative z-10">
                        <span
                          className={cn(
                            "text-muted transition-colors group-data-[selected=true]:text-inherit",
                          )}
                        >
                          {item.icon}
                        </span>
                        <Label className="text-sm tracking-tight text-inherit cursor-pointer">
                          {item.label}
                        </Label>
                      </div>
                      <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-accent rounded-r-full transform -translate-x-full transition-transform duration-200 group-data-[selected=true]:translate-x-0" />
                    </ListBox.Item>
                  )}
                </ListBox>
              </Accordion.Body>
            </Accordion.Panel>
          </Accordion.Item>
        </Accordion>
      </ScrollShadow>

      <div className="p-6 mt-auto shrink-0 space-y-4">
        <div className="p-4 rounded-3xl bg-default/10 border border-border flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <p className="text-[10px] font-black uppercase text-muted tracking-widest">
              Storage
            </p>
            <span className="text-[10px] font-bold text-muted">
              {stats?.system?.diskUsage?.toFixed(0)}%
            </span>
          </div>
          <div className="h-1.5 w-full bg-default/20 rounded-full overflow-hidden">
            <div 
              className="h-full bg-accent transition-all duration-500" 
              style={{ width: `${stats?.system?.diskUsage || 0}%` }}
            />
          </div>
          <p className="text-[10px] text-muted font-medium">
            {formatBytes(stats?.system?.diskFree || 0)} free
          </p>
        </div>

        <div className="p-4 rounded-3xl bg-default/10 border border-border flex flex-col gap-2">
          <p className="text-[10px] font-black uppercase text-muted tracking-widest">
            Session Speed
          </p>
          <div className="flex flex-col">
            <span className="text-xs font-bold text-success">
              DL: {formatBytes(stats?.speeds?.download || 0)}/s
            </span>
            <span className="text-xs font-bold text-accent">
              UL: {formatBytes(stats?.speeds?.upload || 0)}/s
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
          isOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <SidebarContent onClose={onClose} />
      </div>
    </>
  );
};