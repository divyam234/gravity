import {
  Button,
  Card,
  ScrollShadow,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { useEngineActions } from "../hooks/useEngine";
import { DynamicSettings, type SettingGroupConfig } from "../components/ui/FormFields";
import type { components } from "../gen/api";

type Settings = components["schemas"]["model.Settings"];

export const Route = createFileRoute("/settings/torrents")({
  component: TorrentSettingsPage,
});

function TorrentSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <TorrentSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
    />
  );
}

const torrentSettingsSchema = z.object({
  seedEnabled: z.boolean(),
  seedRatio: z.number()
    .min(0, "Ratio cannot be negative")
    .max(100, "Maximum ratio is 100"),
  seedTime: z.number()
    .min(0, "Seed time cannot be negative"),
  enableDht: z.boolean(),
  enablePex: z.boolean(),
  enableLpd: z.boolean(),
  listenPort: z.number()
    .min(1024, "Please use a non-privileged port (1024-65535)")
    .max(65535, "Port must be less than 65536"),
  maxPeers: z.number()
    .min(0, "Max peers cannot be negative")
    .max(1000, "Maximum 1000 peers recommended"),
});

function TorrentSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: Settings;
  updateServerSettings: (settings: Settings) => void;
}) {
  const navigate = useNavigate();
  const torrent = serverSettings.torrent;
  const { changeGlobalOption } = useEngineActions();

  const initialRatio = parseFloat(torrent?.seedRatio || "0");
  
  const form = useForm({
    defaultValues: {
      seedEnabled: initialRatio > 0,
      seedRatio: initialRatio > 0 ? initialRatio : 1.0,
      seedTime: Number(torrent?.seedTime || 0),
      enableDht: !!torrent?.enableDht,
      enablePex: !!torrent?.enablePex,
      enableLpd: !!torrent?.enableLpd,
      listenPort: Number(torrent?.listenPort || 6881),
      maxPeers: Number(torrent?.maxPeers || 55),
    },
    validators: {
      onChange: torrentSettingsSchema as any,
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
        torrent: {
          ...serverSettings.torrent,
          seedRatio: value.seedEnabled ? String(value.seedRatio) : "0",
          seedTime: Number(value.seedTime),
          enableDht: value.enableDht,
          enablePex: value.enablePex,
          enableLpd: value.enableLpd,
          listenPort: Number(value.listenPort),
          maxPeers: Number(value.maxPeers),
        },
      };

      try {
        await changeGlobalOption.mutateAsync({ body: updated as any });
        updateServerSettings(updated);
      } catch (err) {
        // Error toast handled by mutation
      }
    },
  });

  // Use Subscribe to access reactive state
  return (
    <div className="flex flex-col h-full space-y-6">
      <div className="flex items-center gap-4 px-2 shrink-0">
        <Button
          variant="ghost"
          isIconOnly
          onPress={() => navigate({ to: "/settings" })}
        >
          <IconChevronLeft className="w-5 h-5" />
        </Button>
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Torrents</h2>
          <p className="text-xs text-muted">BitTorrent seeding, privacy & performance</p>
        </div>
        <div className="ml-auto">
          <form.Subscribe
            selector={(state) => [state.canSubmit, state.isSubmitting, state.isDirty]}
          >
            {([canSubmit, isSubmitting, isDirty]) => (
              <Button
                variant="primary"
                onPress={() => form.handleSubmit()}
                isDisabled={!canSubmit || !isDirty}
                isPending={isSubmitting as boolean}
                className="rounded-xl font-bold"
              >
                Save Changes
              </Button>
            )}
          </form.Subscribe>
        </div>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        <ScrollShadow className="h-full">
          <div className="max-w-4xl mx-auto p-8 space-y-10">
            {/* Info Banner */}
            <Card className="p-5 bg-warning/5 border-warning/20">
              <Card.Content className="p-0">
                <p className="text-sm text-muted">
                  <span className="font-bold text-warning">ℹ️ Note:</span> These settings apply when downloading via BitTorrent (P2P)
                  instead of through a premium debrid service.
                </p>
              </Card.Content>
            </Card>

            <form.Subscribe
                selector={(state) => state.values.seedEnabled}
            >
                {(seedEnabled) => {
                    const settingGroups: SettingGroupConfig<z.infer<typeof torrentSettingsSchema>>[] = [
                        {
                            id: "seeding",
                            title: "Seeding",
                            fields: [
                                {
                                    name: "seedEnabled",
                                    type: "switch",
                                    label: "Seed After Download",
                                    description: "Share downloaded files with other peers",
                                    colSpan: 2,
                                },
                                ...(seedEnabled ? [
                                    { type: "divider" as const },
                                    {
                                        name: "seedRatio" as const,
                                        type: "number" as const, 
                                        label: "Stop Seeding at Ratio",
                                        description: "0.1 (minimal) to 5.0 (generous)",
                                        placeholder: "1.0",
                                    },
                                    {
                                        name: "seedTime" as const,
                                        type: "number" as const,
                                        label: "Maximum Seed Time (Minutes)",
                                        description: "0 = Unlimited",
                                        placeholder: "60",
                                    }
                                ] : [])
                            ]
                        },
                        {
                            id: "privacy",
                            title: "Privacy & Discovery",
                            fields: [
                                {
                                    name: "enableDht",
                                    type: "switch",
                                    label: "DHT (Distributed Hash Table)",
                                    description: "Find peers without relying on trackers",
                                    colSpan: 2,
                                },
                                { type: "divider" as const },
                                {
                                    name: "enablePex",
                                    type: "switch",
                                    label: "PEX (Peer Exchange)",
                                    description: "Share peer lists with connected peers",
                                    colSpan: 2,
                                },
                                { type: "divider" as const },
                                {
                                    name: "enableLpd",
                                    type: "switch",
                                    label: "LPD (Local Peer Discovery)",
                                    description: "Find peers on your local network",
                                    colSpan: 2,
                                }
                            ]
                        },
                        {
                            id: "connection",
                            title: "Connection",
                            fields: [
                                {
                                    name: "listenPort",
                                    type: "number",
                                    label: "Listen Port",
                                    placeholder: "6881",
                                    description: "TCP port used for incoming connections",
                                },
                                {
                                    name: "maxPeers",
                                    type: "number",
                                    label: "Max Peers",
                                    placeholder: "55",
                                    description: "Max overall number of peers per torrent",
                                }
                            ]
                        }
                    ];
                    return <DynamicSettings form={form} groups={settingGroups} />;
                }}
            </form.Subscribe>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
