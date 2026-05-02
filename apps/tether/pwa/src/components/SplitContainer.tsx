import { Component, Show } from "solid-js";
import { usePanes, SplitNode } from "../context/panes.js";
import { Pane } from "./Pane.js";
import { Divider } from "./Divider.js";

interface SplitContainerProps {
  node: SplitNode;
}

export const SplitContainer: Component<SplitContainerProps> = (props) => {
  const { isNarrow } = usePanes();

  return (
    <Show
      when={props.node.type === "branch" ? props.node : undefined}
      fallback={
        <Pane pane={props.node as Extract<SplitNode, { type: "leaf" }>} />
      }
    >
      {(branch) => {
        const direction = () =>
          isNarrow() ? "vertical" : branch().direction;

        return (
          <div
            class={`flex flex-1 min-h-0 min-w-0 ${
              direction() === "horizontal" ? "flex-row" : "flex-col"
            }`}
            data-testid={`split-${branch().id}`}
          >
            <div
              class="flex min-h-0 min-w-0 overflow-hidden"
              style={{
                "flex": `0 0 ${branch().ratio * 100}%`
              }}
            >
              <SplitContainer node={branch().children[0]} />
            </div>
            <Divider
              branchId={branch().id}
              direction={direction()}
            />
            <div
              class="flex flex-1 min-h-0 min-w-0 overflow-hidden"
            >
              <SplitContainer node={branch().children[1]} />
            </div>
          </div>
        );
      }}
    </Show>
  );
};
