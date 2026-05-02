import {
  Component,
  JSX,
  createContext,
  createSignal,
  onCleanup,
  useContext
} from "solid-js";
import { createWebSocket, WebSocketClient } from "../lib/websocket.js";
import { useSettings } from "./settings.js";

interface ConnectionContextValue {
  connected: () => boolean;
  connect: (initialTabId: string) => void;
  disconnect: () => void;
  send: (message: unknown) => void;
  onMessage: (handler: (data: unknown) => void) => void;
}

const ConnectionContext = createContext<ConnectionContextValue>();

export const ConnectionProvider: Component<{ children: JSX.Element }> = (
  props
) => {
  const { settings } = useSettings();
  const [connected, setConnected] = createSignal(false);
  let ws: WebSocketClient | null = null;
  let pendingInitialTabId: string | null = null;
  const messageHandlers: Array<(data: unknown) => void> = [];

  const connect = (initialTabId: string): void => {
    const { host, port, secret } = settings();
    const params = secret ? `?token=${encodeURIComponent(secret)}` : "";
    pendingInitialTabId = initialTabId;
    ws = createWebSocket(`wss://${host}:${port}${params}`, {
      onOpen: () => {
        setConnected(true);
        if (pendingInitialTabId) {
          ws?.send({ type: "create-tab", tabId: pendingInitialTabId });
          pendingInitialTabId = null;
        }
      },
      onClose: () => setConnected(false),
      onMessage: (data) => {
        for (const handler of messageHandlers) {
          handler(data);
        }
      }
    });
  };

  const disconnect = (): void => {
    ws?.close();
    ws = null;
    setConnected(false);
  };

  const send = (message: unknown): void => {
    ws?.send(message);
  };

  const onMessage = (handler: (data: unknown) => void): void => {
    messageHandlers.push(handler);
    onCleanup(() => {
      const idx = messageHandlers.indexOf(handler);
      if (idx !== -1) {
        messageHandlers.splice(idx, 1);
      }
    });
  };

  return (
    <ConnectionContext.Provider
      value={{ connected, connect, disconnect, send, onMessage }}
    >
      {props.children}
    </ConnectionContext.Provider>
  );
};

export function useConnection(): ConnectionContextValue {
  const ctx = useContext(ConnectionContext);
  if (!ctx) {
    throw new Error("useConnection must be used within ConnectionProvider");
  }
  return ctx;
}
