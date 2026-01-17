import { create } from "zustand";
import { persist } from "zustand/middleware";

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
			rpcUrl: "http://localhost:6800/jsonrpc",
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
