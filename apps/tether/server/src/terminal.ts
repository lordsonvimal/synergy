import { IPty } from "node-pty";
import { basename } from "path";
import { detectPermission } from "./permissions.js";
import {
  sessionExists,
  createSession,
  attachSession,
  resizeSession,
  killSession,
  listTetherSessions,
  captureScrollback,
  tmuxAvailable
} from "./tmux.js";

export interface ServerMessage {
  type: string;
  [key: string]: unknown;
}

type MessageCallback = (message: ServerMessage) => void;

interface TabSession {
  pty: IPty;
  pollInterval: ReturnType<typeof setInterval>;
  idleTimer: ReturnType<typeof setTimeout> | undefined;
  clientAttached: boolean;
}

export class TerminalManager {
  private tabs = new Map<string, TabSession>();
  private onMessage: MessageCallback = () => {};
  private graceTimers = new Map<string, ReturnType<typeof setTimeout>>();
  private gracePeriodMs: number;

  constructor(gracePeriodMs: number = 300_000) {
    this.gracePeriodMs = gracePeriodMs;

    if (!tmuxAvailable()) {
      throw new Error(
        "tmux is not installed. Install with: brew install tmux"
      );
    }
  }

  setMessageCallback(cb: MessageCallback): void {
    this.onMessage = cb;
  }

  createTab(tabId: string, cols: number = 120, rows: number = 40): void {
    if (this.tabs.has(tabId) && this.tabs.get(tabId)!.clientAttached) {
      return;
    }

    this.clearGraceTimer(tabId);

    const isExisting = sessionExists(tabId);

    if (!isExisting) {
      createSession(tabId, cols, rows);
    }

    if (this.tabs.has(tabId)) {
      const session = this.tabs.get(tabId)!;
      session.clientAttached = true;
      this.replayScrollback(tabId);
      this.onMessage({ type: "tab-created", tabId, restored: true });
      return;
    }

    const pty = attachSession(tabId, cols, rows);
    const shell = process.env.SHELL || "/bin/zsh";
    const shellName = basename(shell);

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
        const isShell = fg === shellName || fg === "tmux";
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
      const session = this.tabs.get(tabId);
      if (!session?.clientAttached) return;

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

      try {
        const fg = basename(pty.process);
        const isShell = fg === shellName || fg === "tmux";
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
      } catch {
        // pty.process may throw if disposed
      }
    });

    pty.onExit(() => {
      clearInterval(pollInterval);
      clearTimeout(idleTimer);
      this.tabs.delete(tabId);

      if (sessionExists(tabId)) {
        killSession(tabId);
      }

      this.onMessage({ type: "tab-exited", tabId, exitCode: 0 });
    });

    this.tabs.set(tabId, { pty, pollInterval, idleTimer, clientAttached: true });

    if (isExisting) {
      this.replayScrollback(tabId);
      this.onMessage({ type: "tab-created", tabId, restored: true });
    } else {
      this.onMessage({ type: "tab-created", tabId, restored: false });
    }
  }

  private replayScrollback(tabId: string): void {
    const scrollback = captureScrollback(tabId);
    if (scrollback) {
      this.onMessage({ type: "pty-replay", tabId, data: scrollback });
    }
  }

  getTab(tabId: string): IPty | undefined {
    return this.tabs.get(tabId)?.pty;
  }

  resizeTab(tabId: string, cols: number, rows: number): void {
    const session = this.tabs.get(tabId);
    if (!session) return;
    session.pty.resize(cols, rows);
    resizeSession(tabId, cols, rows);
  }

  closeTab(tabId: string): void {
    this.clearGraceTimer(tabId);
    const session = this.tabs.get(tabId);
    if (!session) {
      killSession(tabId);
      return;
    }
    clearInterval(session.pollInterval);
    clearTimeout(session.idleTimer);
    session.pty.kill();
    this.tabs.delete(tabId);
    killSession(tabId);
  }

  detachAll(): void {
    for (const [tabId, session] of this.tabs) {
      session.clientAttached = false;
      this.startGraceTimer(tabId);
    }
  }

  private startGraceTimer(tabId: string): void {
    this.clearGraceTimer(tabId);
    const timer = setTimeout(() => {
      console.log(`[tmux] grace period expired for ${tabId}, killing session`);
      this.closeTab(tabId);
      this.graceTimers.delete(tabId);
    }, this.gracePeriodMs);
    this.graceTimers.set(tabId, timer);
  }

  private clearGraceTimer(tabId: string): void {
    const timer = this.graceTimers.get(tabId);
    if (timer) {
      clearTimeout(timer);
      this.graceTimers.delete(tabId);
    }
  }

  destroyAll(): void {
    for (const [tabId] of this.tabs) {
      this.closeTab(tabId);
    }
  }

  cleanupOrphans(): void {
    const orphans = listTetherSessions();
    for (const tabId of orphans) {
      if (!this.tabs.has(tabId)) {
        console.log(`[tmux] cleaning orphaned session: ${tabId}`);
        killSession(tabId);
      }
    }
  }

  getActiveSessions(): string[] {
    return Array.from(this.tabs.keys());
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
