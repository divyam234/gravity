import { Button, Card, Input, Label, Switch } from "@heroui/react";
import { useState } from "react";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import IconCheck from "~icons/gravity-ui/check";
import { useProviderActions, useProviders } from "../../../hooks/useProviders";

export const ProviderSettings: React.FC = () => {
  const { data: providers } = useProviders();
  const { configure } = useProviderActions();

  return (
    <div className="space-y-6">
      <div className="grid gap-6">
        {providers?.data?.map((p) => (
          <ProviderCard 
            key={p.name} 
            provider={p} 
            onSave={(config, enabled) => configure.mutate({ name: p.name, config, enabled })} 
            isPending={configure.isPending}
          />
        ))}
      </div>
    </div>
  );
}

function ProviderCard({ provider, onSave, isPending }: { provider: any, onSave: (config: any, enabled: boolean) => void, isPending: boolean }) {
  const [config, setConfig] = useState<Record<string, string>>(provider.config || {});
  const [enabled, setEnabled] = useState(provider.enabled);

  return (
    <Card className="border border-border shadow-sm overflow-hidden">
      <div className="p-0">
        <div className="p-6 border-b border-border bg-default/5 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-accent/10 rounded-2xl text-accent">
              <IconNodesDown className="w-6 h-6" />
            </div>
            <div>
              <h3 className="text-lg font-bold">{provider.displayName}</h3>
              <p className="text-xs text-muted font-medium uppercase tracking-wider">
                {provider.type} provider
              </p>
            </div>
          </div>
          <Switch isSelected={enabled} onChange={setEnabled} />
        </div>

        <div className="p-6 space-y-6">
          {provider.name === 'alldebrid' || provider.name === 'realdebrid' ? (
            <div className="flex flex-col gap-2">
              <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                API Key / Token
              </Label>
              <Input
                type="password"
                placeholder="Enter your API token"
                value={config.api_key || ''}
                onChange={(e) => setConfig({ ...config, api_key: e.target.value })}
                className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
              />
            </div>
          ) : (
            <p className="text-sm text-muted italic">This provider does not require configuration.</p>
          )}

          {provider.account && (
            <div className="bg-success/5 border border-success/20 p-4 rounded-2xl flex items-center gap-3">
              <IconCheck className="w-5 h-5 text-success" />
              <div>
                <p className="text-xs font-bold text-success">
                  Connected as {provider.account.username}
                </p>
                {provider.account.isPremium && (
                  <p className="text-[10px] uppercase font-black tracking-widest text-success opacity-80">
                    Premium Active
                  </p>
                )}
              </div>
            </div>
          )}

          <div className="flex justify-end">
            <Button
              className="px-8 h-10 rounded-xl font-bold bg-accent text-accent-foreground shadow-lg shadow-accent/20"
              onPress={() => onSave(config, enabled)}
              isDisabled={isPending}
            >
              {isPending ? "Saving..." : "Save Configuration"}
            </Button>
          </div>
        </div>
      </div>
    </Card>
  );
}
