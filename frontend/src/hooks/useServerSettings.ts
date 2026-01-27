import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { openapi, client } from "../lib/openapi";

type ApiError = { code: number; error: string };

export function useServerSettingsActions() {
  const queryClient = useQueryClient();

  const resetSettings = openapi.useMutation("post", "/settings/reset", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/settings").queryKey,
      });
      toast.success("Settings reset to defaults");
    },
    onError: (err) => toast.error("Failed to reset settings: " + err.error),
  });

  const importSettings = openapi.useMutation("post", "/settings/import", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/settings").queryKey,
      });
      toast.success("Settings imported successfully");
    },
    onError: (err) => toast.error("Failed to import settings: " + err.error),
  });

  const handleExport = async () => {
    const { data, error } = await client.POST("/settings/export", {});
    if (error) {
      toast.error("Failed to export settings: " + error.error);
      throw new Error(error.error);
    }
    return data;
  };

  return { resetSettings, importSettings, handleExport };
}
