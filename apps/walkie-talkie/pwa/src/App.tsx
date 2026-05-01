import { Component, Show, onMount, onCleanup, createSignal } from "solid-js";
import { ConnectionProvider, useConnection } from "./context/connection.js";
import { SettingsProvider } from "./context/settings.js";
import { StatusBar } from "./components/StatusBar.js";
import { Terminal } from "./components/Terminal.js";
import { InputBar } from "./components/InputBar.js";
import { ConnectScreen } from "./components/ConnectScreen.js";

const Main: Component = () => {
  const { connected, send } = useConnection();
  const [viewHeight, setViewHeight] = createSignal(window.innerHeight);

  const handleSend = (text: string): void => {
    send({ type: "text", data: text });
  };

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
        <InputBar onSend={handleSend} />
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
