import { Component, Show, onMount, onCleanup, createSignal } from "solid-js";
import { ConnectionProvider, useConnection } from "./context/connection.js";
import { SettingsProvider } from "./context/settings.js";
import { PanesProvider, usePanes } from "./context/panes.js";
import { StatusBar } from "./components/StatusBar.js";
import { SplitContainer } from "./components/SplitContainer.js";
import { TerminalToolbar } from "./components/TerminalToolbar.js";
import { ConnectScreen } from "./components/ConnectScreen.js";
import { GlobalSearch } from "./components/GlobalSearch.js";
import { ToastLayer } from "./components/ToastLayer.js";
import { destroyInstance } from "./lib/terminal-instances.js";

const Main: Component = () => {
  const { connected, onMessage } = useConnection();
  const { closeTab, getRoot } = usePanes();
  const [viewHeight, setViewHeight] = createSignal(window.innerHeight);
  const [searchOpen, setSearchOpen] = createSignal(false);

  onMount(() => {
    const vv = window.visualViewport;
    if (vv) {
      const onResize = (): void => {
        setViewHeight(vv.height);
      };
      vv.addEventListener("resize", onResize);
      onCleanup(() => vv.removeEventListener("resize", onResize));
    }

    const handleKeyDown = (e: KeyboardEvent): void => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setSearchOpen(true);
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    onCleanup(() => document.removeEventListener("keydown", handleKeyDown));

    onMessage((data) => {
      const msg = data as { type: string; tabId?: string };
      if (msg.type === "tab-exited" && msg.tabId) {
        destroyInstance(msg.tabId);
        const leaves = getAllLeavesFromRoot();
        for (const leaf of leaves) {
          if (leaf.tabs.some((t) => t.id === msg.tabId)) {
            closeTab(leaf.id, msg.tabId!);
            break;
          }
        }
      }
    });
  });

  const getAllLeavesFromRoot = () => {
    const result: Array<{ id: string; tabs: Array<{ id: string }> }> = [];
    const collect = (node: ReturnType<typeof getRoot>): void => {
      if (node.type === "leaf") {
        result.push(node);
      } else {
        collect(node.children[0]);
        collect(node.children[1]);
      }
    };
    collect(getRoot());
    return result;
  };

  return (
    <Show when={connected()} fallback={<ConnectScreen />}>
      <div
        class="flex flex-col bg-canvas text-ink"
        style={{ height: `${viewHeight()}px` }}
        data-testid="tether-main"
      >
        <StatusBar onSearchOpen={() => setSearchOpen(true)} />
        <div class="flex-1 flex min-h-0 min-w-0 overflow-hidden">
          <SplitContainer node={getRoot()} />
        </div>
        <TerminalToolbar />
        <GlobalSearch
          open={searchOpen()}
          onClose={() => setSearchOpen(false)}
        />
      </div>
    </Show>
  );
};

export const App: Component = () => {
  return (
    <SettingsProvider>
      <PanesProvider>
        <ConnectionProvider>
          <Main />
          <ToastLayer />
        </ConnectionProvider>
      </PanesProvider>
    </SettingsProvider>
  );
};
