import { Component, createEffect, on, onCleanup, onMount } from "solid-js";
import { useConnection } from "../context/connection.js";
import { useSettings } from "../context/settings.js";
import { usePanes, LeafNode } from "../context/panes.js";
import { playChime } from "../lib/chime.js";
import {
  createInstance,
  getInstance,
  getAllInstances,
  darkTheme,
  lightTheme,
  FONT_SIZE_MAP
} from "../lib/terminal-instances.js";
import "xterm/css/xterm.css";

interface PaneTerminalProps {
  paneId: string;
  pane: LeafNode;
}

export const PaneTerminal: Component<PaneTerminalProps> = (props) => {
  let containerRef: HTMLDivElement | undefined;
  const { onMessage, send } = useConnection();
  const { settings } = useSettings();
  const { activePaneId } = usePanes();

  const activeTabId = () => props.pane.activeTabId;

  onMount(() => {
    onMessage((data) => {
      const msg = data as { type: string; tabId?: string; data?: string };
      if (msg.type === "pty" && msg.data && msg.tabId) {
        getInstance(msg.tabId)?.terminal.write(msg.data);
      } else if (msg.type === "command-complete" && msg.tabId) {
        if (
          settings().chimeEnabled &&
          msg.tabId === activeTabId() &&
          activePaneId() === props.paneId
        ) {
          playChime();
        }
      }
    });
  });

  createEffect(
    on(activeTabId, (tabId) => {
      if (!containerRef || !tabId) return;

      // Hide all instances in this container
      for (const child of Array.from(containerRef.children)) {
        (child as HTMLElement).style.display = "none";
      }

      const existing = getInstance(tabId);
      if (existing) {
        if (
          existing.terminal.element &&
          !containerRef.contains(existing.terminal.element)
        ) {
          containerRef.appendChild(existing.terminal.element);
        }
        if (existing.terminal.element) {
          existing.terminal.element.style.display = "";
        }
        requestAnimationFrame(() => {
          existing.fitAddon.fit();
          existing.terminal.focus();
        });
        return;
      }

      const instance = createInstance(
        tabId,
        containerRef,
        { theme: settings().theme, fontSize: settings().fontSize },
        send
      );

      // Set up resize observer for this instance
      let resizeTimer: ReturnType<typeof setTimeout> | undefined;
      const resizeObserver = new ResizeObserver(() => {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(() => {
          instance.fitAddon.fit();
          send({
            type: "resize",
            tabId,
            cols: instance.terminal.cols,
            rows: instance.terminal.rows
          });
        }, 100);
      });
      resizeObserver.observe(containerRef);
      instance.resizeObserver = resizeObserver;
      instance.resizeTimer = resizeTimer;
    })
  );

  createEffect(() => {
    const theme = settings().theme === "light" ? lightTheme : darkTheme;
    for (const [, inst] of getAllInstances()) {
      inst.terminal.options.theme = theme;
    }
  });

  createEffect(() => {
    const size = FONT_SIZE_MAP[settings().fontSize] ?? 14;
    for (const [tabId, inst] of getAllInstances()) {
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
    // Don't destroy instances here — they belong to the global map
    // and may be reused if the pane is reattached
  });

  const focusTerminal = (): void => {
    const tabId = activeTabId();
    if (tabId) getInstance(tabId)?.terminal.focus();
  };

  return (
    <div
      ref={containerRef}
      class="flex-1 min-h-0 min-w-0 overflow-hidden"
      data-testid={`pane-terminal-${props.paneId}`}
      onClick={focusTerminal}
    />
  );
};
