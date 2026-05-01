import {
  Component,
  JSX,
  createContext,
  createSignal,
  useContext
} from "solid-js";
import { createWebSocket, WebSocketClient } from "../lib/websocket.js";
import { useSettings } from "./settings.js";

interface ConnectionContextValue {
  connected: () => boolean;
  connect: () => void;
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
  let messageHandler: ((data: unknown) => void) | null = null;

  const connect = (): void => {
    const { host, port } = settings();
    ws = createWebSocket(`wss://${host}:${port}`, {
      onOpen: () => setConnected(true),
      onClose: () => setConnected(false),
      onMessage: (data) => messageHandler?.(data)
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
    messageHandler = handler;
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
