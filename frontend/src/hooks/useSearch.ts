import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { api } from "../lib/api";
import { toast } from "sonner";
import { useSettingsStore } from "../store/useSettingsStore";
import { useRemotes } from "./useRemotes";
import type { SearchConfig } from "../lib/types";

export function useSearch() {
  const queryClient = useQueryClient();
  const { serverSettings, updateServerSettings } = useSettingsStore();
  const { data: remotes, isLoading: isRemotesLoading } = useRemotes();

  const configs = useMemo(() => {
    if (!serverSettings || !remotes) return [];

    const dbConfigs = serverSettings.search?.configs || [];
    const configMap = new Map(dbConfigs.map((c: any) => [c.remote, c]));

    return remotes.map((remote: any) => {
      if (configMap.has(remote.name)) {
        return configMap.get(remote.name);
      }
      // Default config for discovered remote
      return {
        remote: remote.name,
        autoIndexIntervalMin: 0,
        status: "idle" as const,
      };
    });
  }, [serverSettings, remotes]);

  const triggerIndex = useMutation({
    mutationFn: (remote: string) => api.triggerIndex(remote),
    onSuccess: () => {
      // Invalidate settings to refresh status if backend updates it quickly
      queryClient.invalidateQueries({ queryKey: ["settings"] }); 
      toast.success("Indexing started");
    },
    onError: (err: any) => {
      toast.error("Failed to start indexing: " + err.message);
    }
  });

  const updateConfig = useMutation({
    mutationFn: async (vars: { 
      remote: string; 
      interval: number;
      excludedPatterns?: string;
      includedExtensions?: string;
      minSizeBytes?: number;
    }) => {
      if (!serverSettings) return;

      const currentConfigs: SearchConfig[] = [...(serverSettings.search?.configs || [])];
      const index = currentConfigs.findIndex((c: SearchConfig) => c.remote === vars.remote);

      const newConfig = {
        remote: vars.remote,
        autoIndexIntervalMin: vars.interval,
        excludedPatterns: vars.excludedPatterns,
        includedExtensions: vars.includedExtensions,
        minSizeBytes: vars.minSizeBytes,
        status: "idle" as const, // Reset status on config change? Or keep existing? Keep existing if possible, but here we construct new.
        // Actually, we should preserve other fields if updating existing.
      };

      if (index >= 0) {
        currentConfigs[index] = { ...currentConfigs[index], ...newConfig };
      } else {
        currentConfigs.push(newConfig);
      }

      const newSettings = {
        ...serverSettings,
        search: {
          ...serverSettings.search,
          configs: currentConfigs,
        }
      };

      await api.updateSettings(newSettings);
      updateServerSettings(newSettings);
    },
    onSuccess: () => {
      toast.success("Settings updated");
    },
    onError: (err: any) => {
      toast.error("Failed to update settings: " + err.message);
    }
  });

  const updateConfigs = useMutation({
    mutationFn: async (batch: Record<string, { interval: number; excludedPatterns?: string; includedExtensions?: string; minSizeBytes?: number }>) => {
      if (!serverSettings) return;

      const currentConfigs: SearchConfig[] = [...(serverSettings.search?.configs || [])];
      
      Object.entries(batch).forEach(([remote, cfg]) => {
        const index = currentConfigs.findIndex((c: SearchConfig) => c.remote === remote);
        const newConfig = {
            remote,
            autoIndexIntervalMin: cfg.interval,
            excludedPatterns: cfg.excludedPatterns,
            includedExtensions: cfg.includedExtensions,
            minSizeBytes: cfg.minSizeBytes,
        };

        if (index >= 0) {
            currentConfigs[index] = { ...currentConfigs[index], ...newConfig };
        } else {
            currentConfigs.push({ ...newConfig, status: 'idle' as const });
        }
      });

      const newSettings = {
        ...serverSettings,
        search: {
          ...serverSettings.search,
          configs: currentConfigs,
        }
      };

      await api.updateSettings(newSettings);
      updateServerSettings(newSettings);
    },
    onSuccess: () => {
      toast.success("All settings updated");
    },
    onError: (err: any) => {
      toast.error("Failed to update settings: " + err.message);
    }
  });

  const search = useMutation({
    mutationFn: (query: string) => api.search(query),
  });

  return {
    configs,
    isLoading: isRemotesLoading || !serverSettings,
    triggerIndex,
    updateConfig,
    updateConfigs,
    search,
  };
}
