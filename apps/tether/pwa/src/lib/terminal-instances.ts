import { Terminal as XTerm, ITheme } from "xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";

export const darkTheme: ITheme = {
  background: "#0B1120",
  foreground: "#F1F5F9",
  cursor: "#60A5FA",
  cursorAccent: "#0B1120",
  selectionBackground: "#1D4ED866",
  black: "#1E293B",
  red: "#F87171",
  green: "#4ADE80",
  yellow: "#FBBF24",
  blue: "#60A5FA",
  magenta: "#C084FC",
  cyan: "#22D3EE",
  white: "#F1F5F9",
  brightBlack: "#475569",
  brightRed: "#FCA5A5",
  brightGreen: "#86EFAC",
  brightYellow: "#FCD34D",
  brightBlue: "#93C5FD",
  brightMagenta: "#D8B4FE",
  brightCyan: "#67E8F9",
  brightWhite: "#FFFFFF"
};

export const lightTheme: ITheme = {
  background: "#FAFBFD",
  foreground: "#0F172A",
  cursor: "#1D4ED8",
  cursorAccent: "#FAFBFD",
  selectionBackground: "#1D4ED833",
  black: "#0F172A",
  red: "#B91C1C",
  green: "#15803D",
  yellow: "#A16207",
  blue: "#1D4ED8",
  magenta: "#7C3AED",
  cyan: "#0369A1",
  white: "#F1F5F9",
  brightBlack: "#475569",
  brightRed: "#DC2626",
  brightGreen: "#16A34A",
  brightYellow: "#CA8A04",
  brightBlue: "#2563EB",
  brightMagenta: "#8B5CF6",
  brightCyan: "#0284C7",
  brightWhite: "#FFFFFF"
};

export const FONT_SIZE_MAP: Record<string, number> = {
  small: 12,
  medium: 14,
  large: 17
};

export interface TerminalInstance {
  terminal: XTerm;
  fitAddon: FitAddon;
  resizeObserver: ResizeObserver | null;
  resizeTimer: ReturnType<typeof setTimeout> | undefined;
}

const instances = new Map<string, TerminalInstance>();

export function getInstance(tabId: string): TerminalInstance | undefined {
  return instances.get(tabId);
}

export function getAllInstances(): Map<string, TerminalInstance> {
  return instances;
}

export function createInstance(
  tabId: string,
  container: HTMLDivElement,
  settings: { theme: string; fontSize: string },
  send: (msg: unknown) => void
): TerminalInstance {
  const existing = instances.get(tabId);
  if (existing) {
    if (
      existing.terminal.element &&
      !container.contains(existing.terminal.element)
    ) {
      container.appendChild(existing.terminal.element);
      requestAnimationFrame(() => {
        existing.fitAddon.fit();
        send({
          type: "resize",
          tabId,
          cols: existing.terminal.cols,
          rows: existing.terminal.rows
        });
      });
    }
    return existing;
  }

  const currentTheme = settings.theme === "light" ? lightTheme : darkTheme;
  const terminal = new XTerm({
    theme: currentTheme,
    fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
    fontSize: FONT_SIZE_MAP[settings.fontSize] ?? 14,
    lineHeight: 1.4,
    cursorBlink: true,
    scrollback: 5000,
    convertEol: true,
    allowProposedApi: true,
    scrollOnUserInput: true
  });

  const fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);
  terminal.loadAddon(new WebLinksAddon());

  terminal.open(container);

  const xtermEl = container.querySelector(".xterm") as HTMLElement;
  if (xtermEl) xtermEl.style.height = "100%";
  const screenEl = container.querySelector(".xterm-screen") as HTMLElement;
  if (screenEl) screenEl.style.height = "100%";

  requestAnimationFrame(() => {
    fitAddon.fit();
    send({
      type: "resize",
      tabId,
      cols: terminal.cols,
      rows: terminal.rows
    });
  });

  terminal.onData((data) => {
    send({ type: "key", tabId, data });
  });

  const instance: TerminalInstance = {
    terminal,
    fitAddon,
    resizeObserver: null,
    resizeTimer: undefined
  };
  instances.set(tabId, instance);
  return instance;
}

export function destroyInstance(tabId: string): void {
  const instance = instances.get(tabId);
  if (!instance) return;
  clearTimeout(instance.resizeTimer);
  instance.resizeObserver?.disconnect();
  instance.terminal.dispose();
  instances.delete(tabId);
}

export function destroyAllInstances(): void {
  for (const [, inst] of instances) {
    clearTimeout(inst.resizeTimer);
    inst.resizeObserver?.disconnect();
    inst.terminal.dispose();
  }
  instances.clear();
}
