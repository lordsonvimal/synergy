import {
  Component,
  createEffect,
  createSignal,
  on,
  onCleanup,
  onMount,
  untrack
} from "solid-js";
import { useConnection } from "../context/connection.js";
import { useSettings } from "../context/settings.js";
import { usePanes, LeafNode, SplitDirection } from "../context/panes.js";
import { playChime } from "../lib/chime.js";
import {
  createInstance,
  getInstance,
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
  // eslint-disable-next-line no-unassigned-vars -- SolidJS ref pattern
  let containerRef: HTMLDivElement | undefined;
  const { onMessage, send } = useConnection();
  const { settings } = useSettings();
  const { activePaneId, splitPaneWithTab, moveTab } = usePanes();
  const [dropZone, setDropZone] = createSignal<DropZone>(null);

  const activeTabId = () => props.pane.activeTabId;

  const paneOwnsTab = (tabId: string): boolean =>
    props.pane.tabs.some((t) => t.id === tabId);

  const writePtyData = (tabId: string, data: string): void => {
    getInstance(tabId)?.terminal.write(data);
  };

  const replayPtyData = (tabId: string, data: string): void => {
    const inst = getInstance(tabId);
    if (!inst) return;
    inst.terminal.clear();
    inst.terminal.write(data);
  };

  const handlePtyMessage = (msg: { type: string; tabId: string; data?: string }): void => {
    if (!msg.data) return;
    if (msg.type === "pty") writePtyData(msg.tabId, msg.data);
    else if (msg.type === "pty-replay") replayPtyData(msg.tabId, msg.data);
  };

  const handleCommandComplete = (tabId: string): void => {
    const isActive = settings().chimeEnabled && tabId === activeTabId() && activePaneId() === props.paneId;
    if (isActive) playChime();
  };

  onMount(() => {
    onMessage((data) => {
      const msg = data as { type: string; tabId?: string; data?: string };
      if (!msg.tabId || !paneOwnsTab(msg.tabId)) return;
      if (msg.type === "command-complete") {
        handleCommandComplete(msg.tabId);
      } else {
        handlePtyMessage(msg as { type: string; tabId: string; data?: string });
      }
    });
  });

  const hideAllTerminals = (): void => {
    if (!containerRef) return;
    for (const child of Array.from(containerRef.children)) {
      (child as HTMLElement).style.display = "none";
    }
  };

  const ensureAttached = (el: HTMLElement | undefined): void => {
    if (!el || !containerRef || containerRef.contains(el)) return;
    containerRef.appendChild(el);
  };

  let activeResizeObserver: ResizeObserver | null = null;
  let activeResizeTimer: ReturnType<typeof setTimeout> | undefined;
  let lastCols = 0;
  let lastRows = 0;

  const attachResizeObserver = (tabId: string, instance: { fitAddon: { fit: () => void }; terminal: { cols: number; rows: number } }): void => {
    if (!containerRef) return;
    if (activeResizeObserver) {
      activeResizeObserver.disconnect();
    }
    clearTimeout(activeResizeTimer);
    lastCols = instance.terminal.cols;
    lastRows = instance.terminal.rows;
    activeResizeObserver = new ResizeObserver(() => {
      clearTimeout(activeResizeTimer);
      activeResizeTimer = setTimeout(() => {
        instance.fitAddon.fit();
        if (instance.terminal.cols === lastCols && instance.terminal.rows === lastRows) return;
        lastCols = instance.terminal.cols;
        lastRows = instance.terminal.rows;
        send({ type: "resize", tabId, cols: lastCols, rows: lastRows });
      }, 100);
    });
    activeResizeObserver.observe(containerRef);
  };

  const showExistingTerminal = (tabId: string): boolean => {
    const existing = getInstance(tabId);
    if (!existing || !containerRef) return false;
    ensureAttached(existing.terminal.element);
    if (existing.terminal.element) existing.terminal.element.style.display = "";
    attachResizeObserver(tabId, existing);
    requestAnimationFrame(() => {
      existing.fitAddon.fit();
      existing.terminal.focus();
    });
    return true;
  };

  const createNewTerminal = (tabId: string): void => {
    if (!containerRef) return;
    const container = containerRef;
    const instance = untrack(() => createInstance(
      tabId,
      container,
      { theme: settings().theme, fontSize: settings().fontSize },
      send
    ));
    attachResizeObserver(tabId, instance);
    requestAnimationFrame(() => instance.terminal.focus());
  };

  createEffect(
    on(activeTabId, (tabId) => {
      if (!containerRef || !tabId) return;
      hideAllTerminals();
      if (!showExistingTerminal(tabId)) {
        createNewTerminal(tabId);
      }
    })
  );

  createEffect(
    on(
      () => settings().theme,
      (themeKey) => {
        const theme = themeKey === "light" ? lightTheme : darkTheme;
        const tabId = activeTabId();
        if (!tabId) return;
        const inst = getInstance(tabId);
        if (inst) inst.terminal.options.theme = theme;
      }
    )
  );

  createEffect(
    on(
      () => settings().fontSize,
      (fontKey) => {
        const size = FONT_SIZE_MAP[fontKey] ?? 14;
        const tabId = activeTabId();
        if (!tabId) return;
        const inst = getInstance(tabId);
        if (!inst) return;
        inst.terminal.options.fontSize = size;
        inst.fitAddon.fit();
        send({
          type: "resize",
          tabId,
          cols: inst.terminal.cols,
          rows: inst.terminal.rows
        });
      }
    )
  );

  onCleanup(() => {
    if (activeResizeObserver) {
      activeResizeObserver.disconnect();
      activeResizeObserver = null;
    }
    clearTimeout(activeResizeTimer);
  });

  const focusTerminal = (e: MouseEvent): void => {
    const tabId = activeTabId();
    if (!tabId) return;
    const target = e.target as HTMLElement;
    if (target.closest(".xterm")) return;
    getInstance(tabId)?.terminal.focus();
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

  const isSplitAllowed = (sourcePaneId: string): boolean =>
    sourcePaneId !== props.paneId || props.pane.tabs.length > 1;

  const processDrop = (zone: DropZone, sourcePaneId: string, tabId: string): void => {
    const split = zoneToSplit(zone);
    if (split) {
      if (isSplitAllowed(sourcePaneId)) {
        splitPaneWithTab(props.paneId, split.direction, split.insertBefore, sourcePaneId, tabId);
      }
      return;
    }
    if (zone === "center" && sourcePaneId !== props.paneId) {
      moveTab(sourcePaneId, tabId, props.paneId);
    }
  };

  const handleDrop = (e: DragEvent): void => {
    e.preventDefault();
    e.stopPropagation();
    const zone = dropZone();
    setDropZone(null);
    if (!e.dataTransfer) return;
    const raw = e.dataTransfer.getData("application/x-tether-tab");
    if (!raw) return;
    const { sourcePaneId, tabId } = JSON.parse(raw) as { sourcePaneId: string; tabId: string };
    processDrop(zone, sourcePaneId, tabId);
  };

  const ZONE_CLASSES: Record<string, string> = {
    left: "inset-y-0 left-0 w-1/2",
    right: "inset-y-0 right-0 w-1/2",
    top: "inset-x-0 top-0 h-1/2",
    bottom: "inset-x-0 bottom-0 h-1/2",
    center: "inset-0"
  };

  const overlayClasses = (): string => {
    const zone = dropZone();
    const base = "absolute pointer-events-none rounded-sm";
    if (!zone) return `${base} inset-0 opacity-0`;
    return `${base} ${ZONE_CLASSES[zone] ?? "inset-0"} bg-primary/20 border-2 border-primary border-dashed`;
  };

  return (
    <div
      class="relative flex-1 min-h-0 min-w-0 overflow-hidden"
      data-testid={`pane-terminal-${props.paneId}`}
      onClick={focusTerminal}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <div ref={containerRef} class="absolute inset-0" />
      <div class={overlayClasses()} />
    </div>
  );
};
