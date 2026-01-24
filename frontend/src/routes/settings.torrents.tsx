import { Button, Card, ScrollShadow, Slider } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { api } from "../lib/api";
import { FormTextField, FormSwitch } from "../components/ui/FormFields";
import { toast } from "sonner";

export const Route = createFileRoute("/settings/torrents")({
  component: TorrentsSettingsPage,
});

function TorrentsSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <TorrentsSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
    />
  );
}

function TorrentsSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: any;
  updateServerSettings: any;
}) {
  const navigate = useNavigate();
  const { torrent } = serverSettings;

  const initialRatio = parseFloat(torrent.seedRatio || "0");
  
  const form = useForm({
    defaultValues: {
      seedEnabled: initialRatio > 0,
      seedRatio: initialRatio > 0 ? initialRatio : 1.0,
      seedTime: Number(torrent.seedTime),
      encryption: torrent.encryption,
      enableDht: torrent.enableDht,
      enablePex: torrent.enablePex,
      enableLpd: torrent.enableLpd,
      listenPort: Number(torrent.listenPort),
      forceSave: torrent.forceSave,
      maxPeers: Number(torrent.maxPeers),
    },
    onSubmit: async ({ value }) => {
      const updated = {
        ...serverSettings,
        torrent: {
          ...serverSettings.torrent,
          seedRatio: value.seedEnabled ? String(value.seedRatio) : "0",
          seedTime: Number(value.seedTime),
          encryption: value.encryption,
          enableDht: value.enableDht,
          enablePex: value.enablePex,
          enableLpd: value.enableLpd,
          listenPort: Number(value.listenPort),
          forceSave: value.forceSave,
          maxPeers: Number(value.maxPeers),
        },
      };

      try {
        await api.updateSettings(updated);
        updateServerSettings(updated);
        toast.success("Settings saved successfully");
      } catch (err) {
        console.error(err);
        toast.error("Failed to save settings");
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
          <p className="text-xs text-muted">BitTorrent seeding, privacy & performance</p>
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
                        isPending={isSubmitting}
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
              <p className="text-sm text-muted">
                <span className="font-bold text-warning">ℹ️ Note:</span> These settings apply when downloading via BitTorrent (P2P)
                instead of through a premium debrid service.
              </p>
            </Card>

            {/* Seeding */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Seeding</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <FormSwitch
                        form={form}
                        name="seedEnabled"
                        label="Seed After Download"
                        description="Share downloaded files with other peers"
                    />

                    <form.Subscribe
                        selector={(state) => [state.values.seedEnabled]}
                    >
                        {([seedEnabled]) => (
                            seedEnabled && (
                                <div className="space-y-6 animate-in slide-in-from-top-2 duration-200">
                                    <div className="h-px bg-border" />
                                    
                                    <form.Field name="seedRatio">
                                        {(field) => (
                                            <div className="space-y-4">
                                                <div className="flex items-center justify-between">
                                                    <span className="text-sm font-bold">Stop Seeding at Ratio</span>
                                                    <span className="text-sm font-bold text-accent bg-accent/10 px-3 py-1 rounded-lg">
                                                        {field.state.value.toFixed(1)}
                                                    </span>
                                                </div>
                                                <Slider
                                                    value={field.state.value}
                                                    onChange={(val) => field.handleChange(val as number)}
                                                    minValue={0.1}
                                                    maxValue={5.0}
                                                    step={0.1}
                                                >
                                                    <Slider.Track className="h-2 bg-default/10">
                                                        <Slider.Fill className="bg-accent" />
                                                        <Slider.Thumb className="w-5 h-5 border-2 border-accent bg-background" />
                                                    </Slider.Track>
                                                </Slider>
                                                <div className="flex justify-between text-xs text-muted">
                                                    <span>0.1 (minimal)</span>
                                                    <span>1.0 (fair share)</span>
                                                    <span>5.0 (generous)</span>
                                                </div>
                                            </div>
                                        )}
                                    </form.Field>

                                    <div className="h-px bg-border" />

                                    <FormTextField
                                        form={form}
                                        name="seedTime"
                                        label="Maximum Seed Time (Minutes)"
                                        type="number"
                                        placeholder="60"
                                        description="0 = Unlimited"
                                    />
                                </div>
                            )
                        )}
                    </form.Subscribe>
                </Card.Content>
              </Card>
            </section>

            {/* Privacy & Encryption */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Privacy & Encryption</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <form.Field name="encryption">
                        {(field) => (
                            <div className="space-y-3">
                                <span className="text-sm font-bold">Protocol Encryption</span>
                                <div className="flex gap-2">
                                    {([
                                        { id: "disabled", label: "Disabled" },
                                        { id: "plain", label: "Prefer Encrypted" },
                                        { id: "required", label: "Require Encrypted" },
                                    ] as const).map((opt) => (
                                        <Button
                                            key={opt.id}
                                            size="sm"
                                            variant={field.state.value === opt.id ? "primary" : "secondary"}
                                            onPress={() => field.handleChange(opt.id)}
                                            className="rounded-xl font-bold"
                                        >
                                            {opt.label}
                                        </Button>
                                    ))}
                                </div>
                                <p className="text-xs text-muted">
                                    Encrypted connections help prevent ISP throttling
                                </p>
                            </div>
                        )}
                    </form.Field>

                    <div className="h-px bg-border" />
                    <FormSwitch
                        form={form}
                        name="enableDht"
                        label="DHT (Distributed Hash Table)"
                        description="Find peers without relying on trackers"
                    />
                    <div className="h-px bg-border" />
                    <FormSwitch
                        form={form}
                        name="enablePex"
                        label="PEX (Peer Exchange)"
                        description="Share peer lists with connected peers"
                    />
                    <div className="h-px bg-border" />
                    <FormSwitch
                        form={form}
                        name="enableLpd"
                        label="LPD (Local Peer Discovery)"
                        description="Find peers on your local network"
                    />
                </Card.Content>
              </Card>
            </section>

            {/* Listening Port */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Listening Port</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <FormTextField
                            form={form}
                            name="listenPort"
                            label="Listen Port"
                            placeholder="6881"
                            type="number"
                        />
                        <FormTextField
                            form={form}
                            name="maxPeers"
                            label="Max Peers"
                            type="number"
                            placeholder="55"
                        />
                    </div>
                    <div className="h-px bg-border" />
                    <FormSwitch
                        form={form}
                        name="forceSave"
                        label="Force Save Resume Data"
                        description="Ensures resume data is saved frequently to prevent data loss"
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
