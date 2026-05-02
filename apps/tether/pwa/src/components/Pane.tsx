import { Component } from "solid-js";
import { usePanes, LeafNode } from "../context/panes.js";
import { PaneTabBar } from "./PaneTabBar.js";
import { PaneTerminal } from "./PaneTerminal.js";

interface PaneProps {
  pane: LeafNode;
}

export const Pane: Component<PaneProps> = (props) => {
  const { activePaneId, setActivePane } = usePanes();

  const isActive = () => activePaneId() === props.pane.id;

  const handleClick = (): void => {
    setActivePane(props.pane.id);
  };

  return (
    <div
      class="relative flex flex-col flex-1 min-h-0 min-w-0"
      onClick={handleClick}
      data-testid={`pane-${props.pane.id}`}
    >
      <div
        class={`absolute inset-0 pointer-events-none z-10 ${
          isActive()
            ? "border-2 border-primary"
            : "border border-edge"
        }`}
      />
      <PaneTabBar paneId={props.pane.id} pane={props.pane} />
      <PaneTerminal paneId={props.pane.id} pane={props.pane} />
    </div>
  );
};
