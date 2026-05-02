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
  // eslint-disable-next-line no-unassigned-vars -- SolidJS ref pattern
  let containerRef: HTMLDivElement | undefined;
  const { onMessage, send } = useConnection();
  const { settings } = useSettings();
  const { activePaneId, splitPaneWithTab, moveTab } = usePanes();
  const [dropZone, setDropZone] = createSignal<DropZone>(null);

  const activeTabId = () => props.pane.activeTabId;

  const paneTabIds = (): Set<string> =>
    new Set(props.pane.tabs.map((t) => t.id));

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
      if (!msg.tabId || !paneTabIds().has(msg.tabId)) return;
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
      const el = child as HTMLElement;
      if (el.dataset.dropOverlay !== undefined) continue;
      el.style.display = "none";
    }
  };

  const ensureAttached = (el: HTMLElement | undefined): void => {
    if (!el || !containerRef || containerRef.contains(el)) return;
    containerRef.appendChild(el);
  };

  const showExistingTerminal = (tabId: string): boolean => {
    const existing = getInstance(tabId);
    if (!existing || !containerRef) return false;
    ensureAttached(existing.terminal.element);
    if (existing.terminal.element) existing.terminal.element.style.display = "";
    requestAnimationFrame(() => {
      existing.fitAddon.fit();
      existing.terminal.focus();
    });
    return true;
  };

  const createNewTerminal = (tabId: string): void => {
    if (!containerRef) return;
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
        send({ type: "resize", tabId, cols: instance.terminal.cols, rows: instance.terminal.rows });
      }, 100);
    });
    resizeObserver.observe(containerRef);
    instance.resizeObserver = resizeObserver;
    instance.resizeTimer = resizeTimer;
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

  const processDrop = (zone: DropZone, sourcePaneId: string, tabId: string): void => {
    const split = zoneToSplit(zone);
    if (split) {
      splitPaneWithTab(props.paneId, split.direction, split.insertBefore, sourcePaneId, tabId);
    } else if (zone === "center" && sourcePaneId !== props.paneId) {
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
    if (!zone) return "hidden";
    const base = "absolute pointer-events-none bg-primary/20 border-2 border-primary border-dashed rounded-sm";
    return `${base} ${ZONE_CLASSES[zone] ?? "inset-0"}`;
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
