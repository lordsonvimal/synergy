import { Component, onMount, onCleanup } from "solid-js";
import { Terminal as XTerm } from "xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "xterm/css/xterm.css";
import { useConnection } from "../context/connection.js";

export const Terminal: Component = () => {
  let containerRef: HTMLDivElement | undefined;
  let terminal: XTerm | undefined;
  let fitAddon: FitAddon | undefined;

  const { onMessage, send } = useConnection();

  onMount(() => {
    if (!containerRef) {
      return;
    }

    terminal = new XTerm({
      theme: {
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
      },
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      fontSize: 14,
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

    // Force xterm container to fill parent height via JS
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

    const resizeObserver = new ResizeObserver(() => {
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
    });

    resizeObserver.observe(containerRef);

    onCleanup(() => {
      resizeObserver.disconnect();
      terminal?.dispose();
    });
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
