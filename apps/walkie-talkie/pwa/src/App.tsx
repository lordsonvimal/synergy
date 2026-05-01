import { Component, Show, onMount, onCleanup, createSignal } from "solid-js";
import { ConnectionProvider, useConnection } from "./context/connection.js";
import { SettingsProvider } from "./context/settings.js";
import { StatusBar } from "./components/StatusBar.js";
import { Terminal } from "./components/Terminal.js";
import { TerminalToolbar } from "./components/TerminalToolbar.js";
import { ConnectScreen } from "./components/ConnectScreen.js";

const Main: Component = () => {
  const { connected } = useConnection();
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
  });

  return (
    <Show when={connected()} fallback={<ConnectScreen />}>
      <div
        class="flex flex-col bg-canvas text-ink"
        style={{ height: `${viewHeight()}px` }}
        data-testid="walkie-talkie-main"
      >
        <StatusBar />
        <Terminal />
        <TerminalToolbar />
      </div>
    </Show>
  );
};

export const App: Component = () => {
  return (
    <SettingsProvider>
      <ConnectionProvider>
        <Main />
      </ConnectionProvider>
    </SettingsProvider>
  );
};
