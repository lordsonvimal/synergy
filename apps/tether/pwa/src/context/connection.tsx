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
  connect: (getTabIds: () => string[]) => void;
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
  const messageHandlers: Array<(data: unknown) => void> = [];

  const connect = (getTabIds: () => string[]): void => {
    const { host, port, secret } = settings();
    const params = secret ? `?token=${encodeURIComponent(secret)}` : "";
    ws = createWebSocket(`wss://${host}:${port}${params}`, {
      onOpen: () => {
        setConnected(true);
        for (const tabId of getTabIds()) {
          ws?.send({ type: "create-tab", tabId });
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
