import { create } from 'zustand';

export interface WSEvent {
  type: string;
  timestamp: string;
  data: any;
}

interface WebSocketState {
  connected: boolean;
  lastEvent: WSEvent | null;
  actions: {
    setConnected: (connected: boolean) => void;
    setLastEvent: (event: WSEvent | null) => void;
  };
}

export const useWebSocketStore = create<WebSocketState>((set) => ({
  connected: false,
  lastEvent: null,
  actions: {
    setConnected: (connected) => set({ connected }),
    setLastEvent: (lastEvent) => set({ lastEvent }),
  },
}));

export const useWebSocket = () => {
  const connected = useWebSocketStore((state) => state.connected);
  const lastEvent = useWebSocketStore((state) => state.lastEvent);
  return { connected, lastEvent };
};
