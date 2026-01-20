import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "../lib/api";

export function useRemoteStats(enabled = true) {
	return useQuery({
		queryKey: ["gravity", "remote", "stats"],
		queryFn: () => Promise.resolve({}), // Not exposed via API yet
		enabled,
	});
}

export function useRemoteVersion() {
	return useQuery({
		queryKey: ["gravity", "remote", "version"],
		queryFn: () => Promise.resolve("v1.0"),
	});
}

export function useRemotes() {
	return useQuery({
		queryKey: ["gravity", "remotes"],
		queryFn: () => api.getRemotes().then(res => res.data),
	});
}

export function useRemoteActions() {
	const queryClient = useQueryClient();

	const createRemote = useMutation({
		mutationFn: (params: { name: string; type: string; parameters: Record<string, string> }) =>
            api.request('POST', '/remotes', {
                name: params.name,
                type: params.type,
                config: params.parameters
            }),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["gravity", "remotes"] });
			toast.success("Remote created successfully");
		},
		onError: (err: Error) => toast.error(`Failed to create remote: ${err.message}`),
	});

	const deleteRemote = useMutation({
		mutationFn: (name: string) =>
            api.request('DELETE', `/remotes/${name}`),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["gravity", "remotes"] });
			toast.success("Remote deleted");
		},
		onError: (err: Error) => toast.error(`Failed to delete remote: ${err.message}`),
	});

    const testRemote = useMutation({
        mutationFn: (name: string) => api.request('POST', `/remotes/${name}/test`),
        onSuccess: () => toast.success("Connection successful"),
        onError: (err: Error) => toast.error(`Connection failed: ${err.message}`),
    });

	return { createRemote, deleteRemote, testRemote };
}