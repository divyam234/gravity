import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";
import { toast } from "sonner";

export function useSearch() {
  const queryClient = useQueryClient();

  const { data: configs, isLoading } = useQuery({
    queryKey: ["search", "config"],
    queryFn: () => api.getSearchConfigs(),
  });

  const triggerIndex = useMutation({
    mutationFn: (remote: string) => api.triggerIndex(remote),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["search", "config"] });
      toast.success("Indexing started");
    },
    onError: (err: any) => {
      toast.error("Failed to start indexing: " + err.message);
    }
  });

  const updateConfig = useMutation({
    mutationFn: (vars: { 
      remote: string; 
      interval: number;
      excludedPatterns?: string;
      includedExtensions?: string;
      minSizeBytes?: number;
    }) =>
      api.updateSearchConfig(vars.remote, {
        interval: vars.interval,
        excludedPatterns: vars.excludedPatterns,
        includedExtensions: vars.includedExtensions,
        minSizeBytes: vars.minSizeBytes
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["search", "config"] });
      toast.success("Settings updated");
    },
    onError: (err: any) => {
      toast.error("Failed to update settings: " + err.message);
    }
  });

  const updateConfigs = useMutation({
    mutationFn: (configs: Record<string, { interval: number; excludedPatterns?: string; includedExtensions?: string; minSizeBytes?: number }>) =>
      api.updateSearchConfigs(configs),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["search", "config"] });
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
    configs: configs?.data || [],
    isLoading,
    triggerIndex,
    updateConfig,
    updateConfigs,
    search,
  };
}
