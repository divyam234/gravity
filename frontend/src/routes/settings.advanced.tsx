import { Button, Card, ScrollShadow } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { useEngineActions } from "../hooks/useEngine";
import { FormTextField, FormSwitch, FormSelect } from "../components/ui/FormFields";
import type { components } from "../gen/api";

type Settings = components["schemas"]["model.Settings"];

export const Route = createFileRoute("/settings/advanced")({
  component: AdvancedSettingsPage,
});

function AdvancedSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <AdvancedSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
    />
  );
}

function AdvancedSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: Settings;
  updateServerSettings: (settings: Settings) => void;
}) {
  const navigate = useNavigate();
  const advanced = serverSettings.advanced;
  const { changeGlobalOption } = useEngineActions();

  const form = useForm({
    defaultValues: {
      logLevel: advanced?.logLevel || "info",
      debugMode: !!advanced?.debugMode,
      saveInterval: Number(advanced?.saveInterval || 60),
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
        advanced: {
          ...serverSettings.advanced,
          logLevel: value.logLevel as NonNullable<Settings["advanced"]>["logLevel"],
          debugMode: value.debugMode,
          saveInterval: Number(value.saveInterval),
        },
      };

      try {
        await changeGlobalOption.mutateAsync({ body: updated });
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
          <h2 className="text-2xl font-bold tracking-tight">Advanced</h2>
          <p className="text-xs text-muted">
            System internals & debugging
          </p>
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
            {/* System Internals */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-danger rounded-full" />
                <h3 className="text-lg font-bold">System Internals</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <FormSelect
                        form={form}
                        name="logLevel"
                        label="Log Level"
                        items={[
                            { value: "debug", label: "Debug (Verbose)" },
                            { value: "info", label: "Info (Standard)" },
                            { value: "warn", label: "Warning" },
                            { value: "error", label: "Error" },
                        ]}
                    />
                    <FormTextField
                        form={form}
                        name="saveInterval"
                        label="Save Interval (Seconds)"
                        type="number"
                        description="How often to save session state to disk"
                    />
                  </div>
                  
                  <div className="h-px bg-border" />

                  <FormSwitch
                    form={form}
                    name="debugMode"
                    label="Enable Debug Mode"
                    description="Enables additional runtime checks and logging. Recommended only for troubleshooting."
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