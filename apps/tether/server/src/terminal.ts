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
  private clientId: string;

  constructor(clientId: string, gracePeriodMs: number = 300_000) {
    this.clientId = clientId;
    this.gracePeriodMs = gracePeriodMs;

    if (!tmuxAvailable()) {
      throw new Error(
        "tmux is not installed. Install with: brew install tmux"
      );
    }
  }

  private sessionKey(tabId: string): string {
    return `${this.clientId}_${tabId}`;
  }

  setMessageCallback(cb: MessageCallback): void {
    this.onMessage = cb;
  }

  private reattachExistingTab(tabId: string): boolean {
    const session = this.tabs.get(tabId);
    if (!session) return false;
    session.clientAttached = true;
    this.replayScrollback(tabId);
    this.onMessage({ type: "tab-created", tabId, restored: true });
    return true;
  }

  private handlePtyData(
    raw: string,
    tabId: string,
    ctx: { buffer: string; outputBytes: number; idleTimer: ReturnType<typeof setTimeout> | undefined },
    shellName: string,
    pty: IPty
  ): void {
    const session = this.tabs.get(tabId);
    if (!session?.clientAttached) return;

    ctx.buffer += raw;

    const permission = detectPermission(ctx.buffer);
    if (permission) {
      this.onMessage({ type: "permission", tabId, action: permission.action, options: permission.options });
      ctx.buffer = "";
      return;
    }

    if (ctx.buffer.length > 4096) {
      ctx.buffer = ctx.buffer.slice(-4096);
    }

    this.onMessage({ type: "pty", tabId, data: raw });
    this.trackIdleCompletion(raw, tabId, ctx, shellName, pty);
  }

  private trackIdleCompletion(
    raw: string,
    tabId: string,
    ctx: { outputBytes: number; idleTimer: ReturnType<typeof setTimeout> | undefined },
    shellName: string,
    pty: IPty
  ): void {
    try {
      const fg = basename(pty.process);
      const isShell = fg === shellName || fg === "tmux";
      if (!isShell) {
        ctx.outputBytes += raw.length;
        clearTimeout(ctx.idleTimer);
        ctx.idleTimer = setTimeout(() => {
          if (ctx.outputBytes >= 100) {
            this.onMessage({ type: "command-complete", tabId });
          }
          ctx.outputBytes = 0;
        }, 3000);
      }
    } catch {
      // pty.process may throw if disposed
    }
  }

  private ensureTmuxSession(key: string, cols: number, rows: number): boolean {
    const exists = sessionExists(key);
    if (!exists) createSession(key, cols, rows);
    return exists;
  }

  private setupPty(
    tabId: string,
    key: string,
    cols: number,
    rows: number
  ): { pty: IPty; pollInterval: ReturnType<typeof setInterval>; ctx: { buffer: string; outputBytes: number; idleTimer: ReturnType<typeof setTimeout> | undefined } } {
    const pty = attachSession(key, cols, rows);
    const shellName = basename(process.env.SHELL || "/bin/zsh");
    let wasRunning = false;
    const ctx = { buffer: "", outputBytes: 0, idleTimer: undefined as ReturnType<typeof setTimeout> | undefined };

    const pollInterval = setInterval(() => {
      try {
        const fg = basename(pty.process);
        const isShell = fg === shellName || fg === "tmux";
        if (wasRunning && isShell) {
          clearTimeout(ctx.idleTimer);
          ctx.outputBytes = 0;
          this.onMessage({ type: "command-complete", tabId });
        }
        wasRunning = !isShell;
      } catch {
        // PTY may be disposed
      }
    }, 500);

    pty.onData(raw => this.handlePtyData(raw, tabId, ctx, shellName, pty));
    pty.onExit(() => {
      clearInterval(pollInterval);
      clearTimeout(ctx.idleTimer);
      this.tabs.delete(tabId);
      if (sessionExists(key)) killSession(key);
      this.onMessage({ type: "tab-exited", tabId, exitCode: 0 });
    });

    return { pty, pollInterval, ctx };
  }

  private registerNewTab(tabId: string, key: string, cols: number, rows: number, restored: boolean): void {
    const { pty, pollInterval, ctx } = this.setupPty(tabId, key, cols, rows);
    this.tabs.set(tabId, { pty, pollInterval, idleTimer: ctx.idleTimer, clientAttached: true });
    if (restored) this.replayScrollback(tabId);
    this.onMessage({ type: "tab-created", tabId, restored });
  }

  private isClientAttached(tabId: string): boolean {
    return this.tabs.get(tabId)?.clientAttached === true;
  }

  createTab(tabId: string, cols: number = 120, rows: number = 40): void {
    if (this.isClientAttached(tabId)) return;
    this.clearGraceTimer(tabId);
    const key = this.sessionKey(tabId);
    const isExisting = this.ensureTmuxSession(key, cols, rows);
    if (this.reattachExistingTab(tabId)) return;
    this.registerNewTab(tabId, key, cols, rows, isExisting);
  }

  private replayScrollback(tabId: string): void {
    const scrollback = captureScrollback(this.sessionKey(tabId));
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
    resizeSession(this.sessionKey(tabId), cols, rows);
  }

  closeTab(tabId: string): void {
    this.clearGraceTimer(tabId);
    const key = this.sessionKey(tabId);
    const session = this.tabs.get(tabId);
    if (!session) {
      killSession(key);
      return;
    }
    clearInterval(session.pollInterval);
    clearTimeout(session.idleTimer);
    session.pty.kill();
    this.tabs.delete(tabId);
    killSession(key);
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
      console.log(`[tmux] grace period expired for ${this.sessionKey(tabId)}, killing session`);
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
    const prefix = `${this.clientId}_`;
    const allSessions = listTetherSessions();
    for (const sessionKey of allSessions) {
      if (sessionKey.startsWith(prefix)) {
        const tabId = sessionKey.slice(prefix.length);
        if (!this.tabs.has(tabId)) {
          console.log(`[tmux] cleaning orphaned session: ${sessionKey}`);
          killSession(sessionKey);
        }
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
