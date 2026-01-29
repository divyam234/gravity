import { 
  Button, 
  Card, 
  ScrollShadow, 
  Input, 
  Select, 
  ListBox, 
  Label, 
  Description 
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useSettingsStore } from "../store/useSettingsStore";
import { useEngineActions } from "../hooks/useEngine";
import { FormTextField, FormSwitch } from "../components/ui/FormFields";
import type { components } from "../gen/api";

type Settings = components["schemas"]["model.Settings"];

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

function ProxyListEditor({ 
  value, 
  onChange 
}: { 
  value: components["schemas"]["model.Proxy"][], 
  onChange: (val: components["schemas"]["model.Proxy"][]) => void 
}) {
  const addProxy = () => {
    onChange([...value, { url: "", type: "all" }]);
  };

  const removeProxy = (index: number) => {
    onChange(value.filter((_, i) => i !== index));
  };

  const updateProxy = (index: number, updates: Partial<components["schemas"]["model.Proxy"]>) => {
    onChange(value.map((p, i) => i === index ? { ...p, ...updates } : p));
  };

  return (
    <div className="space-y-6">
      {value.map((proxy, index) => (
        <Card key={`${proxy.url}-${index}`} className="p-6 bg-default-50 border-default-200">
          <div className="flex flex-col md:flex-row gap-6 items-start">
            <div className="flex-1 w-full space-y-2">
              <Label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">
                Proxy URL
              </Label>
              <Input
                value={proxy.url || ""}
                onChange={(e) => updateProxy(index, { url: e.target.value })}
                placeholder="http://user:pass@host:port"
                className="w-full"
              />
              <Description>Include protocol (e.g. http:// or socks5://)</Description>
            </div>
            
            <div className="w-full md:w-48 space-y-2">
              <Label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground ml-1">
                Scope
              </Label>
              <Select 
                placeholder="Select scope"
                value={proxy.type || "all"}
                onChange={(val) => updateProxy(index, { type: val as any })}
              >
                <Select.Trigger>
                  <Select.Value />
                  <Select.Indicator />
                </Select.Trigger>
                <Select.Popover>
                  <ListBox>
                    <ListBox.Item id="all" textValue="Global">Global (All)</ListBox.Item>
                    <ListBox.Item id="downloads" textValue="Downloads">Downloads Only</ListBox.Item>
                    <ListBox.Item id="uploads" textValue="Uploads">Uploads Only</ListBox.Item>
                    <ListBox.Item id="magnets" textValue="Magnets">Magnets Only</ListBox.Item>
                  </ListBox>
                </Select.Popover>
              </Select>
            </div>

            <Button
              isIconOnly
              variant="ghost"
              className="mt-6 text-danger"
              onPress={() => removeProxy(index)}
            >
              <IconTrashBin className="w-5 h-5" />
            </Button>
          </div>
        </Card>
      ))}
      
      <Button
        variant="secondary"
        className="w-full h-16 border-dashed border-2 bg-transparent hover:bg-default-100 font-bold"
        onPress={addProxy}
      >
        + Add Proxy Server
      </Button>
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
      proxies: network?.proxies || [],
      
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
              proxies: value.proxies,
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
            {/* Proxy Servers */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Proxy Servers</h3>
              </div>
              <form.Field name="proxies">
                {(field: any) => (
                  <ProxyListEditor 
                    value={field.state.value} 
                    onChange={field.handleChange} 
                  />
                )}
              </form.Field>
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