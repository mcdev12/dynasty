import { useEffect, useRef, useState } from 'react';
import { DraftEvent } from '../types/draft';

export function useDraftWebSocket(
  draftId: string,
  userId: string,
  onEvent: (event: DraftEvent) => void
) {
  const ws = useRef<WebSocket | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const wsUrl = `ws://localhost:8081/ws/draft?draft_id=${draftId}&user_id=${userId}`;
    ws.current = new WebSocket(wsUrl);

    ws.current.onopen = () => {
      console.log('WebSocket connected');
      setConnected(true);
    };

    ws.current.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        console.log('WebSocket received:', data);
        onEvent(data);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
        console.error('Raw message data:', event.data);
      }
    };

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.current.onclose = () => {
      console.log('WebSocket disconnected');
      setConnected(false);
    };

    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [draftId, userId]);

  const sendMessage = (message: any) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
    }
  };

  return { connected, sendMessage };
}