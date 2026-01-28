import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { openapi } from "../lib/openapi";

export function useDownloads(params?: {
  status?: string[];
  limit?: number;
  offset?: number;
}) {
  // We use SSE for real-time updates now, so standard polling is largely unnecessary
  // We can keep a slow heartbeat (e.g., 30s) just for eventual consistency
  const pollingInterval = 30000;

  return useQuery(
    openapi.queryOptions(
      "get",
      "/downloads",
      {
        params: {
          query: {
            status: params?.status?.join(","),
            limit: params?.limit,
            offset: params?.offset,
          },
        },
      },
      {
        refetchInterval: (query) =>
          query.state.status === "error" ? false : pollingInterval,
      },
    ),
  );
}

export function useDownloadActions() {
  const queryClient = useQueryClient();

  const invalidate = () => {
    queryClient.invalidateQueries({
      queryKey: openapi.queryOptions("get", "/downloads").queryKey,
    });
    queryClient.invalidateQueries({
      queryKey: openapi.queryOptions("get", "/stats").queryKey,
    });
  };

  const create = openapi.useMutation("post", "/downloads", {
    onSuccess: () => {
      invalidate();
      toast.success("Download started");
    },
    onError: (err) => toast.error(`Failed to start download: ${err.error}`),
  });

  const downloadMagnet = openapi.useMutation("post", "/magnets/download", {
    onSuccess: () => {
      invalidate();
      toast.success("Download started");
    },
    onError: (err) => toast.error(`Failed to start download: ${err.error}`),
  });

  const pause = openapi.useMutation("post", "/downloads/{id}/pause", {
    onSuccess: () => {
      invalidate();
      toast.info("Download paused");
    },
    onError: (err) => toast.error(`Failed to pause download: ${err.error}`),
  });

  const resume = openapi.useMutation("post", "/downloads/{id}/resume", {
    onSuccess: () => {
      invalidate();
      toast.success("Download resumed");
    },
    onError: (err) => toast.error(`Failed to resume download: ${err.error}`),
  });

  const remove = openapi.useMutation("delete", "/downloads/{id}", {
    onSuccess: () => {
      invalidate();
      toast.info("Download removed");
    },
    onError: (err) => toast.error(`Failed to remove download: ${err.error}`),
  });

  return { create, downloadMagnet, pause, resume, remove };
}
