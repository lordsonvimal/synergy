import { Component, Show, createSignal } from "solid-js";
import { usePanes, LeafNode, SplitDirection } from "../context/panes.js";
import { PaneTabBar } from "./PaneTabBar.js";
import { PaneTerminal } from "./PaneTerminal.js";

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

interface PaneProps {
  pane: LeafNode;
}

export const Pane: Component<PaneProps> = (props) => {
  const { activePaneId, setActivePane, splitPaneWithTab, moveTab } = usePanes();
  const [dropZone, setDropZone] = createSignal<DropZone>(null);

  let containerRef: HTMLDivElement | undefined;

  const isActive = () => activePaneId() === props.pane.id;

  const handleClick = (): void => {
    setActivePane(props.pane.id);
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
        props.pane.id,
        split.direction,
        split.insertBefore,
        sourcePaneId,
        tabId
      );
    } else if (zone === "center" && sourcePaneId !== props.pane.id) {
      moveTab(sourcePaneId, tabId, props.pane.id);
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
      class={`relative flex flex-col flex-1 min-h-0 min-w-0 ${
        isActive()
          ? "ring-2 ring-primary ring-inset"
          : "ring-1 ring-edge ring-inset"
      }`}
      onClick={handleClick}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      data-testid={`pane-${props.pane.id}`}
    >
      <PaneTabBar paneId={props.pane.id} pane={props.pane} />
      <PaneTerminal paneId={props.pane.id} pane={props.pane} />
      <Show when={dropZone()}>
        <div class={overlayClasses()} />
      </Show>
    </div>
  );
};
