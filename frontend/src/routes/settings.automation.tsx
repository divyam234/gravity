import { Button, Card, ScrollShadow } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import { useSettingsStore } from "../store/useSettingsStore";
import { api } from "../lib/api";
import { FormTextField, FormSwitch, FormSelect } from "../components/ui/FormFields";
import { toast } from "sonner";
import type { Settings } from "../lib/types";

export const Route = createFileRoute("/settings/automation")({
  component: AutomationSettingsPage,
});

function AutomationSettingsPage() {
  const { serverSettings, updateServerSettings } = useSettingsStore();

  if (!serverSettings) {
    return <div className="p-8">Loading settings...</div>;
  }

  return (
    <AutomationSettingsForm
      serverSettings={serverSettings}
      updateServerSettings={updateServerSettings}
    />
  );
}

function AutomationSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: Settings;
  updateServerSettings: (settings: Settings) => void;
}) {
  const navigate = useNavigate();
  const { automation } = serverSettings;

  const form = useForm({
    defaultValues: {
      scheduleEnabled: automation.scheduleEnabled,
      onCompleteAction: automation.onCompleteAction,
      scriptPath: automation.scriptPath,
    },
    onSubmit: async ({ value }) => {
      const updated = {
        ...serverSettings,
        automation: {
          ...serverSettings.automation,
          scheduleEnabled: value.scheduleEnabled,
          onCompleteAction: value.onCompleteAction,
          scriptPath: value.scriptPath,
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
          <h2 className="text-2xl font-bold tracking-tight">Automation</h2>
          <p className="text-xs text-muted">
            Scheduling, scripting & organization
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
            {/* Scheduling */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Scheduling</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <FormSwitch
                    form={form}
                    name="scheduleEnabled"
                    label="Enable Scheduler"
                    description="Restrict downloads to specific times of day"
                  />
                  
                  <div className="p-4 bg-default/10 rounded-xl border border-dashed border-border text-center">
                    <p className="text-sm text-muted">
                        Detailed schedule rules interface coming soon
                    </p>
                  </div>
                </Card.Content>
              </Card>
            </section>

            {/* Post-Processing */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Post-Processing</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                  <FormSelect
                    form={form}
                    name="onCompleteAction"
                    label="Action on Completion"
                    items={[
                      { value: "none", label: "None" },
                      { value: "run_script", label: "Run Script" },
                      { value: "shutdown", label: "Shutdown System" },
                      { value: "sleep", label: "Sleep/Suspend" },
                    ]}
                  />

                  <form.Subscribe
                    selector={(state) => [state.values.onCompleteAction]}
                  >
                    {([action]) => (
                        action === "run_script" && (
                            <div className="animate-in slide-in-from-top-2 duration-200">
                                <FormTextField
                                    form={form}
                                    name="scriptPath"
                                    label="Script Path"
                                    placeholder="/path/to/script.sh"
                                    description="Absolute path to executable script"
                                />
                            </div>
                        )
                    )}
                  </form.Subscribe>
                </Card.Content>
              </Card>
            </section>

            {/* Auto-Organization */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <div className="w-1.5 h-6 bg-accent rounded-full" />
                <h3 className="text-lg font-bold">Auto-Organization</h3>
              </div>
              <Card className="bg-background/50 border-border overflow-hidden">
                <Card.Content className="p-6 space-y-6">
                    <div className="p-4 bg-default/10 rounded-xl border border-dashed border-border text-center">
                        <p className="text-sm text-muted">
                            Category management interface coming soon
                        </p>
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
