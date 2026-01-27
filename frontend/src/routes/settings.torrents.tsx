import { Button, Card, ScrollShadow } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { useEngineActions } from "../hooks/useEngine";
import { FormTextField, FormSwitch } from "../components/ui/FormFields";
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

  const form = useForm({
    defaultValues: {
      enablePeerExchange: !!torrent?.enablePex,
      enableDht: !!torrent?.enableDht,
      enableLpd: !!torrent?.enableLpd,
      seedRatio: torrent?.seedRatio || "1.0",
      seedTime: Number(torrent?.seedTime || 0),
      listenPort: Number(torrent?.listenPort || 6881),
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
        torrent: {
          ...serverSettings.torrent,
          enablePex: value.enablePeerExchange,
          enableDht: value.enableDht,
          enableLpd: value.enableLpd,
          seedRatio: value.seedRatio,
          seedTime: Number(value.seedTime),
          listenPort: Number(value.listenPort),
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
          <p className="text-xs text-muted">P2P, seeding & tracker options</p>
        </div>
        <div className="ml-auto">
          <form.Subscribe
            selector={(state) => [state.canSubmit, state.isSubmitting]}
          >
            {([canSubmit, isSubmitting]) => (
              <Button
                variant="primary"
                onPress={() => form.handleSubmit()}
                isDisabled={!canSubmit}
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
            {/* Swarm & Connectivity */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Swarm Connectivity</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <FormSwitch
                    form={form}
                    name="enableDht"
                    label="Distributed Hash Table (DHT)"
                    description="Find peers without a central tracker"
                  />
                  <div className="h-px bg-border" />
                  <FormSwitch
                    form={form}
                    name="enablePeerExchange"
                    label="Peer Exchange (PEX)"
                    description="Share peer lists directly between clients"
                  />
                  <div className="h-px bg-border" />
                  <FormSwitch
                    form={form}
                    name="enableLpd"
                    label="Local Peer Discovery (LPD)"
                    description="Find peers on your local network"
                  />
                </Card.Content>
              </Card>
            </section>

            {/* Seeding & Limits */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Seeding & Bandwidth</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <FormTextField
                      form={form}
                      name="seedRatio"
                      label="Seed Ratio"
                      placeholder="1.0"
                      description="Stop seeding at this ratio (0 to disable)"
                    />
                    <FormTextField
                      form={form}
                      name="seedTime"
                      label="Minimum Seed Time (min)"
                      type="number"
                      placeholder="0"
                      description="Seed for at least this long"
                    />
                  </div>
                  <div className="h-px bg-border" />
                  <FormTextField
                    form={form}
                    name="listenPort"
                    label="Listen Port"
                    type="number"
                    placeholder="6881"
                    description="Port for incoming P2P connections"
                  />
                </Card.Content>
              </Card>
            </section>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}