import {
  Button,
  Card,
  Input,
  Label,
  ScrollShadow,
  Switch,
  Spinner,
  Chip,
} from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconThunderbolt from "~icons/gravity-ui/thunderbolt";
import IconArrowUpRightFromSquare from "~icons/gravity-ui/arrow-up-right-from-square";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useProviders, useProviderActions } from "../hooks/useProviders";
import { cn } from "../lib/utils";
import { toast } from "sonner";
import type { components } from "../gen/api";

type ProviderSummary = components["schemas"]["model.ProviderSummary"];

export const Route = createFileRoute("/settings/premium")({
  component: PremiumServicesPage,
});

interface ProviderDefinition {
    id: string;
    name: string;
    description: string;
    color: string;
    icon: string;
    website: string;
    configFields: {
        key: string;
        label: string;
        type: string;
        required?: boolean;
        placeholder?: string;
    }[];
}

// Premium provider definitions
const PREMIUM_PROVIDERS: ProviderDefinition[] = [
  {
    id: "alldebrid",
    name: "AllDebrid",
    description:
      "Fast debrid service with 80+ hosts and instant torrent caching",
    color: "bg-green-500/10 text-green-500 border-green-500/20",
    icon: "AD",
    website: "https://alldebrid.com",
    configFields: [
      { key: "api_key", label: "API Key", type: "password", required: true },
      { key: "proxy_url", label: "Proxy URL", type: "text", placeholder: "http://user:pass@host:port" }
    ],
  },
  {
    id: "realdebrid",
    name: "Real-Debrid",
    description: "Premium link generator with 100+ hosts support",
    color: "bg-red-500/10 text-red-500 border-red-500/20",
    icon: "RD",
    website: "https://real-debrid.com",
    configFields: [
      { key: "api_key", label: "API Key", type: "password", required: true },
      { key: "proxy_url", label: "Proxy URL", type: "text", placeholder: "http://user:pass@host:port" }
    ],
  },
  {
    id: "premiumize",
    name: "Premiumize",
    description: "Cloud downloader with VPN and Usenet support",
    color: "bg-orange-500/10 text-orange-500 border-orange-500/20",
    icon: "PM",
    website: "https://premiumize.me",
    configFields: [
      { key: "api_key", label: "API Key", type: "password", required: true },
      { key: "proxy_url", label: "Proxy URL", type: "text", placeholder: "http://user:pass@host:port" }
    ],
  },
  {
    id: "debridlink",
    name: "Debrid-Link",
    description: "European debrid service with competitive pricing",
    color: "bg-blue-500/10 text-blue-500 border-blue-500/20",
    icon: "DL",
    website: "https://debrid-link.com",
    configFields: [
      { key: "api_key", label: "API Key", type: "password", required: true },
      { key: "proxy_url", label: "Proxy URL", type: "text", placeholder: "http://user:pass@host:port" }
    ],
  },
  {
    id: "torbox",
    name: "TorBox",
    description: "All-in-one torrent, Usenet, and debrid service",
    color: "bg-purple-500/10 text-purple-500 border-purple-500/20",
    icon: "TB",
    website: "https://torbox.app",
    configFields: [
      { key: "api_key", label: "API Key", type: "password", required: true },
      { key: "proxy_url", label: "Proxy URL", type: "text", placeholder: "http://user:pass@host:port" }
    ],
  },
  {
    id: "megadebrid",
    name: "MegaDebrid.eu",
    description: "Premium link generator with high speed and multiple hosters",
    color: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
    icon: "MD",
    website: "https://www.mega-debrid.eu",
    configFields: [
      { key: "username", label: "Username", type: "text" },
      { key: "password", label: "Password", type: "password" },
      { key: "token", label: "API Token (Optional)", type: "password", placeholder: "Permanent API Token" },
      { key: "proxy_url", label: "Proxy URL", type: "text", placeholder: "http://user:pass@host:port" }
    ],
  },
];

function PremiumServicesPage() {
  const navigate = useNavigate();
  const { data: providersResponse, isLoading } = useProviders();
  const { configureProvider } = useProviderActions();

  // Preferences (mock state for now, should be connected to backend settings)
  const [usePremium, setUsePremium] = useState(true);
  const [usePremiumForMagnets, setUsePremiumForMagnets] = useState(true);

  const providers = providersResponse?.data || [];

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
          <h2 className="text-2xl font-bold tracking-tight">
            Premium Services
          </h2>
          <p className="text-xs text-muted">
            Debrid providers & link resolvers
          </p>
        </div>
      </div>

      <div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
        <ScrollShadow className="h-full">
          <div className="max-w-4xl mx-auto p-8 space-y-10">
            {/* Info Banner */}
            <Card className="p-5 bg-accent/5 border-accent/20">
              <div className="flex items-start gap-4">
                <div className="p-2.5 rounded-xl bg-accent/10">
                  <IconThunderbolt className="w-5 h-5 text-accent" />
                </div>
                <div>
                  <h4 className="font-bold mb-1">Supercharge Downloads</h4>
                  <p className="text-sm text-muted">
                    Connect your debrid account to unlock high-speed downloads from file hosts and instant torrent caching.
                    Now with proxy support for better connectivity.
                  </p>
                </div>
              </div>
            </Card>

            {/* Preferences */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Preferences</h3>
              </div>
              <Card className="p-6 bg-background/50 border-border space-y-5">
                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-sm font-bold">
                      Use premium services
                    </Label>
                    <p className="text-xs text-muted mt-0.5">
                      Automatically use debrid for supported links
                    </p>
                  </div>
                  <Switch isSelected={usePremium} onChange={setUsePremium}>
                    <Switch.Control>
                      <Switch.Thumb />
                    </Switch.Control>
                  </Switch>
                </div>

                <div className="h-px bg-border" />

                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-sm font-bold">
                      Cache Torrents
                    </Label>
                    <p className="text-xs text-muted mt-0.5">
                      Download cached torrents via debrid instead of P2P
                    </p>
                  </div>
                  <Switch
                    isSelected={usePremiumForMagnets}
                    onChange={setUsePremiumForMagnets}
                  >
                    <Switch.Control>
                      <Switch.Thumb />
                    </Switch.Control>
                  </Switch>
                </div>
              </Card>
            </section>

            {/* Providers Grid */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-primary rounded-full" />
                <h3 className="text-lg font-bold">Providers</h3>
              </div>

              {isLoading ? (
                <div className="flex justify-center py-12">
                  <Spinner size="md" />
                </div>
              ) : (
                <div className="grid grid-cols-1 gap-6">
                  {PREMIUM_PROVIDERS.map((def) => {
                    const connectedProvider = providers.find(
                      (p) => p.name === def.id
                    );
                    
                    return (
                      <ProviderCard
                        key={def.id}
                        def={def}
                        provider={connectedProvider}
                        onConfigure={(config: Record<string, string>, enabled: boolean) => configureProvider.mutate({ 
                            params: { path: { name: def.id } },
                            body: { config, enabled }
                        })}
                        isPending={configureProvider.isPending}
                      />
                    );
                  })}
                </div>
              )}
            </section>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}

interface ProviderCardProps {
    def: ProviderDefinition;
    provider?: ProviderSummary;
    onConfigure: (config: Record<string, string>, enabled: boolean) => void;
    isPending: boolean;
}

function ProviderCard({ def, provider, onConfigure, isPending }: ProviderCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const isConnected = provider?.enabled;

  const form = useForm({
    defaultValues: def.configFields.reduce((acc: Record<string, string>, field) => {
      acc[field.key] = (provider as any)?.config?.[field.key] || "";
      return acc;
    }, {}),
    onSubmit: async ({ value }) => {
      onConfigure(value, true);
      toast.success(`${def.name} configuration saved`);
      setIsExpanded(false);
    },
  });

  const handleDisconnect = () => {
    if (confirm(`Disconnect ${def.name}?`)) {
      onConfigure({}, false);
      form.reset();
    }
  };

  return (
    <Card className={cn(
        "bg-background/50 border-border overflow-hidden transition-all duration-300",
        isConnected && "border-success/30 bg-success/5"
    )}>
      <div className="p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-5">
            <div className={cn(
              "w-14 h-14 rounded-2xl flex items-center justify-center font-black text-xl shadow-sm",
              isConnected ? "bg-success/10 text-success" : def.color
            )}>
              {def.icon}
            </div>
            <div>
              <div className="flex items-center gap-3">
                <h4 className="font-bold text-lg">{def.name}</h4>
                {isConnected ? (
                  <Chip size="sm" variant="soft" color="success" className="font-bold uppercase tracking-wider text-[10px] h-5">
                    Connected
                  </Chip>
                ) : (
                  <Chip size="sm" variant="soft" className="font-bold uppercase tracking-wider text-[10px] h-5 opacity-50">
                    Not Configured
                  </Chip>
                )}
              </div>
              <p className="text-sm text-muted mt-1 max-w-md line-clamp-1">
                {def.description}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <Button
                size="sm"
                variant="ghost"
                isIconOnly
                onPress={() => window.open(def.website, "_blank")}
                className="rounded-xl text-muted hover:text-foreground"
            >
                <IconArrowUpRightFromSquare className="w-4 h-4" />
            </Button>
            <Button
              variant={isConnected ? "secondary" : "primary"}
              className="rounded-xl font-bold"
              onPress={() => setIsExpanded(!isExpanded)}
            >
              {isConnected ? "Configure" : "Connect"}
            </Button>
          </div>
        </div>

        {/* Inline Configuration Form */}
        {isExpanded && (
            <div className="mt-6 pt-6 border-t border-border animate-in slide-in-from-top-2 duration-200">
                {def.configFields.map((field) => (
                    <form.Field key={field.key} name={field.key}>
                        {(f: any) => (
                            <div className="mb-4 last:mb-6">
                                <Label className="text-sm font-bold mb-1.5 block">{field.label}</Label>
                                <Input 
                                    type={field.type || "text"}
                                    value={f.state.value} 
                                    onChange={(e) => f.handleChange(e.target.value)}
                                    className={cn(
                                        "bg-default/10 border-none rounded-xl",
                                        field.key === "proxy_url" && "font-mono text-xs"
                                    )}
                                    placeholder={field.placeholder || `Enter ${field.label}`}
                                />
                                {field.key === "api_key" && (
                                    <p className="text-xs text-muted mt-1.5 flex items-center gap-1">
                                        Need a key? <a href={def.website} target="_blank" rel="noreferrer" className="text-accent hover:underline">Get it here</a>
                                    </p>
                                )}
                                {field.key === "proxy_url" && (
                                    <p className="text-xs text-muted mt-1.5">
                                        Use a proxy if the service is blocked in your region. Supports HTTP/HTTPS/SOCKS5.
                                    </p>
                                )}
                            </div>
                        )}
                    </form.Field>
                ))}

                <div className="flex items-center justify-between">
                    {isConnected ? (
                        <Button 
                            variant="ghost" 
                            className="rounded-xl font-bold text-danger"
                            onPress={handleDisconnect}
                        >
                            <IconTrashBin className="w-4 h-4 mr-2" />
                            Disconnect
                        </Button>
                    ) : <div />}
                    
                    <div className="flex gap-3">
                        <Button 
                            variant="ghost" 
                            className="rounded-xl font-bold"
                            onPress={() => setIsExpanded(false)}
                        >
                            Cancel
                        </Button>
                        <form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting]}>
                            {(state) => {
                                const canSubmit = state[0];
                                const isSubmitting = state[1];
                                return (
                                    <Button 
                                        variant="primary" 
                                        className="rounded-xl font-bold px-6"
                                        onPress={form.handleSubmit}
                                        isDisabled={!canSubmit}
                                        isPending={isPending || isSubmitting}
                                    >
                                        Save Configuration
                                    </Button>
                                );
                            }}
                        </form.Subscribe>
                    </div>
                </div>
            </div>
        )}
      </div>
    </Card>
  );
}