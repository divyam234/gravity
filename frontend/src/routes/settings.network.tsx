import { Button, Card, ScrollShadow, Tabs } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { useEngineActions } from "../hooks/useEngine";
import { FormTextField, FormSwitch } from "../components/ui/FormFields";
import type { components } from "../gen/api";

type Settings = components["schemas"]["model.Settings"];
type ProxyConfig = components["schemas"]["model.ProxyConfig"];

export const Route = createFileRoute("/settings/network")({
  component: NetworkSettingsPage,
});

function NetworkSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <NetworkSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
    />
  );
}

// Helper to parse/format proxy URL
const parseProxyUrl = (url: string) => {
  if (!url) return { type: "http", host: "", port: "1080" }; // Default to http to show placeholders
  const match = url.match(/^(socks5|http):\/\/([^:]+):(\d+)/);
  if (match) {
    return { type: match[1], host: match[2], port: match[3] };
  }
  return { type: "http", host: "", port: "1080" };
};

const formatProxyUrl = (type: string, host: string, port: string) => {
  if (!host) return "";
  return `${type}://${host}:${port}`;
};

// Reusable Proxy Configuration Component
function ProxyConfigEditor({
  value,
  onChange
}: {
  value: ProxyConfig,
  onChange: (val: ProxyConfig) => void
}) {
  const parsed = parseProxyUrl(value.url || "");
  
  const update = (updates: { type?: string, host?: string, port?: string, user?: string, password?: string, enabled?: boolean }) => {
    const newType = updates.type ?? parsed.type;
    const newHost = updates.host ?? parsed.host;
    const newPort = updates.port ?? parsed.port;
    const newUser = updates.user ?? value.user;
    const newPass = updates.password ?? value.password;
    const newEnabled = updates.enabled ?? value.enabled;

    onChange({
      enabled: !!newEnabled,
      url: formatProxyUrl(newType, newHost, newPort),
      user: newUser,
      password: newPass,
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <span className="text-sm font-bold">Enable Proxy</span>
        <Button
            size="sm"
            variant={value.enabled ? "primary" : "secondary"}
            onPress={() => update({ enabled: !value.enabled })}
            className="capitalize"
        >
            {value.enabled ? "Enabled" : "Disabled"}
        </Button>
      </div>

      {value.enabled && (
        <div className="space-y-6 animate-in slide-in-from-top-2 duration-200">
            <div className="space-y-3">
                <span className="text-sm font-bold">Protocol</span>
                <div className="flex gap-2">
                    {(["http", "socks5"] as const).map((type) => (
                        <Button
                            key={type}
                            size="sm"
                            variant={parsed.type === type ? "primary" : "secondary"}
                            onPress={() => update({ type })}
                            className="rounded-xl font-bold capitalize"
                        >
                            {type.toUpperCase()}
                        </Button>
                    ))}
                </div>
            </div>

            <div className="grid grid-cols-3 gap-4">
                <div className="col-span-2">
                    <div className="space-y-1">
                        <span className="text-xs font-bold text-muted uppercase tracking-wider ml-1">Host</span>
                        <input
                            value={parsed.host}
                            onChange={(e) => update({ host: e.target.value })}
                            placeholder="proxy.example.com"
                            className="w-full h-11 bg-default/10 rounded-2xl border-none font-mono text-xs px-3 outline-none focus:ring-2 ring-accent/50 transition-all"
                        />
                    </div>
                </div>
                <div className="space-y-1">
                    <span className="text-xs font-bold text-muted uppercase tracking-wider ml-1">Port</span>
                    <input
                        value={parsed.port}
                        onChange={(e) => update({ port: e.target.value })}
                        placeholder="1080"
                        className="w-full h-11 bg-default/10 rounded-2xl border-none font-mono text-xs px-3 outline-none focus:ring-2 ring-accent/50 transition-all"
                    />
                </div>
            </div>

            <div className="h-px bg-border" />

            <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                    <span className="text-xs font-bold text-muted uppercase tracking-wider ml-1">Username</span>
                    <input
                        value={value.user || ""}
                        onChange={(e) => update({ user: e.target.value })}
                        placeholder="Optional"
                        className="w-full h-11 bg-default/10 rounded-2xl border-none font-mono text-xs px-3 outline-none focus:ring-2 ring-accent/50 transition-all"
                    />
                </div>
                <div className="space-y-1">
                    <span className="text-xs font-bold text-muted uppercase tracking-wider ml-1">Password</span>
                    <input
                        value={value.password || ""}
                        onChange={(e) => update({ password: e.target.value })}
                        type="password"
                        placeholder="Optional"
                        className="w-full h-11 bg-default/10 rounded-2xl border-none font-mono text-xs px-3 outline-none focus:ring-2 ring-accent/50 transition-all"
                    />
                </div>
            </div>
        </div>
      )}
    </div>
  );
}

function NetworkSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: Settings;
  updateServerSettings: (settings: Settings) => void;
}) {
  const navigate = useNavigate();
  const network = serverSettings.network;
  const download = serverSettings.download;
  const { changeGlobalOption } = useEngineActions();

  const form = useForm({
    defaultValues: {
      proxyMode: network?.proxyMode || "global",
      globalProxy: network?.globalProxy || { enabled: false, url: "", user: "", password: "" },
      magnetProxy: network?.magnetProxy || { enabled: false, url: "", user: "", password: "" },
      downloadProxy: network?.downloadProxy || { enabled: false, url: "", user: "", password: "" },
      uploadProxy: network?.uploadProxy || { enabled: false, url: "", user: "", password: "" },
      
      dnsOverHttps: network?.dnsOverHttps || "",
      interfaceBinding: network?.interfaceBinding || "",
      tcpPortRange: network?.tcpPortRange || "",
      maxConnectionPerServer: Number(download?.maxConnectionPerServer || 8),
      connectTimeout: Number(download?.connectTimeout || 60),
      maxTries: Number(download?.maxTries || 0),
      checkCertificate: !!download?.checkCertificate,
      userAgent: download?.userAgent || "",
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
          network: {
            ...serverSettings.network,
            proxyMode: value.proxyMode as any,
            globalProxy: value.globalProxy,
            magnetProxy: value.magnetProxy,
            downloadProxy: value.downloadProxy,
            uploadProxy: value.uploadProxy,
            dnsOverHttps: value.dnsOverHttps,
            interfaceBinding: value.interfaceBinding,
            tcpPortRange: value.tcpPortRange,
          },
        download: {
          ...serverSettings.download,
          downloadDir: serverSettings.download?.downloadDir || "/downloads",
          preferredEngine: serverSettings.download?.preferredEngine || "aria2",
          preferredMagnetEngine: serverSettings.download?.preferredMagnetEngine || "aria2",
          maxConnectionPerServer: Number(value.maxConnectionPerServer),
          connectTimeout: Number(value.connectTimeout),
          maxTries: Number(value.maxTries),
          checkCertificate: value.checkCertificate,
          userAgent: value.userAgent,
        } as any,
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
          <h2 className="text-2xl font-bold tracking-tight">Network</h2>
          <p className="text-xs text-muted">Proxy, connections & security</p>
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
            {/* Proxy */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Proxy Configuration</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6">
                    <form.Field name="proxyMode">
                        {(field: any) => (
                            <div className="flex justify-center mb-8">
                                <div className="bg-default/10 p-1 rounded-2xl flex">
                                    <button
                                        type="button"
                                        onClick={() => field.handleChange("global")}
                                        className={`px-6 py-2 rounded-xl text-sm font-bold transition-all ${
                                            field.state.value === "global" 
                                            ? "bg-background shadow text-foreground" 
                                            : "text-muted hover:text-foreground"
                                        }`}
                                    >
                                        Global
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => field.handleChange("granular")}
                                        className={`px-6 py-2 rounded-xl text-sm font-bold transition-all ${
                                            field.state.value === "granular" 
                                            ? "bg-background shadow text-foreground" 
                                            : "text-muted hover:text-foreground"
                                        }`}
                                    >
                                        Per-Protocol
                                    </button>
                                </div>
                            </div>
                        )}
                    </form.Field>

                    <form.Subscribe
                        selector={(state: any) => [state.values.proxyMode]}
                    >
                        {([proxyMode]) => (
                            proxyMode === "global" ? (
                                <form.Field name="globalProxy">
                                    {(field: any) => (
                                        <ProxyConfigEditor 
                                            value={field.state.value} 
                                            onChange={field.handleChange} 
                                        />
                                    )}
                                </form.Field>
                            ) : (
                                <div className="space-y-8">
                                    <Tabs className="w-full">
                                        <Tabs.ListContainer>
                                            <Tabs.List aria-label="Proxy Protocols">
                                                <Tabs.Tab id="magnet">Magnet / Torrent<Tabs.Indicator /></Tabs.Tab>
                                                <Tabs.Tab id="download">File Download<Tabs.Indicator /></Tabs.Tab>
                                                <Tabs.Tab id="upload">Upload<Tabs.Indicator /></Tabs.Tab>
                                            </Tabs.List>
                                        </Tabs.ListContainer>

                                        <Tabs.Panel id="magnet">
                                            <div className="pt-4">
                                                <form.Field name="magnetProxy">
                                                    {(field: any) => (
                                                        <ProxyConfigEditor 
                                                            value={field.state.value} 
                                                            onChange={field.handleChange} 
                                                        />
                                                    )}
                                                </form.Field>
                                            </div>
                                        </Tabs.Panel>
                                        <Tabs.Panel id="download">
                                            <div className="pt-4">
                                                <form.Field name="downloadProxy">
                                                    {(field: any) => (
                                                        <ProxyConfigEditor 
                                                            value={field.state.value} 
                                                            onChange={field.handleChange} 
                                                        />
                                                    )}
                                                </form.Field>
                                            </div>
                                        </Tabs.Panel>
                                        <Tabs.Panel id="upload">
                                            <div className="pt-4">
                                                <form.Field name="uploadProxy">
                                                    {(field: any) => (
                                                        <ProxyConfigEditor 
                                                            value={field.state.value} 
                                                            onChange={field.handleChange} 
                                                        />
                                                    )}
                                                </form.Field>
                                            </div>
                                        </Tabs.Panel>
                                    </Tabs>
                                </div>
                            )
                        )}
                    </form.Subscribe>
                </Card.Content>
              </Card>
            </section>

            {/* Connection Limits */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Connection Limits</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <FormTextField
                            form={form}
                            name="maxConnectionPerServer"
                            label="Max Connections per Server"
                            type="number"
                            placeholder="8"
                        />
                        <FormTextField
                            form={form}
                            name="connectTimeout"
                            label="Connection Timeout (sec)"
                            type="number"
                            placeholder="60"
                        />
                    </div>
                    <div className="h-px bg-border" />
                    <FormTextField
                        form={form}
                        name="maxTries"
                        label="Max Retries"
                        type="number"
                        placeholder="0 (Unlimited)"
                    />
                </Card.Content>
              </Card>
            </section>

            {/* Security */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Security</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <FormSwitch
                        form={form}
                        name="checkCertificate"
                        label="Verify SSL/TLS Certificates"
                        description="Reject connections with invalid certificates"
                    />
                    <div className="h-px bg-border" />
                    <FormTextField
                        form={form}
                        name="userAgent"
                        label="User-Agent"
                        placeholder="gravity/1.0"
                        description="The User-Agent header sent with HTTP requests"
                    />
                </Card.Content>
              </Card>
            </section>

            {/* DNS & Connectivity */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">DNS & Connectivity</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <FormTextField
                        form={form}
                        name="dnsOverHttps"
                        label="DNS over HTTPS (DoH)"
                        placeholder="https://cloudflare-dns.com/dns-query"
                        description="Secure DNS resolver URL"
                    />
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <FormTextField
                            form={form}
                            name="interfaceBinding"
                            label="Interface Binding"
                            placeholder="eth0"
                            description="Bind to specific network interface"
                        />
                        <FormTextField
                            form={form}
                            name="tcpPortRange"
                            label="TCP Port Range"
                            placeholder="6881-6999"
                            description="Range of ports for direct connections"
                        />
                    </div>
                </Card.Content>
              </Card>
            </section>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
