import { Component } from "solid-js";
import { usePanes, LeafNode } from "../context/panes.js";
import { PaneTabBar } from "./PaneTabBar.js";
import { PaneTerminal } from "./PaneTerminal.js";

interface PaneProps {
  pane: LeafNode;
}

export const Pane: Component<PaneProps> = (props) => {
  const { setActivePane } = usePanes();

  const handleClick = (): void => {
    setActivePane(props.pane.id);
  };

  return (
    <div
      class="flex flex-col flex-1 min-h-0 min-w-0 border border-edge"
      onClick={handleClick}
      data-testid={`pane-${props.pane.id}`}
    >
      <PaneTabBar paneId={props.pane.id} pane={props.pane} />
      <PaneTerminal paneId={props.pane.id} pane={props.pane} />
    </div>
  );
};
