import { Component, Show, onMount, onCleanup, createSignal } from "solid-js";
import { ConnectionProvider, useConnection } from "./context/connection.js";
import { SettingsProvider } from "./context/settings.js";
import { TabsProvider, useTabs } from "./context/tabs.js";
import { StatusBar } from "./components/StatusBar.js";
import { TabBar } from "./components/TabBar.js";
import { Terminal, destroyTabTerminal } from "./components/Terminal.js";
import { TerminalToolbar } from "./components/TerminalToolbar.js";
import { ConnectScreen } from "./components/ConnectScreen.js";
import { ToastLayer } from "./components/ToastLayer.js";

const Main: Component = () => {
  const { connected, onMessage } = useConnection();
  const { closeTab } = useTabs();
  const [viewHeight, setViewHeight] = createSignal(window.innerHeight);

  onMount(() => {
    const vv = window.visualViewport;
    if (vv) {
      const onResize = (): void => {
        setViewHeight(vv.height);
      };
      vv.addEventListener("resize", onResize);
      onCleanup(() => vv.removeEventListener("resize", onResize));
    }

    onMessage((data) => {
      const msg = data as { type: string; tabId?: string };
      if (msg.type === "tab-exited" && msg.tabId) {
        destroyTabTerminal(msg.tabId);
        closeTab(msg.tabId);
      }
    });
  });

  return (
    <Show when={connected()} fallback={<ConnectScreen />}>
      <div
        class="flex flex-col bg-canvas text-ink"
        style={{ height: `${viewHeight()}px` }}
        data-testid="tether-main"
      >
        <StatusBar />
        <TabBar />
        <Terminal />
        <TerminalToolbar />
      </div>
    </Show>
  );
};

export const App: Component = () => {
  return (
    <SettingsProvider>
      <TabsProvider>
        <ConnectionProvider>
          <Main />
          <ToastLayer />
        </ConnectionProvider>
      </TabsProvider>
    </SettingsProvider>
  );
};
