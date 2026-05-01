import { readFileSync } from "fs";
import { resolve } from "path";
import https from "https";
import express from "express";
import { WebSocketServer } from "ws";
import { createTerminal, destroyTerminal } from "./terminal.js";
import { handleMessage } from "./handler.js";

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

const wss = new WebSocketServer({ server });

wss.on("connection", (ws) => {
  console.log("[ws] client connected");

  const pty = createTerminal((data) => {
    ws.send(JSON.stringify(data));
  });

  ws.on("message", (raw) => {
    const message = JSON.parse(String(raw));
    handleMessage(message, pty);
  });

  ws.on("close", () => {
    console.log("[ws] client disconnected");
    destroyTerminal(pty);
  });
});

server.listen(PORT, () => {
  console.log(`[server] running on https://localhost:${PORT}`);
});
