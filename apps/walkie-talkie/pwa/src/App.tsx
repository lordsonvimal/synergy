import { Component, Show, onMount, createSignal } from "solid-js";
import { ConnectionProvider, useConnection } from "./context/connection.js";
import { ChatProvider, useChat } from "./context/chat.js";
import { SettingsProvider } from "./context/settings.js";
import { StatusBar } from "./components/StatusBar.js";
import { ChatArea } from "./components/ChatArea.js";
import { InputBar } from "./components/InputBar.js";
import { ConnectScreen } from "./components/ConnectScreen.js";

interface OutputMessage {
  type: "output";
  text: string;
  streaming: boolean;
}

interface OutputCompleteMessage {
  type: "output_complete";
  fullText: string;
}

interface PermissionMessage {
  type: "permission";
  action: string;
  options: string[];
}

interface StatusMessage {
  type: "status";
  connected: boolean;
  claudeReady: boolean;
}

type ServerMessage =
  | OutputMessage
  | OutputCompleteMessage
  | PermissionMessage
  | StatusMessage;

const Main: Component = () => {
  const { connected, onMessage } = useConnection();
  const { addMessage, updateMessage, completeMessage } = useChat();
  const [activeMessageId, setActiveMessageId] = createSignal<string | null>(
    null
  );

  const buf = { text: "" };

  const handleBeforeSend = (): void => {
    const current = activeMessageId();
    if (current) {
      completeMessage(current);
      setActiveMessageId(null);
    }
    buf.text = "";
  };

  onMount(() => {
    onMessage((data) => {
      const msg = data as ServerMessage;

      switch (msg.type) {
        case "output": {
          const current = activeMessageId();
          if (current) {
            buf.text += msg.text;
            updateMessage(current, buf.text);
          } else {
            buf.text = msg.text;
            const id = addMessage("assistant", buf.text);
            setActiveMessageId(id);
          }
          break;
        }
        case "output_complete": {
          const current = activeMessageId();
          if (current) {
            updateMessage(current, msg.fullText);
            completeMessage(current);
          }
          buf.text = "";
          setActiveMessageId(null);
          break;
        }
        case "permission": {
          const current = activeMessageId();
          if (current) {
            completeMessage(current);
            setActiveMessageId(null);
          }
          buf.text = "";
          addMessage(
            "assistant",
            `⚠️ Permission: ${msg.action}\n\nOptions: ${msg.options.join(", ")}`
          );
          break;
        }
        case "status":
          break;
      }
    });
  });

  return (
    <Show when={connected()} fallback={<ConnectScreen />}>
      <div class="flex flex-col h-screen bg-bg text-text-primary">
        <StatusBar />
        <ChatArea />
        <InputBar onBeforeSend={handleBeforeSend} />
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
