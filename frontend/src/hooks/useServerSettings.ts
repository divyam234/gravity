import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "../lib/api";

export function useServerSettings() {
	return useQuery({
		queryKey: ["gravity", "server-settings"],
		queryFn: () => api.getSettings(),
	});
}

export function useUpdateServerSettings() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (settings: Record<string, string>) => api.updateSettings(settings),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["gravity", "server-settings"] });
			toast.success("Settings saved");
		},
		onError: (err: Error) => toast.error(`Failed to save settings: ${err.message}`),
	});
}
