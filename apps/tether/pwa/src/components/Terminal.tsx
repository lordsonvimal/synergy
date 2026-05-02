import {
  Component,
  createEffect,
  on,
  onCleanup,
  onMount
} from "solid-js";
import { Terminal as XTerm, ITheme } from "xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "xterm/css/xterm.css";
import { useConnection } from "../context/connection.js";
import { useSettings } from "../context/settings.js";
import { useTabs } from "../context/tabs.js";
import { playChime } from "../lib/chime.js";

const darkTheme: ITheme = {
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

const lightTheme: ITheme = {
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

const FONT_SIZE_MAP: Record<string, number> = {
  small: 12,
  medium: 14,
  large: 17
};

interface TerminalInstance {
  terminal: XTerm;
  fitAddon: FitAddon;
  resizeObserver: ResizeObserver;
  resizeTimer: ReturnType<typeof setTimeout> | undefined;
}

const instances = new Map<string, TerminalInstance>();

function getOrCreateInstance(
  tabId: string,
  container: HTMLDivElement,
  settings: { theme: string; fontSize: string },
  send: (msg: unknown) => void
): TerminalInstance {
  const existing = instances.get(tabId);
  if (existing) {
    if (existing.terminal.element && !container.contains(existing.terminal.element)) {
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
  const screenEl = container.querySelector(
    ".xterm-screen"
  ) as HTMLElement;
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

  let resizeTimer: ReturnType<typeof setTimeout> | undefined;
  const resizeObserver = new ResizeObserver(() => {
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(() => {
      fitAddon.fit();
      send({
        type: "resize",
        tabId,
        cols: terminal.cols,
        rows: terminal.rows
      });
    }, 100);
  });
  resizeObserver.observe(container);

  const instance: TerminalInstance = {
    terminal,
    fitAddon,
    resizeObserver,
    resizeTimer
  };
  instances.set(tabId, instance);
  return instance;
}

export function destroyTabTerminal(tabId: string): void {
  const instance = instances.get(tabId);
  if (!instance) return;
  clearTimeout(instance.resizeTimer);
  instance.resizeObserver.disconnect();
  instance.terminal.dispose();
  instances.delete(tabId);
}

export const Terminal: Component = () => {
  let containerRef: HTMLDivElement | undefined;
  const { onMessage, send } = useConnection();
  const { settings } = useSettings();
  const { activeTabId } = useTabs();

  onMount(() => {
    onMessage((data) => {
      const msg = data as {
        type: string;
        tabId?: string;
        data?: string;
      };
      if (msg.type === "pty" && msg.data && msg.tabId) {
        instances.get(msg.tabId)?.terminal.write(msg.data);
      } else if (msg.type === "command-complete" && msg.tabId) {
        if (settings().chimeEnabled && msg.tabId === activeTabId()) {
          playChime();
        }
      }
    });
  });

  createEffect(
    on(activeTabId, (tabId) => {
      if (!containerRef || !tabId) return;

      for (const [id, inst] of instances) {
        if (inst.terminal.element) {
          inst.terminal.element.style.display =
            id === tabId ? "" : "none";
        }
      }

      const existing = instances.get(tabId);
      if (existing) {
        if (existing.terminal.element) {
          existing.terminal.element.style.display = "";
        }
        requestAnimationFrame(() => {
          existing.fitAddon.fit();
          existing.terminal.focus();
        });
        return;
      }

      getOrCreateInstance(
        tabId,
        containerRef,
        { theme: settings().theme, fontSize: settings().fontSize },
        send
      );
    })
  );

  createEffect(() => {
    const theme = settings().theme === "light" ? lightTheme : darkTheme;
    for (const [, inst] of instances) {
      inst.terminal.options.theme = theme;
    }
  });

  createEffect(() => {
    const size = FONT_SIZE_MAP[settings().fontSize] ?? 14;
    for (const [tabId, inst] of instances) {
      inst.terminal.options.fontSize = size;
      inst.fitAddon.fit();
      send({
        type: "resize",
        tabId,
        cols: inst.terminal.cols,
        rows: inst.terminal.rows
      });
    }
  });

  onCleanup(() => {
    for (const [, inst] of instances) {
      clearTimeout(inst.resizeTimer);
      inst.resizeObserver.disconnect();
      inst.terminal.dispose();
    }
    instances.clear();
  });

  const focusTerminal = (): void => {
    instances.get(activeTabId())?.terminal.focus();
  };

  return (
    <div
      ref={containerRef}
      class="flex-1 min-h-0 min-w-0 overflow-hidden"
      data-testid="terminal-area"
      onClick={focusTerminal}
    />
  );
};
