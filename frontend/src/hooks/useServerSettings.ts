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

export function useServerSettingsActions() {
	const queryClient = useQueryClient();

	const resetSettings = useMutation({
		mutationFn: () => api.resetSettings(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["gravity", "server-settings"] });
			toast.success("Settings reset to defaults");
		},
		onError: (err: Error) => toast.error(`Failed to reset settings: ${err.message}`),
	});

	const importSettings = useMutation({
		mutationFn: (settings: Record<string, string>) => api.importSettings(settings),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["gravity", "server-settings"] });
			toast.success("Settings imported successfully");
		},
		onError: (err: Error) => toast.error(`Failed to import settings: ${err.message}`),
	});

    // Helper for export since it's a download
    const handleExport = async () => {
        try {
            const settings = await api.exportSettings();
            const blob = new Blob([JSON.stringify(settings, null, 2)], { type: "application/json" });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = "gravity-settings.json";
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            window.URL.revokeObjectURL(url);
            toast.success("Settings exported");
        } catch (err: any) {
            toast.error(`Failed to export settings: ${err.message}`);
        }
    };

	return { resetSettings, importSettings, handleExport };
}
