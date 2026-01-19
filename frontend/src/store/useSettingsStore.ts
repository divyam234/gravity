import { create } from "zustand";
import { persist } from "zustand/middleware";
import { api } from "../lib/api";

export interface ServerConfig {
	id: string;
	name: string;
	serverUrl: string;
	apiKey: string;
}

interface SettingsState {
	pollingInterval: number;
	theme: "light" | "dark" | "system";
	viewMode: "list" | "grid";
	searchQuery: string;
	enableNotifications: boolean;
	isSelectionMode: boolean;
	selectedGids: Set<string>;
	defaultRemote: string;

    // Multi-server state
    servers: ServerConfig[];
    activeServerId: string | null;

	setSearchQuery: (query: string) => void;
	setEnableNotifications: (enable: boolean) => void;
	setDefaultRemote: (remote: string) => void;
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
			pollingInterval: 1000,
			theme: "dark",
			viewMode: "list",
			searchQuery: "",
			enableNotifications: false,
			isSelectionMode: false,
			selectedGids: new Set(),
			defaultRemote: "",
            
            servers: [],
            activeServerId: null,

			setSearchQuery: (searchQuery) => set({ searchQuery }),
			setEnableNotifications: (enableNotifications) =>
				set({ enableNotifications }),
			setDefaultRemote: (defaultRemote) => set({ defaultRemote }),
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
                    // If first server, activate it
                    if (servers.length === 1) {
                        return { servers, activeServerId: id };
                    }
                    return { servers };
                });
            },

            updateServer: (id, updates) => {
                set((state) => ({
                    servers: state.servers.map((s) => s.id === id ? { ...s, ...updates } : s)
                }));
            },

            removeServer: (id) => {
                set((state) => {
                    const servers = state.servers.filter((s) => s.id !== id);
                    let activeServerId = state.activeServerId;
                    if (id === activeServerId) {
                        activeServerId = servers.length > 0 ? servers[0].id : null;
                    }
                    return { servers, activeServerId };
                });
            },

            setActiveServer: (id) => set({ activeServerId: id }),

			setPollingInterval: (pollingInterval) => set({ pollingInterval }),
			setTheme: (theme) => set({ theme }),
		}),
		{
			name: "gravity-settings",
			partialize: (state) => ({
				theme: state.theme,
				viewMode: state.viewMode,
				enableNotifications: state.enableNotifications,
				defaultRemote: state.defaultRemote,
				pollingInterval: state.pollingInterval,
                servers: state.servers,
                activeServerId: state.activeServerId,
			}),
            onRehydrateStorage: () => (state) => {
                if (!state) return;
                
                // Ensure default server if empty
                if (state.servers.length === 0) {
                    const defaultId = "default-local";
                    state.servers = [{
                        id: defaultId,
                        name: "Local Gravity",
                        serverUrl: "/api/v1",
                        apiKey: ""
                    }];
                    state.activeServerId = defaultId;
                }

                // Sync API client with rehydrated state
                const active = state.servers.find(s => s.id === state.activeServerId);
                if (active) {
                    api.setBaseUrl(active.serverUrl);
                    api.setApiKey(active.apiKey);
                }
            }
		},
	),
);

// Sync API client
const syncApi = (state: SettingsState) => {
    const active = state.servers.find(s => s.id === state.activeServerId);
    if (active) {
        api.setBaseUrl(active.serverUrl);
        api.setApiKey(active.apiKey);
    }
};

useSettingsStore.subscribe(syncApi);

// Initial sync
// We need to wait for rehydration? persist middleware handles it.
// But we can try to sync initially if state exists.
const initialState = useSettingsStore.getState();
syncApi(initialState);