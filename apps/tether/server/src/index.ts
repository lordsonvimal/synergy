import { readFileSync } from "fs";
import { resolve } from "path";
import { URL } from "url";
import https from "https";
import express from "express";
import { WebSocketServer } from "ws";
import { TerminalManager } from "./terminal.js";
import { handleMessage } from "./handler.js";
import { getBattery } from "./battery.js";
import { validateToken } from "./auth.js";

const PORT = Number(process.env.PORT) || 5100;
const CERT_DIR = resolve(import.meta.dirname, "../../certs");

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

wss.on("connection", (ws) => {
  console.log("[ws] client connected");

  const manager = new TerminalManager((data) => {
    ws.send(JSON.stringify(data));
  });

  const sendBattery = (): void => {
    const info = getBattery();
    ws.send(JSON.stringify({ type: "battery", ...info }));
  };

  sendBattery();
  const batteryInterval = setInterval(sendBattery, 60_000);

  ws.on("message", (raw) => {
    const message = JSON.parse(String(raw));
    handleMessage(message, manager);
  });

  ws.on("close", () => {
    console.log("[ws] client disconnected");
    clearInterval(batteryInterval);
    manager.destroyAll();
  });
});

server.listen(PORT, "0.0.0.0", () => {
  console.log(`[server] running on https://0.0.0.0:${PORT}`);
});
