import {
  Component,
  Show,
  createEffect,
  createSignal,
  on,
  onCleanup,
  onMount
} from "solid-js";
import { useConnection } from "../context/connection.js";
import { useSettings } from "../context/settings.js";
import { usePanes, LeafNode, SplitDirection } from "../context/panes.js";
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

type DropZone = "left" | "right" | "top" | "bottom" | "center" | null;

function getDropZone(e: DragEvent, el: HTMLElement): DropZone {
  const rect = el.getBoundingClientRect();
  const x = (e.clientX - rect.left) / rect.width;
  const y = (e.clientY - rect.top) / rect.height;
  const threshold = 0.25;

  if (x < threshold) return "left";
  if (x > 1 - threshold) return "right";
  if (y < threshold) return "top";
  if (y > 1 - threshold) return "bottom";
  return "center";
}

function zoneToSplit(
  zone: DropZone
): { direction: SplitDirection; insertBefore: boolean } | null {
  if (zone === "left") return { direction: "horizontal", insertBefore: true };
  if (zone === "right") return { direction: "horizontal", insertBefore: false };
  if (zone === "top") return { direction: "vertical", insertBefore: true };
  if (zone === "bottom") return { direction: "vertical", insertBefore: false };
  return null;
}

interface PaneTerminalProps {
  paneId: string;
  pane: LeafNode;
}

export const PaneTerminal: Component<PaneTerminalProps> = (props) => {
  let containerRef: HTMLDivElement | undefined;
  const { onMessage, send } = useConnection();
  const { settings } = useSettings();
  const { activePaneId, splitPaneWithTab, moveTab } = usePanes();
  const [dropZone, setDropZone] = createSignal<DropZone>(null);

  const activeTabId = () => props.pane.activeTabId;

  const paneTabIds = (): Set<string> =>
    new Set(props.pane.tabs.map((t) => t.id));

  onMount(() => {
    onMessage((data) => {
      const msg = data as { type: string; tabId?: string; data?: string };
      if (!msg.tabId || !paneTabIds().has(msg.tabId)) return;
      if (msg.type === "pty" && msg.data) {
        getInstance(msg.tabId)?.terminal.write(msg.data);
      } else if (msg.type === "command-complete") {
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

      for (const child of Array.from(containerRef.children)) {
        const el = child as HTMLElement;
        if (el.dataset.dropOverlay !== undefined) continue;
        el.style.display = "none";
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

  onCleanup(() => {});

  const focusTerminal = (): void => {
    const tabId = activeTabId();
    if (tabId) getInstance(tabId)?.terminal.focus();
  };

  const handleDragOver = (e: DragEvent): void => {
    if (!e.dataTransfer) return;
    if (!e.dataTransfer.types.includes("application/x-tether-tab")) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    if (containerRef) {
      setDropZone(getDropZone(e, containerRef));
    }
  };

  const handleDragLeave = (e: DragEvent): void => {
    if (
      containerRef &&
      e.relatedTarget instanceof Node &&
      containerRef.contains(e.relatedTarget)
    ) {
      return;
    }
    setDropZone(null);
  };

  const handleDrop = (e: DragEvent): void => {
    e.preventDefault();
    e.stopPropagation();
    const zone = dropZone();
    setDropZone(null);
    if (!e.dataTransfer) return;

    const raw = e.dataTransfer.getData("application/x-tether-tab");
    if (!raw) return;
    const { sourcePaneId, tabId } = JSON.parse(raw) as {
      sourcePaneId: string;
      tabId: string;
    };

    const split = zoneToSplit(zone);
    if (split) {
      splitPaneWithTab(
        props.paneId,
        split.direction,
        split.insertBefore,
        sourcePaneId,
        tabId
      );
    } else if (zone === "center" && sourcePaneId !== props.paneId) {
      moveTab(sourcePaneId, tabId, props.paneId);
    }
  };

  const overlayClasses = (): string => {
    const zone = dropZone();
    if (!zone) return "hidden";
    const base =
      "absolute pointer-events-none bg-primary/20 border-2 border-primary border-dashed rounded-sm";
    if (zone === "left") return `${base} inset-y-0 left-0 w-1/2`;
    if (zone === "right") return `${base} inset-y-0 right-0 w-1/2`;
    if (zone === "top") return `${base} inset-x-0 top-0 h-1/2`;
    if (zone === "bottom") return `${base} inset-x-0 bottom-0 h-1/2`;
    return `${base} inset-0`;
  };

  return (
    <div
      ref={containerRef}
      class="relative flex-1 min-h-0 min-w-0 overflow-hidden"
      data-testid={`pane-terminal-${props.paneId}`}
      onClick={focusTerminal}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <Show when={dropZone()}>
        <div class={overlayClasses()} data-drop-overlay />
      </Show>
    </div>
  );
};
