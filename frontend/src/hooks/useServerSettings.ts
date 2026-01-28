import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { openapi, client } from "../lib/openapi";

export function useServerSettingsActions() {
  const queryClient = useQueryClient();

  const resetSettings = openapi.useMutation("post", "/settings/reset", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/settings").queryKey,
      });
      toast.success("Settings reset to defaults");
    },
    onError: (err) => toast.error("Failed to reset settings: " + (err.error || "Unknown error")),
  });

  const importSettings = openapi.useMutation("post", "/settings/import", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/settings").queryKey,
      });
      toast.success("Settings imported successfully");
    },
    onError: (err) => toast.error("Failed to import settings: " + (err.error || "Unknown error")),
  });

  const handleExport = async () => {
    const { data, error } = await client.POST("/settings/export", {});
    if (error) {
      const errorMsg = (error as { error?: string }).error || "Unknown error";
      toast.error("Failed to export settings: " + errorMsg);
      throw new Error(errorMsg);
    }
    return data;
  };

  return { resetSettings, importSettings, handleExport };
}
