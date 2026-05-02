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

const manager = new TerminalManager(GRACE_PERIOD_MS);

manager.cleanupOrphans();
console.log("[tmux] orphaned sessions cleaned");

let activeClient: WebSocket | null = null;

function sendToClient(data: Record<string, unknown>): void {
  if (activeClient && activeClient.readyState === WebSocket.OPEN) {
    activeClient.send(JSON.stringify(data));
  }
}

manager.setMessageCallback(sendToClient);

wss.on("connection", (ws) => {
  console.log("[ws] client connected");

  if (activeClient && activeClient.readyState === WebSocket.OPEN) {
    console.log("[ws] replacing previous client connection");
    activeClient.close(4000, "Replaced by new connection");
  }

  activeClient = ws;
  manager.setMessageCallback(sendToClient);

  const sendBattery = (): void => {
    const info = getBattery();
    ws.send(JSON.stringify({ type: "battery", ...info }));
  };

  sendBattery();
  const batteryInterval = setInterval(sendBattery, 60_000);

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

  ws.on("close", () => {
    console.log("[ws] client disconnected");
    clearInterval(batteryInterval);
    if (activeClient === ws) {
      activeClient = null;
    }
    manager.detachAll();
  });
});

process.on("SIGTERM", () => {
  console.log("[server] SIGTERM received, cleaning up");
  manager.destroyAll();
  process.exit(0);
});

process.on("SIGINT", () => {
  console.log("[server] SIGINT received, cleaning up");
  manager.destroyAll();
  process.exit(0);
});

server.listen(PORT, "0.0.0.0", () => {
  console.log(`[server] running on https://0.0.0.0:${PORT}`);
});
