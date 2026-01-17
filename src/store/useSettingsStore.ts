import { create } from "zustand";
import { persist } from "zustand/middleware";
import { aria2 } from "../lib/aria2-rpc";

interface SettingsState {
	rpcUrl: string;
	rpcSecret: string;
	pollingInterval: number;
	theme: "light" | "dark" | "system";
	setRpcUrl: (url: string) => void;
	setRpcSecret: (secret: string) => void;
	setPollingInterval: (ms: number) => void;
	setTheme: (theme: "light" | "dark" | "system") => void;
}

export const useSettingsStore = create<SettingsState>()(
	persist(
		(set) => ({
			rpcUrl: "",
			rpcSecret: "",
			pollingInterval: 1000,
			theme: "dark",
			setRpcUrl: (rpcUrl) => set({ rpcUrl }),
			setRpcSecret: (rpcSecret) => set({ rpcSecret }),
			setPollingInterval: (pollingInterval) => set({ pollingInterval }),
			setTheme: (theme) => set({ theme }),
		}),
		{
			name: "aria2-settings",
		},
	),
);

// Synchronously initialize the aria2 client with persisted settings
const initSettings = useSettingsStore.getState();
if (initSettings.rpcUrl) {
	aria2.updateConfig(initSettings.rpcUrl, initSettings.rpcSecret);
}

// Subscribe to store changes to keep the aria2 client in sync
useSettingsStore.subscribe((state) => {
	aria2.updateConfig(state.rpcUrl, state.rpcSecret);
});
