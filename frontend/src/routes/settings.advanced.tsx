import {
  Button,
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

const advancedSettingsSchema = z.object({
  logLevel: z.enum(["debug", "info", "warn", "error"], {
    error: "Select a valid log level",
  }),
  saveInterval: z.number()
    .min(1, "Interval must be at least 1 second")
    .max(3600, "Interval cannot exceed 1 hour"),
  debugMode: z.boolean(),
});

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
      logLevel: (advanced?.logLevel as "debug" | "info" | "warn" | "error") || "info",
      debugMode: !!advanced?.debugMode,
      saveInterval: Number(advanced?.saveInterval || 60),
    },
    validators: {
      onChange: advancedSettingsSchema as any,
    },
    onSubmit: async ({ value }) => {
      const updated: Settings = {
        ...serverSettings,
        advanced: {
          ...serverSettings.advanced,
          logLevel: value.logLevel,
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

  const settingGroups: SettingGroupConfig<z.infer<typeof advancedSettingsSchema>>[] = [
    {
      id: "internals",
      title: "System Internals",
      fields: [
        {
          name: "logLevel",
          type: "select",
          label: "Log Level",
          options: [
            { value: "debug", label: "Debug (Verbose)" },
            { value: "info", label: "Info (Standard)" },
            { value: "warn", label: "Warning" },
            { value: "error", label: "Error" },
          ],
          description: "Verbosity of server logs",
        },
        {
          name: "saveInterval",
          type: "number",
          label: "Save Interval (Seconds)",
          description: "How often to save session state to disk",
        },
        { type: "divider" },
        {
          name: "debugMode",
          type: "switch",
          label: "Enable Debug Mode",
          description: "Enables additional runtime checks and logging. Recommended only for troubleshooting.",
          colSpan: 2,
        },
      ],
    },
  ];

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
            <DynamicSettings form={form} groups={settingGroups} />
          </div>
        </ScrollShadow>
      </div>
    </div>
  );
}
