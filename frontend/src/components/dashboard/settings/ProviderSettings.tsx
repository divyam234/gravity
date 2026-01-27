import { Button, Card, Input, Label, Switch, TextField } from "@heroui/react";
import type React from "react";
import { useForm } from "@tanstack/react-form";
import IconNodesDown from "~icons/gravity-ui/nodes-down";
import IconCheck from "~icons/gravity-ui/check";
import { useProviderActions, useProviders } from "../../../hooks/useProviders";
import type { components } from "../../../gen/api";

type ProviderSummary = components["schemas"]["model.ProviderSummary"];

export const ProviderSettings: React.FC = () => {
  const { data: providersResponse } = useProviders();
  const providers = providersResponse?.data || [];
  const { configureProvider } = useProviderActions();

  return (
    <div className="space-y-6">
      <div className="grid gap-6">
        {providers.map((p) => (
          <ProviderCard 
            key={p.name} 
            provider={p} 
            onSave={(config, enabled) => {
                if (p.name) {
                    configureProvider.mutate({ 
                        params: { path: { name: p.name } },
                        body: { config, enabled }
                    });
                }
            }} 
            isPending={configureProvider.isPending}
          />
        ))}
      </div>
    </div>
  );
}

function ProviderCard({ provider, onSave, isPending }: { 
    provider: ProviderSummary, 
    onSave: (config: Record<string, string>, enabled: boolean) => void, 
    isPending: boolean 
}) {
  const form = useForm({
    defaultValues: {
      config: (provider as any).config || {} as Record<string, string>,
      enabled: !!provider.enabled,
    },
    onSubmit: async ({ value }) => {
      onSave(value.config as Record<string, string>, !!value.enabled);
    },
  });

  return (
    <Card className="border border-border shadow-sm overflow-hidden">
      <Card.Header className="p-6 border-b border-border bg-default/5 flex flex-row items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-accent/10 rounded-2xl text-accent">
            <IconNodesDown className="w-6 h-6" />
          </div>
          <div>
            <Card.Title className="text-lg">{provider.displayName || provider.name}</Card.Title>
            <Card.Description className="text-xs font-medium uppercase tracking-wider">
              {provider.type} provider
            </Card.Description>
          </div>
        </div>
        <form.Field
          name="enabled"
          children={(field: any) => (
            <Switch isSelected={!!field.state.value} onChange={(v) => field.handleChange(v)} />
          )}
        />
      </Card.Header>

      <Card.Content className="p-6 space-y-6">
        {provider.name === 'alldebrid' || provider.name === 'realdebrid' ? (
          <form.Field
            name="config.api_key"
            children={(field: any) => (
              <TextField className="flex flex-col gap-2">
                <Label className="text-xs font-black uppercase tracking-widest text-muted px-1">
                  API Key / Token
                </Label>
                <Input
                  type="password"
                  placeholder="Enter your API token"
                  value={(field.state.value as string) || ''}
                  onChange={(e) => field.handleChange(e.target.value)}
                  className="w-full h-12 px-4 bg-default/10 rounded-2xl text-sm border-none focus:bg-default/15 transition-all outline-none"
                />
              </TextField>
            )}
          />
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
            onPress={() => form.handleSubmit()}
            isPending={isPending}
          >
            Save Configuration
          </Button>
        </div>
      </Card.Content>
    </Card>
  );
}