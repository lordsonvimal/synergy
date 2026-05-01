import { Component, Show } from "solid-js";
import { ConnectionProvider, useConnection } from "./context/connection.js";
import { ChatProvider } from "./context/chat.js";
import { SettingsProvider } from "./context/settings.js";
import { StatusBar } from "./components/StatusBar.js";
import { ChatArea } from "./components/ChatArea.js";
import { InputBar } from "./components/InputBar.js";
import { ConnectScreen } from "./components/ConnectScreen.js";

const Main: Component = () => {
  const { connected } = useConnection();

  return (
    <Show when={connected()} fallback={<ConnectScreen />}>
      <div class="main-layout">
        <StatusBar />
        <ChatArea />
        <InputBar />
      </div>
    </Show>
  );
};

export const App: Component = () => {
  return (
    <SettingsProvider>
      <ConnectionProvider>
        <ChatProvider>
          <Main />
        </ChatProvider>
      </ConnectionProvider>
    </SettingsProvider>
  );
};
