import { create } from "zustand";
import { persist } from "zustand/middleware";
import { aria2 } from "../lib/aria2-rpc";

export interface ServerConfig {
	id: string;
	name: string;
	rpcUrl: string;
	rpcSecret: string;
}

interface SettingsState {
	// Legacy/Active properties for compatibility
	rpcUrl: string;
	rpcSecret: string;

	// Multi-server state
	servers: ServerConfig[];
	activeServerId: string | null;

	pollingInterval: number;
	theme: "light" | "dark" | "system";

	// Actions
	setRpcUrl: (url: string) => void;
	setRpcSecret: (secret: string) => void;

	addServer: (server: Omit<ServerConfig, "id">) => void;
	updateServer: (id: string, updates: Partial<ServerConfig>) => void;
	removeServer: (id: string) => void;
	setActiveServer: (id: string) => void;

	setPollingInterval: (ms: number) => void;
	setTheme: (theme: "light" | "dark" | "system") => void;
}

export const useSettingsStore = create<SettingsState>()(
	persist(
		(set) => ({
			rpcUrl: "",
			rpcSecret: "",
			servers: [],
			activeServerId: null,
			pollingInterval: 1000,
			theme: "dark",

			setRpcUrl: (rpcUrl) => {
				set((state) => {
					// If we have an active server, update it too
					const servers = state.servers.map((s) =>
						s.id === state.activeServerId ? { ...s, rpcUrl } : s,
					);
					return { rpcUrl, servers };
				});
			},

			setRpcSecret: (rpcSecret) => {
				set((state) => {
					const servers = state.servers.map((s) =>
						s.id === state.activeServerId ? { ...s, rpcSecret } : s,
					);
					return { rpcSecret, servers };
				});
			},

			addServer: (serverData) => {
				const id = crypto.randomUUID();
				const newServer = { ...serverData, id };
				set((state) => {
					const servers = [...state.servers, newServer];
					// If this is the first server, make it active
					if (servers.length === 1) {
						return {
							servers,
							activeServerId: id,
							rpcUrl: newServer.rpcUrl,
							rpcSecret: newServer.rpcSecret,
						};
					}
					return { servers };
				});
			},

			updateServer: (id, updates) => {
				set((state) => {
					const servers = state.servers.map((s) =>
						s.id === id ? { ...s, ...updates } : s,
					);

					// If updating active server, sync global state
					if (id === state.activeServerId) {
						const activeServer = servers.find((s) => s.id === id);
						if (activeServer) {
							return {
								servers,
								rpcUrl: activeServer.rpcUrl,
								rpcSecret: activeServer.rpcSecret,
							};
						}
					}
					return { servers };
				});
			},

			removeServer: (id) => {
				set((state) => {
					const servers = state.servers.filter((s) => s.id !== id);
					let activeServerId = state.activeServerId;
					let rpcUrl = state.rpcUrl;
					let rpcSecret = state.rpcSecret;

					if (id === activeServerId) {
						// If we removed the active server, pick the first one available or clear
						if (servers.length > 0) {
							activeServerId = servers[0].id;
							rpcUrl = servers[0].rpcUrl;
							rpcSecret = servers[0].rpcSecret;
						} else {
							activeServerId = null;
							rpcUrl = "";
							rpcSecret = "";
						}
					}

					return { servers, activeServerId, rpcUrl, rpcSecret };
				});
			},

			setActiveServer: (id) => {
				set((state) => {
					const server = state.servers.find((s) => s.id === id);
					if (server) {
						return {
							activeServerId: id,
							rpcUrl: server.rpcUrl,
							rpcSecret: server.rpcSecret,
						};
					}
					return {};
				});
			},

			setPollingInterval: (pollingInterval) => set({ pollingInterval }),
			setTheme: (theme) => set({ theme }),
		}),
		{
			name: "aria2-settings",
			onRehydrateStorage: () => (state) => {
				if (!state) return;

				// Migration: If rpcUrl exists but no servers, create default
				if (state.rpcUrl && (!state.servers || state.servers.length === 0)) {
					const defaultId = "default-1";
					const defaultServer: ServerConfig = {
						id: defaultId,
						name: "Default Server",
						rpcUrl: state.rpcUrl,
						rpcSecret: state.rpcSecret || "",
					};
					state.servers = [defaultServer];
					state.activeServerId = defaultId;
				}
			},
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
