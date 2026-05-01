import { spawn, IPty } from "node-pty";
import { stripAnsi } from "./ansi.js";
import { detectPermission } from "./permissions.js";

export interface ServerMessage {
  type: string;
  [key: string]: unknown;
}

type MessageCallback = (message: ServerMessage) => void;

export function createTerminal(onMessage: MessageCallback): IPty {
  const pty = spawn("/bin/zsh", ["-l", "-c", "claude"], {
    name: "xterm-256color",
    cols: 120,
    rows: 40,
    cwd: process.cwd(),
    env: process.env as Record<string, string>
  });

  let buffer = "";
  let fullResponse = "";
  let idleTimer: ReturnType<typeof setTimeout> | null = null;

  const IDLE_TIMEOUT = 500;

  function resetIdleTimer(): void {
    if (idleTimer) {
      clearTimeout(idleTimer);
    }
    idleTimer = setTimeout(() => {
      if (fullResponse) {
        onMessage({ type: "output_complete", fullText: fullResponse });
        fullResponse = "";
      }
    }, IDLE_TIMEOUT);
  }

  pty.onData((raw) => {
    const text = stripAnsi(raw);
    buffer += text;

    const permission = detectPermission(buffer);
    if (permission) {
      if (fullResponse) {
        onMessage({ type: "output_complete", fullText: fullResponse });
        fullResponse = "";
      }
      onMessage({
        type: "permission",
        action: permission.action,
        options: permission.options
      });
      buffer = "";
      if (idleTimer) {
        clearTimeout(idleTimer);
      }
      return;
    }

    fullResponse += text;
    onMessage({ type: "output", text, streaming: true });
    resetIdleTimer();
  });

  pty.onExit(({ exitCode }) => {
    if (idleTimer) {
      clearTimeout(idleTimer);
    }
    if (fullResponse) {
      onMessage({ type: "output_complete", fullText: fullResponse });
      fullResponse = "";
    }
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
