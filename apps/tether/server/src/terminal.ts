import { spawn, IPty } from "node-pty";
import { basename } from "path";
import { detectPermission } from "./permissions.js";

export interface ServerMessage {
  type: string;
  [key: string]: unknown;
}

type MessageCallback = (message: ServerMessage) => void;

interface TabSession {
  pty: IPty;
  pollInterval: ReturnType<typeof setInterval>;
  idleTimer: ReturnType<typeof setTimeout> | undefined;
}

export class TerminalManager {
  private tabs = new Map<string, TabSession>();
  private onMessage: MessageCallback;

  constructor(onMessage: MessageCallback) {
    this.onMessage = onMessage;
  }

  createTab(tabId: string): void {
    if (this.tabs.has(tabId)) return;

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
    let idleTimer: ReturnType<typeof setTimeout> | undefined;
    let outputBytes = 0;
    const IDLE_THRESHOLD_MS = 3000;
    const MIN_OUTPUT_BYTES = 100;

    const pollInterval = setInterval(() => {
      try {
        const fg = basename(pty.process);
        const isShell = fg === shellName;
        if (wasRunning && isShell) {
          clearTimeout(idleTimer);
          outputBytes = 0;
          this.onMessage({ type: "command-complete", tabId });
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
        this.onMessage({
          type: "permission",
          tabId,
          action: permission.action,
          options: permission.options
        });
        buffer = "";
        return;
      }

      if (buffer.length > MAX_BUFFER) {
        buffer = buffer.slice(-MAX_BUFFER);
      }

      this.onMessage({ type: "pty", tabId, data: raw });

      const fg = basename(pty.process);
      const isShell = fg === shellName;
      if (!isShell) {
        outputBytes += raw.length;
        clearTimeout(idleTimer);
        idleTimer = setTimeout(() => {
          if (outputBytes >= MIN_OUTPUT_BYTES) {
            this.onMessage({ type: "command-complete", tabId });
          }
          outputBytes = 0;
        }, IDLE_THRESHOLD_MS);
      }
    });

    pty.onExit(({ exitCode }) => {
      clearInterval(pollInterval);
      clearTimeout(idleTimer);
      this.tabs.delete(tabId);
      this.onMessage({ type: "tab-exited", tabId, exitCode });
    });

    this.tabs.set(tabId, { pty, pollInterval, idleTimer });
    this.onMessage({ type: "tab-created", tabId });
  }

  getTab(tabId: string): IPty | undefined {
    return this.tabs.get(tabId)?.pty;
  }

  closeTab(tabId: string): void {
    const session = this.tabs.get(tabId);
    if (!session) return;
    clearInterval(session.pollInterval);
    clearTimeout(session.idleTimer);
    session.pty.kill();
    this.tabs.delete(tabId);
  }

  destroyAll(): void {
    for (const [tabId] of this.tabs) {
      this.closeTab(tabId);
    }
  }
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
