import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { openapi } from "../lib/openapi";

export function useProviders() {
  return useQuery(openapi.queryOptions("get", "/providers"));
}

export function useProviderActions() {
  const queryClient = useQueryClient();

  const configureProvider = openapi.useMutation("put", "/providers/{name}", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/providers").queryKey,
      });
      toast.success("Provider configured successfully");
    },
    onError: (err) => toast.error("Failed to configure provider: " + err.error),
  });

  return { configureProvider };
}
