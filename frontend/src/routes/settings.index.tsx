import {
  Card,
  Chip,
  ScrollShadow,
  Spinner,
} from "@heroui/react";
import { createFileRoute, Link } from "@tanstack/react-router";
import IconChevronRight from "~icons/gravity-ui/chevron-right";
import IconCloud from "~icons/gravity-ui/cloud";
import IconGear from "~icons/gravity-ui/gear";
import IconThunderbolt from "~icons/gravity-ui/thunderbolt";
import IconGlobe from "~icons/gravity-ui/globe";
import IconRocket from "~icons/gravity-ui/rocket";
import { useProviders } from "../hooks/useProviders";
import { useSettingsStore } from "../store/useSettingsStore";
import type { components } from "../gen/api";

type ProviderSummary = components["schemas"]["model.ProviderSummary"];

export const Route = createFileRoute("/settings/")({
  component: SettingsOverview,
});

function SettingsOverview() {
  const { serverSettings } = useSettingsStore();
  const { data: providersResponse, isLoading: isLoadingProviders } = useProviders();
  const providers = providersResponse?.data || [];

  if (!serverSettings) {
    return (
      <div className="flex items-center justify-center h-full">
        <Spinner size="lg" color="accent" />
      </div>
    );
  }

  const sections = [
    {
      id: "downloads",
      title: "Downloads",
      icon: <IconRocket className="w-5 h-5" />,
      color: "bg-orange-500/10 text-orange-500",
      description: "Speed limits, concurrent tasks & paths",
      to: "/settings/downloads" as const,
    },
    {
      id: "uploads",
      title: "Cloud Remotes",
      icon: <IconCloud className="w-5 h-5" />,
      color: "bg-cyan-500/10 text-cyan-500",
      description: "Manage Rclone destinations & auto-upload",
      to: "/settings/uploads" as const,
    },
    {
      id: "premium",
      title: "Premium Services",
      icon: <IconThunderbolt className="w-5 h-5" />,
      color: "bg-yellow-500/10 text-yellow-500",
      description: "Debrid accounts & multi-hosters",
      to: "/settings/premium" as const,
    },
    {
      id: "network",
      title: "Network",
      icon: <IconGlobe className="w-5 h-5" />,
      color: "bg-blue-500/10 text-blue-500",
      description: "Proxy, DNS over HTTPS & ports",
      to: "/settings/network" as const,
    },
    {
      id: "browser",
      title: "Browser",
      icon: <IconGear className="w-5 h-5" />,
      color: "bg-purple-500/10 text-purple-500",
      description: "VFS cache & search indexing",
      to: "/settings/browser" as const,
    },
  ];

  return (
    <div className="flex flex-col h-full space-y-6">
      <div className="px-2 shrink-0">
        <h2 className="text-2xl font-bold tracking-tight">Settings</h2>
        <p className="text-xs text-muted">
          Configure Gravity download engine and integrations
        </p>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        <ScrollShadow className="h-full">
          <div className="max-w-4xl mx-auto p-8 space-y-10">
            {/* Main Sections */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {sections.map((section) => (
                <Link
                  key={section.id}
                  to={section.to}
                  className="block group outline-none"
                >
                  <Card className="bg-background/50 border-border group-hover:bg-background transition-colors cursor-pointer group-focus-visible:ring-2 ring-accent">
                    <Card.Content className="p-6 flex items-center gap-4">
                      <div className={`p-3 rounded-2xl ${section.color}`}>
                        {section.icon}
                      </div>
                      <div className="flex-1 min-w-0">
                        <h4 className="font-bold text-base leading-tight group-hover:text-accent transition-colors">
                          {section.title}
                        </h4>
                        <p className="text-xs text-muted mt-1 leading-relaxed">
                          {section.description}
                        </p>
                      </div>
                      <IconChevronRight className="w-4 h-4 text-muted group-hover:text-accent group-hover:translate-x-0.5 transition-all" />
                    </Card.Content>
                  </Card>
                </Link>
              ))}
            </div>

            {/* Providers Status */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Service Status</h3>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                {isLoadingProviders ? (
                  <div className="col-span-full py-12 flex justify-center">
                    <Spinner size="sm" />
                  </div>
                ) : (
                  providers.map((p: ProviderSummary) => (
                    <Card
                      key={p.name}
                      className="bg-background/50 border-border p-4 flex flex-col gap-3"
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-bold text-sm">{p.name}</span>
                        <Chip
                          size="sm"
                          variant="soft"
                          color={p.enabled ? "success" : "default"}
                          className="h-5 text-[9px] font-black uppercase tracking-widest"
                        >
                          {p.enabled ? "Active" : "Disabled"}
                        </Chip>
                      </div>
                      {p.enabled && p.account && (
                        <div className="space-y-1">
                          <p className="text-[10px] text-muted font-bold uppercase tracking-widest">
                            {p.account.username || "Guest"}
                          </p>
                          <div className="flex items-center gap-2">
                            <span className="w-1.5 h-1.5 rounded-full bg-success animate-pulse" />
                            <span className="text-[10px] font-black text-success uppercase tracking-widest">
                              {p.account.isPremium ? "Premium" : "Free"}
                            </span>
                          </div>
                        </div>
                      )}
                    </Card>
                  ))
                )}
              </div>
            </section>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}