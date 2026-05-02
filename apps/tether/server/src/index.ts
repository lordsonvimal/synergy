import { readFileSync } from "fs";
import { resolve } from "path";
import { URL } from "url";
import https from "https";
import express from "express";
import { WebSocketServer, WebSocket } from "ws";
import { TerminalManager } from "./terminal.js";
import { handleMessage } from "./handler.js";
import { getBattery } from "./battery.js";
import { validateToken } from "./auth.js";

const PORT = Number(process.env.PORT) || 5100;
const CERT_DIR = resolve(import.meta.dirname, "../../certs");
const GRACE_PERIOD_MS = Number(process.env.GRACE_PERIOD_MS) || 300_000;
const MIRROR_CLIENT_ID = "mirror";

const app = express();

app.use(express.static(resolve(import.meta.dirname, "../../dist/pwa")));

const server = https.createServer(
  {
    cert: readFileSync(resolve(CERT_DIR, "cert.pem")),
    key: readFileSync(resolve(CERT_DIR, "key.pem"))
  },
  app
);

const wss = new WebSocketServer({
  server,
  verifyClient: (info, cb) => {
    const reqUrl = new URL(info.req.url ?? "/", `https://${info.req.headers.host}`);
    const token = reqUrl.searchParams.get("token") ?? undefined;
    if (validateToken(token)) {
      cb(true);
    } else {
      console.log("[ws] auth rejected — invalid token");
      cb(false, 401, "Unauthorized");
    }
  }
});

interface ClientInfo {
  ws: WebSocket;
  clientId: string;
  batteryInterval: ReturnType<typeof setInterval>;
}

const clients = new Map<WebSocket, ClientInfo>();
const managers = new Map<string, TerminalManager>();

function getOrCreateManager(clientId: string): TerminalManager {
  const existing = managers.get(clientId);
  if (existing) return existing;

  const manager = new TerminalManager(clientId, GRACE_PERIOD_MS);
  manager.setMessageCallback((data) => {
    const payload = JSON.stringify(data);
    for (const info of clients.values()) {
      if (info.clientId === clientId && info.ws.readyState === WebSocket.OPEN) {
        info.ws.send(payload);
      }
    }
  });
  managers.set(clientId, manager);
  return manager;
}

function clientCountForId(clientId: string): number {
  let count = 0;
  for (const info of clients.values()) {
    if (info.clientId === clientId) count++;
  }
  return count;
}

function cleanupOrphans(): void {
  for (const [, manager] of managers) {
    manager.cleanupOrphans();
  }
}

cleanupOrphans();
console.log("[tmux] orphaned sessions cleaned");

function resolveClientId(reqUrl: URL): string {
  const mode = reqUrl.searchParams.get("mode") ?? "independent";
  if (mode === "mirror") return MIRROR_CLIENT_ID;
  return reqUrl.searchParams.get("clientId") ?? "default";
}

function handleDisconnect(ws: WebSocket, manager: TerminalManager): void {
  const info = clients.get(ws);
  clients.delete(ws);
  if (!info) return;
  clearInterval(info.batteryInterval);
  const remaining = clientCountForId(info.clientId);
  console.log(`[ws] client disconnected (id=${info.clientId}, remaining=${remaining}, total=${clients.size})`);
  if (remaining === 0) {
    manager.detachAll();
  }
}

wss.on("connection", (ws, req) => {
  const reqUrl = new URL(req.url ?? "/", `https://${req.headers.host}`);
  const clientId = resolveClientId(reqUrl);
  const manager = getOrCreateManager(clientId);

  const sendBattery = (): void => {
    const info = getBattery();
    ws.send(JSON.stringify({ type: "battery", ...info }));
  };

  sendBattery();
  const batteryInterval = setInterval(sendBattery, 60_000);

  clients.set(ws, { ws, clientId, batteryInterval });
  console.log(`[ws] client connected (id=${clientId}, total=${clients.size})`);

  const existingSessions = manager.getActiveSessions();
  if (existingSessions.length > 0) {
    ws.send(JSON.stringify({
      type: "sessions-available",
      tabIds: existingSessions
    }));
  }

  ws.on("message", (raw) => {
    const message = JSON.parse(String(raw));
    handleMessage(message, manager);
  });

  ws.on("close", () => handleDisconnect(ws, manager));
});

process.on("SIGTERM", () => {
  console.log("[server] SIGTERM received, cleaning up");
  for (const [, manager] of managers) {
    manager.destroyAll();
  }
  process.exit(0);
});

process.on("SIGINT", () => {
  console.log("[server] SIGINT received, cleaning up");
  for (const [, manager] of managers) {
    manager.destroyAll();
  }
  process.exit(0);
});

server.listen(PORT, "0.0.0.0", () => {
  console.log(`[server] running on https://0.0.0.0:${PORT}`);
});
