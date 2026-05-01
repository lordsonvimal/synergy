import { Component, createSignal, Show } from "solid-js";
import { useConnection } from "../context/connection.js";
import { useChat } from "../context/chat.js";
import { MicButton } from "./MicButton.js";

export const InputBar: Component = () => {
  const { send } = useConnection();
  const { addMessage } = useChat();
  const [textMode, setTextMode] = createSignal(false);
  const [inputText, setInputText] = createSignal("");

  const handleSendText = (): void => {
    const text = inputText().trim();
    if (!text) {
      return;
    }
    addMessage("user", text);
    send({ type: "text", data: text });
    setInputText("");
  };

  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSendText();
    }
  };

  const handleVoiceSend = (text: string): void => {
    addMessage("user", text);
    send({ type: "text", data: text });
  };

  return (
    <footer class="px-4 py-3 border-t border-border bg-bg">
      <Show
        when={textMode()}
        fallback={
          <div class="flex items-center gap-3">
            <button
              class="bg-surface border border-border text-text-primary px-4 py-2.5 rounded-lg text-sm cursor-pointer"
              onClick={() => setTextMode(true)}
            >
              &#9000; Type
            </button>
            <MicButton onSend={handleVoiceSend} />
          </div>
        }
      >
        <div class="flex items-center gap-3">
          <button
            class="bg-surface border border-border text-text-primary px-3 py-2.5 rounded-lg text-lg cursor-pointer"
            onClick={() => setTextMode(false)}
          >
            &#127908;
          </button>
          <input
            class="flex-1 bg-surface border border-border text-text-primary px-4 py-3 rounded-lg text-[15px] outline-none focus:border-accent"
            type="text"
            placeholder="Type your prompt..."
            value={inputText()}
            onInput={(e) => setInputText(e.currentTarget.value)}
            onKeyDown={handleKeyDown}
          />
          <button
            class="bg-accent border-none text-white w-11 h-11 rounded-lg text-lg cursor-pointer"
            onClick={handleSendText}
          >
            &#10148;
          </button>
        </div>
      </Show>
    </footer>
  );
};
