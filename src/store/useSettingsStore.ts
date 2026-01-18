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
	viewMode: "list" | "grid";
	searchQuery: string;
	isSelectionMode: boolean;
	selectedGids: Set<string>;

	// Actions
	setRpcUrl: (url: string) => void;
	setRpcSecret: (secret: string) => void;

	setSearchQuery: (query: string) => void;
	setViewMode: (mode: "list" | "grid") => void;
	setIsSelectionMode: (isSelectionMode: boolean) => void;
	setSelectedGids: (gids: Set<string>) => void;
	toggleGidSelection: (gid: string) => void;

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
			viewMode: "list",
			searchQuery: "",
			isSelectionMode: false,
			selectedGids: new Set(),

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

			setSearchQuery: (searchQuery) => set({ searchQuery }),
			setViewMode: (viewMode) => set({ viewMode }),
			setIsSelectionMode: (isSelectionMode) =>
				set({ isSelectionMode, selectedGids: new Set() }),
			setSelectedGids: (selectedGids) => set({ selectedGids }),
			toggleGidSelection: (gid) => {
				set((state) => {
					const newSelected = new Set(state.selectedGids);
					if (newSelected.has(gid)) newSelected.delete(gid);
					else newSelected.add(gid);
					return { selectedGids: newSelected };
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
			partialize: (state) => ({
				theme: state.theme,
				viewMode: state.viewMode,
				servers: state.servers,
				activeServerId: state.activeServerId,
				pollingInterval: state.pollingInterval,
				// Include these for migration and instant rehydration
				rpcUrl: state.rpcUrl,
				rpcSecret: state.rpcSecret,
			}),
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

				// Ensure rpcUrl/rpcSecret are synced with the active server if they are empty
				if (state.activeServerId && state.servers.length > 0 && !state.rpcUrl) {
					const active = state.servers.find(
						(s) => s.id === state.activeServerId,
					);
					if (active) {
						state.rpcUrl = active.rpcUrl;
						state.rpcSecret = active.rpcSecret;
					}
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
