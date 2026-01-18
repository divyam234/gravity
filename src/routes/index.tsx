import { Button, Kbd, Tooltip } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import IconPlus from "~icons/gravity-ui/plus";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { StatsOverview } from "../components/dashboard/StatsOverview";
import { globalStatOptions, useAria2Actions } from "../hooks/useAria2";
import { useNotifications } from "../hooks/useNotifications";
import { useSettingsStore } from "../store/useSettingsStore";

export const Route = createFileRoute("/")({
  component: Dashboard,
  loader: async ({ context: { queryClient } }) => {
    const { rpcUrl, pollingInterval } = useSettingsStore.getState();

    if (!rpcUrl) return;

    // Prefetch essential data for the dashboard
    await queryClient.ensureQueryData(
      globalStatOptions(rpcUrl, pollingInterval),
    );
  },
});

function Dashboard() {
  useNotifications();

  return (
    <div className="space-y-6">
      {/* Toolbar */}
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
      </div>

      <StatsOverview />

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* You could add more overview cards here later */}
      </div>
    </div>
  );
}
