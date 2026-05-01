import { spawn, IPty } from "node-pty";
import { stripAnsi } from "./ansi.js";
import { detectPermission } from "./permissions.js";

export interface ServerMessage {
  type: string;
  [key: string]: unknown;
}

type MessageCallback = (message: ServerMessage) => void;

export function createTerminal(onMessage: MessageCallback): IPty {
  const pty = spawn("claude", [], {
    name: "xterm-256color",
    cols: 120,
    rows: 40,
    cwd: process.cwd()
  });

  let buffer = "";

  pty.onData((raw) => {
    const text = stripAnsi(raw);
    buffer += text;

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

    onMessage({ type: "output", text, streaming: true });
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

export function stopTerminal(pty: IPty): void {
  pty.write("\x03");
}

export function destroyTerminal(pty: IPty): void {
  pty.kill();
}
