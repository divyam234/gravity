import { useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { openapi } from "../lib/openapi";
import { toast } from "sonner";
import { useSettingsStore } from "../store/useSettingsStore";
import { useRemotes } from "./useRemotes";
import type { components } from "../gen/api";

type SearchConfig = components["schemas"]["model.RemoteIndexConfig"];
type ApiError = { code: number; error: string };

export function useSearch() {
  const queryClient = useQueryClient();
  const { serverSettings } = useSettingsStore();
  const { data: remotes = [], isLoading: isRemotesLoading } = useRemotes();

  const configs = useMemo(() => {
    if (!serverSettings) return [];

    const dbConfigs = serverSettings.search?.configs || [];
    const configMap = new Map(dbConfigs.map((c) => [c.remote, c]));

    return remotes.map((remote: components["schemas"]["engine.Remote"]) => {
      if (remote.name && configMap.has(remote.name)) {
        return configMap.get(remote.name) as SearchConfig;
      }
      // Default config for discovered remote
      return {
        remote: remote.name || "unknown",
        autoIndexIntervalMin: 0,
        status: "idle",
      } as SearchConfig;
    });
  }, [serverSettings, remotes]);

  const triggerIndex = openapi.useMutation("post", "/search/index/{remote}", {
    onSuccess: () => {
      // Invalidate settings to refresh status if backend updates it quickly
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/settings").queryKey,
      });
      toast.success("Indexing started");
    },
    onError: (err) => {
      toast.error("Failed to start indexing: " + err.error);
    },
  });

  const updateConfig = openapi.useMutation("post", "/search/config/{remote}", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/settings").queryKey,
      });
      toast.success("Settings updated");
    },
    onError: (err) => {
      toast.error("Failed to update settings: " + err.error);
    },
  });

  const updateConfigs = openapi.useMutation("post", "/search/config", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/settings").queryKey,
      });
      toast.success("All settings updated");
    },
    onError: (err) => {
      toast.error("Failed to update settings: " + err.error);
    },
  });

  const search = openapi.useMutation("get", "/search", {
    onError: (err) => toast.error("Search failed: " + err.error),
  });

  return {
    configs,
    isLoading: isRemotesLoading || !serverSettings,
    triggerIndex,
    updateConfig,
    updateConfigs,
    search: {
      ...search,
      data: search.data?.data || [],
    },
  };
}
