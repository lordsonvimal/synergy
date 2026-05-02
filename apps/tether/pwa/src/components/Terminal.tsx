import { Component, onMount, onCleanup, createEffect } from "solid-js";
import { Terminal as XTerm, ITheme } from "xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "xterm/css/xterm.css";
import { useConnection } from "../context/connection.js";
import { useSettings } from "../context/settings.js";

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

export const Terminal: Component = () => {
  let containerRef: HTMLDivElement | undefined;
  let terminal: XTerm | undefined;
  let fitAddon: FitAddon | undefined;

  const { onMessage, send } = useConnection();
  const { settings } = useSettings();

  onMount(() => {
    if (!containerRef) {
      return;
    }

    const currentTheme = settings().theme === "light" ? lightTheme : darkTheme;

    terminal = new XTerm({
      theme: currentTheme,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      fontSize: FONT_SIZE_MAP[settings().fontSize] ?? 14,
      lineHeight: 1.4,
      cursorBlink: true,
      scrollback: 5000,
      convertEol: true,
      allowProposedApi: true,
      scrollOnUserInput: true
    });

    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebLinksAddon());

    terminal.open(containerRef);

    const xtermEl = containerRef.querySelector(".xterm") as HTMLElement;
    if (xtermEl) {
      xtermEl.style.height = "100%";
    }
    const screenEl = containerRef.querySelector(".xterm-screen") as HTMLElement;
    if (screenEl) {
      screenEl.style.height = "100%";
    }

    requestAnimationFrame(() => {
      fitAddon?.fit();
      if (terminal) {
        send({
          type: "resize",
          cols: terminal.cols,
          rows: terminal.rows
        });
      }
    });

    terminal.onData((data) => {
      send({ type: "key", data });
    });

    onMessage((data) => {
      const msg = data as { type: string; data?: string };
      if (msg.type === "pty" && msg.data) {
        terminal?.write(msg.data);
      }
    });

    let resizeTimer: ReturnType<typeof setTimeout> | undefined;
    const resizeObserver = new ResizeObserver(() => {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(() => {
        fitAddon?.fit();
        if (terminal) {
          send({
            type: "resize",
            cols: terminal.cols,
            rows: terminal.rows
          });
        }
      }, 100);
    });

    resizeObserver.observe(containerRef);

    onCleanup(() => {
      clearTimeout(resizeTimer);
      resizeObserver.disconnect();
      terminal?.dispose();
    });
  });

  createEffect(() => {
    const theme = settings().theme === "light" ? lightTheme : darkTheme;
    if (terminal) {
      terminal.options.theme = theme;
    }
  });

  createEffect(() => {
    const size = FONT_SIZE_MAP[settings().fontSize] ?? 14;
    if (terminal) {
      terminal.options.fontSize = size;
      fitAddon?.fit();
      send({ type: "resize", cols: terminal.cols, rows: terminal.rows });
    }
  });

  const focusTerminal = (): void => {
    terminal?.focus();
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
