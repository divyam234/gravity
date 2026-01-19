import { useEffect, useRef } from 'react';
import { useSettingsStore } from '../store/useSettingsStore';
import { useWebSocketStore } from '../store/useWebSocketStore';
import type { WSEvent } from '../store/useWebSocketStore';

export type { WSEvent };

// This hook should be called ONLY ONCE at the top level of the app (e.g., in Layout)
export function useWebSocketManager() {
  const ws = useRef<WebSocket | null>(null);
  const { servers, activeServerId } = useSettingsStore();
  const { setConnected, setLastEvent } = useWebSocketStore((state) => state.actions);

  const activeServer = servers.find(s => s.id === activeServerId);
  const serverUrl = activeServer?.serverUrl || "/api/v1";
  const token = activeServer?.apiKey || "";

  useEffect(() => {
    let socketUrl = "";

    if (serverUrl.startsWith("http")) {
      const url = new URL(serverUrl);
      const protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      socketUrl = `${protocol}//${url.host}/ws`;
    } else {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host;
      socketUrl = `${protocol}//${host}/ws`;
    }

    if (token) {
      socketUrl += `?token=${token}`;
    }

    let reconnectTimer: any;

    const connect = () => {
      if (ws.current) {
        ws.current.close();
      }

      console.log(`WS: Connecting to ${socketUrl}`);
      ws.current = new WebSocket(socketUrl);

      ws.current.onopen = () => {
        setConnected(true);
        console.log('WS: Connected');
      };

      ws.current.onmessage = (event) => {
        try {
          const ev: WSEvent = JSON.parse(event.data);
          setLastEvent(ev);
        } catch (err) {
          console.error('WS: Failed to parse message', err);
        }
      };

      ws.current.onclose = (event) => {
        setConnected(false);
        if (event.code !== 1000) {
            console.log('WS: Disconnected, retrying in 5s...');
            reconnectTimer = setTimeout(connect, 5000);
        }
      };

      ws.current.onerror = (err) => {
        console.error('WS: Error', err);
        ws.current?.close();
      };
    };

    connect();

    return () => {
      clearTimeout(reconnectTimer);
      if (ws.current) {
        ws.current.onclose = () => {}; 
        ws.current.close(1000);
      }
    };
  }, [serverUrl, token, setConnected, setLastEvent]);
}

// Re-export the state hook from the store for convenience
export { useWebSocket } from '../store/useWebSocketStore';
