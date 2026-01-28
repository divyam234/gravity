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

const automationSettingsSchema = z.object({
  scheduleEnabled: z.boolean(),
  onCompleteAction: z.enum(["none", "run_script", "shutdown", "sleep"], {
    error: "Select a valid completion action",
  }),
  scriptPath: z.string().refine((val) => val.length > 0 || true, {
    error: "Script path is required when Run Script is selected",
  }),
});

function AutomationSettingsForm({
  serverSettings,
  updateServerSettings,
}: {
  serverSettings: Settings;
  updateServerSettings: (settings: Settings) => void;
}) {
  const navigate = useNavigate();
  const automation = serverSettings.automation;
  const { changeGlobalOption } = useEngineActions();

  const form = useForm({
    defaultValues: {
      scheduleEnabled: !!automation?.scheduleEnabled,
      onCompleteAction: (automation?.onCompleteAction as "none" | "run_script" | "shutdown" | "sleep") || "none",
      scriptPath: automation?.scriptPath || "",
    },
    validators: {
      onChange: automationSettingsSchema as any,
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
        automation: {
          ...serverSettings.automation,
          scheduleEnabled: value.scheduleEnabled,
          onCompleteAction: value.onCompleteAction,
          scriptPath: value.scriptPath,
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
          <h2 className="text-2xl font-bold tracking-tight">Automation</h2>
          <p className="text-xs text-muted">
            Scheduling, scripting & organization
          </p>
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
            <form.Subscribe
                selector={(state) => state.values.onCompleteAction}
            >
                {(onCompleteAction) => {
                    const settingGroups: SettingGroupConfig<z.infer<typeof automationSettingsSchema>>[] = [
                        {
                            id: "scheduling",
                            title: "Scheduling",
                            fields: [
                                {
                                    name: "scheduleEnabled",
                                    type: "switch",
                                    label: "Enable Scheduler",
                                    description: "Restrict downloads to specific times of day",
                                    colSpan: 2,
                                },
                            ],
                        },
                        {
                            id: "post-processing",
                            title: "Post-Processing",
                            fields: [
                                {
                                    name: "onCompleteAction",
                                    type: "select",
                                    label: "Action on Completion",
                                    options: [
                                        { value: "none", label: "None" },
                                        { value: "run_script", label: "Run Script" },
                                        { value: "shutdown", label: "Shutdown System" },
                                        { value: "sleep", label: "Sleep/Suspend" },
                                    ],
                                    description: "What to do after a download finishes",
                                    colSpan: 2,
                                },
                                ...(onCompleteAction === "run_script" ? [{
                                    name: "scriptPath" as const,
                                    type: "text" as const,
                                    label: "Script Path",
                                    placeholder: "/path/to/script.sh",
                                    description: "Absolute path to executable script",
                                    colSpan: 2,
                                }] : []),
                            ],
                        },
                    ];
                    return (
                        <>
                            <DynamicSettings form={form} groups={settingGroups} />
                            
                            {/* Manual sections for things not yet supported by DynamicSettings (like placeholder boxes) */}
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
                        </>
                    );
                }}
            </form.Subscribe>
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
