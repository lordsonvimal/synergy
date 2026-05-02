import { execSync, execFileSync } from "child_process";
import { spawn, IPty } from "node-pty";

const SESSION_PREFIX = "tether-";

export function sessionName(tabId: string): string {
  return `${SESSION_PREFIX}${tabId}`;
}

export function tmuxAvailable(): boolean {
  try {
    execFileSync("tmux", ["-V"], { stdio: "pipe" });
    return true;
  } catch {
    return false;
  }
}

export function sessionExists(tabId: string): boolean {
  try {
    execFileSync("tmux", ["has-session", "-t", sessionName(tabId)], {
      stdio: "pipe"
    });
    return true;
  } catch {
    return false;
  }
}

export function createSession(
  tabId: string,
  cols: number,
  rows: number
): void {
  const name = sessionName(tabId);
  const shell = process.env.SHELL || "/bin/zsh";
  execFileSync("tmux", [
    "new-session",
    "-d",
    "-s", name,
    "-x", String(cols),
    "-y", String(rows),
    shell
  ], {
    env: { ...process.env, TERM: "xterm-256color" },
    cwd: process.env.HOME || process.cwd(),
    stdio: "pipe"
  });
}

export function attachSession(tabId: string, cols: number, rows: number): IPty {
  const name = sessionName(tabId);
  const pty = spawn("tmux", ["attach-session", "-t", name], {
    name: "xterm-256color",
    cols,
    rows,
    cwd: process.env.HOME || process.cwd(),
    env: process.env as Record<string, string>
  });
  return pty;
}

export function resizeSession(tabId: string, cols: number, rows: number): void {
  try {
    execFileSync("tmux", [
      "resize-window",
      "-t", sessionName(tabId),
      "-x", String(cols),
      "-y", String(rows)
    ], { stdio: "pipe" });
  } catch {
    // Session may not exist yet
  }
}

export function killSession(tabId: string): void {
  try {
    execFileSync("tmux", ["kill-session", "-t", sessionName(tabId)], {
      stdio: "pipe"
    });
  } catch {
    // Session may already be dead
  }
}

export function listTetherSessions(): string[] {
  try {
    const output = execSync(
      "tmux list-sessions -F \"#{session_name}\" 2>/dev/null",
      { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] }
    );
    return output
      .trim()
      .split("\n")
      .filter(name => name.startsWith(SESSION_PREFIX))
      .map(name => name.slice(SESSION_PREFIX.length));
  } catch {
    return [];
  }
}

export function captureScrollback(tabId: string, lines: number = 5000): string {
  try {
    const output = execFileSync("tmux", [
      "capture-pane",
      "-t", sessionName(tabId),
      "-p",
      "-S", `-${lines}`,
      "-e"
    ], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return output;
  } catch {
    return "";
  }
}

export function killAllTetherSessions(): void {
  const sessions = listTetherSessions();
  for (const tabId of sessions) {
    killSession(tabId);
  }
}
