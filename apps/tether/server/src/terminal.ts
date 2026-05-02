import { spawn, IPty } from "node-pty";
import { basename } from "path";
import { detectPermission } from "./permissions.js";

export interface ServerMessage {
  type: string;
  [key: string]: unknown;
}

type MessageCallback = (message: ServerMessage) => void;

export function createTerminal(onMessage: MessageCallback): IPty {
  const shell = process.env.SHELL || "/bin/zsh";
  const shellName = basename(shell);
  const pty = spawn(shell, ["-l"], {
    name: "xterm-256color",
    cols: 120,
    rows: 40,
    cwd: process.env.HOME || process.cwd(),
    env: process.env as Record<string, string>
  });

  let buffer = "";
  const MAX_BUFFER = 4096;
  let wasRunning = false;
  let pollInterval: ReturnType<typeof setInterval> | undefined;
  let idleTimer: ReturnType<typeof setTimeout> | undefined;
  let outputBytes = 0;
  const IDLE_THRESHOLD_MS = 3000;
  const MIN_OUTPUT_BYTES = 100;

  pollInterval = setInterval(() => {
    try {
      const fg = basename(pty.process);
      const isShell = fg === shellName;
      if (wasRunning && isShell) {
        clearTimeout(idleTimer);
        outputBytes = 0;
        onMessage({ type: "command-complete" });
      }
      wasRunning = !isShell;
    } catch {
      // PTY may be disposed
    }
  }, 500);

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

    // Idle detection for interactive processes (e.g. Claude CLI)
    const fg = basename(pty.process);
    const isShell = fg === shellName;
    if (!isShell) {
      outputBytes += raw.length;
      clearTimeout(idleTimer);
      idleTimer = setTimeout(() => {
        if (outputBytes >= MIN_OUTPUT_BYTES) {
          onMessage({ type: "command-complete" });
        }
        outputBytes = 0;
      }, IDLE_THRESHOLD_MS);
    }
  });

  pty.onExit(({ exitCode }) => {
    clearInterval(pollInterval);
    clearTimeout(idleTimer);
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
