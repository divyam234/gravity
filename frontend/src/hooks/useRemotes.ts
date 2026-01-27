import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { openapi } from "../lib/openapi";

type ApiError = { code: number; error: string };

export function useRemotes() {
  return useQuery({
    ...openapi.queryOptions("get", "/remotes"),
    select: (res) => res.data || [],
  });
}

export function useRemoteActions() {
  const queryClient = useQueryClient();

  const createRemote = openapi.useMutation("post", "/remotes", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/remotes").queryKey,
      });
      toast.success("Remote created");
    },
    onError: (err) => toast.error("Failed to create remote: " + err.error),
  });

  const deleteRemote = openapi.useMutation("delete", "/remotes/{name}", {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/remotes").queryKey,
      });
      toast.success("Remote deleted");
    },
    onError: (err) => toast.error("Failed to delete remote: " + err.error),
  });

  return { createRemote, deleteRemote };
}
