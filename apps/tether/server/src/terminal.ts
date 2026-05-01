import { spawn, IPty } from "node-pty";
import { detectPermission } from "./permissions.js";

export interface ServerMessage {
  type: string;
  [key: string]: unknown;
}

type MessageCallback = (message: ServerMessage) => void;

export function createTerminal(onMessage: MessageCallback): IPty {
  const shell = process.env.SHELL || "/bin/zsh";
  const pty = spawn(shell, ["-l"], {
    name: "xterm-256color",
    cols: 120,
    rows: 40,
    cwd: process.env.HOME || process.cwd(),
    env: process.env as Record<string, string>
  });

  let buffer = "";
  const MAX_BUFFER = 4096;

  pty.onData((raw) => {
    buffer += raw;

    const permission = detectPermission(buffer);
    if (permission) {
      onMessage({
        type: "permission",
        action: permission.action,
        options: permission.options
      });
      buffer = "";
      return;
    }

    if (buffer.length > MAX_BUFFER) {
      buffer = buffer.slice(-MAX_BUFFER);
    }

    onMessage({ type: "pty", data: raw });
  });

  pty.onExit(({ exitCode }) => {
    onMessage({
      type: "status",
      connected: false,
      claudeReady: false,
      exitCode
    });
  });

  return pty;
}

export function writeToTerminal(pty: IPty, text: string): void {
  pty.write(text + "\r");
}

export function resizeTerminal(pty: IPty, cols: number, rows: number): void {
  pty.resize(cols, rows);
}

export function stopTerminal(pty: IPty): void {
  pty.write("\x03");
}

export function destroyTerminal(pty: IPty): void {
  pty.kill();
}
