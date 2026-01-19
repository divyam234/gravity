import { useEffect, useRef, useState } from 'react';

export interface WSEvent {
  type: string;
  timestamp: string;
  data: any;
}

export function useWebSocket(token?: string) {
  const [connected, setConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<WSEvent | null>(null);
  const ws = useRef<WebSocket | null>(null);

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const url = `${protocol}//${host}/ws${token ? `?token=${token}` : ''}`;

    const connect = () => {
      ws.current = new WebSocket(url);

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

      ws.current.onclose = () => {
        setConnected(false);
        console.log('WS: Disconnected, retrying in 5s...');
        setTimeout(connect, 5000);
      };

      ws.current.onerror = (err) => {
        console.error('WS: Error', err);
        ws.current?.close();
      };
    };

    connect();

    return () => {
      ws.current?.close();
    };
  }, [token]);

  return { connected, lastEvent };
}
