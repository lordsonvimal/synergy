import { Component, createSignal } from "solid-js";
import { usePanes, SplitDirection } from "../context/panes.js";

interface DividerProps {
  branchId: string;
  direction: SplitDirection;
}

export const Divider: Component<DividerProps> = (props) => {
  const { setRatio } = usePanes();
  const [dragging, setDragging] = createSignal(false);

  let containerRef: HTMLDivElement | undefined;

  const handlePointerDown = (e: PointerEvent): void => {
    e.preventDefault();
    setDragging(true);
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
  };

  const handlePointerMove = (e: PointerEvent): void => {
    if (!dragging()) return;
    const parent = containerRef?.parentElement;
    if (!parent) return;

    const rect = parent.getBoundingClientRect();
    let ratio: number;

    if (props.direction === "horizontal") {
      ratio = (e.clientX - rect.left) / rect.width;
    } else {
      ratio = (e.clientY - rect.top) / rect.height;
    }

    setRatio(props.branchId, ratio);
  };

  const handlePointerUp = (): void => {
    setDragging(false);
  };

  const isHorizontal = () => props.direction === "horizontal";

  return (
    <div
      ref={containerRef}
      class={`shrink-0 flex items-center justify-center touch-none select-none transition-colors ${
        isHorizontal()
          ? "w-1 cursor-col-resize hover:bg-primary/30"
          : "h-1 cursor-row-resize hover:bg-primary/30"
      } ${dragging() ? "bg-primary/40" : "bg-edge"}`}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
      data-testid={`divider-${props.branchId}`}
    />
  );
};
