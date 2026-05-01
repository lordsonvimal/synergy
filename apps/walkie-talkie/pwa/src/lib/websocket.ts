export interface WebSocketCallbacks {
  onOpen: () => void;
  onClose: () => void;
  onMessage: (data: unknown) => void;
}

export interface WebSocketClient {
  send: (message: unknown) => void;
  close: () => void;
}

const MAX_RETRIES = 5;
const BASE_DELAY = 1000;

export function createWebSocket(
  url: string,
  callbacks: WebSocketCallbacks
): WebSocketClient {
  let socket: WebSocket | null = null;
  let retries = 0;
  let closed = false;

  function connect(): void {
    socket = new WebSocket(url);

    socket.onopen = () => {
      retries = 0;
      callbacks.onOpen();
    };

    socket.onclose = () => {
      callbacks.onClose();
      if (!closed) {
        scheduleReconnect();
      }
    };

    socket.onmessage = (event) => {
      const data: unknown = JSON.parse(String(event.data));
      callbacks.onMessage(data);
    };
  }

  function scheduleReconnect(): void {
    if (retries >= MAX_RETRIES) {
      return;
    }
    const delay = BASE_DELAY * Math.pow(2, retries);
    retries += 1;
    setTimeout(connect, delay);
  }

  function send(message: unknown): void {
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(message));
    }
  }

  function close(): void {
    closed = true;
    socket?.close();
  }

  connect();

  return { send, close };
}
