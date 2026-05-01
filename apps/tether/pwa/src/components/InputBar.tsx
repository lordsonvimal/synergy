import { Component, createSignal, Show } from "solid-js";
import { MicButton } from "./MicButton.js";

interface InputBarProps {
  onSend: (text: string) => void;
}

export const InputBar: Component<InputBarProps> = (props) => {
  const [textMode, setTextMode] = createSignal(false);
  const [inputText, setInputText] = createSignal("");
  let inputRef: HTMLInputElement | undefined;

  const handleSendText = (): void => {
    const text = inputText().trim();
    if (!text) {
      return;
    }
    props.onSend(text);
    setInputText("");
  };

  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSendText();
    }
  };

  const handleVoiceSend = (text: string): void => {
    props.onSend(text);
  };

  return (
    <footer
      class="shrink-0 px-4 py-3 border-t border-edge bg-surface"
      data-testid="input-bar"
    >
      <Show
        when={textMode()}
        fallback={
          <div class="flex items-center gap-3">
            <button
              class="bg-muted border border-edge text-ink px-4 py-2.5 rounded-md text-sm cursor-pointer hover:bg-surface-raised transition-colors"
              onClick={() => {
                setTextMode(true);
                requestAnimationFrame(() => inputRef?.focus());
              }}
              aria-label="Switch to text input"
            >
              &#9000; Type
            </button>
            <MicButton onSend={handleVoiceSend} />
          </div>
        }
      >
        <div class="flex items-center gap-3">
          <button
            class="bg-muted border border-edge text-ink px-3 py-2.5 rounded-md text-lg cursor-pointer hover:bg-surface-raised transition-colors"
            onClick={() => setTextMode(false)}
            aria-label="Switch to voice input"
          >
            &#127908;
          </button>
          <input
            ref={inputRef}
            class="flex-1 bg-canvas border border-edge-strong text-ink px-4 py-3 rounded-md text-[15px] outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim transition-colors"
            type="text"
            placeholder="Type your prompt..."
            value={inputText()}
            onInput={e => setInputText(e.currentTarget.value)}
            onKeyDown={handleKeyDown}
            autocomplete="off"
            autocorrect="off"
            autocapitalize="off"
            spellcheck={false}
            data-testid="input-bar-text-field"
          />
          <button
            class="bg-primary border-none text-on-primary w-11 h-11 rounded-md text-lg cursor-pointer hover:bg-primary-hover active:scale-95 transition-all"
            onClick={handleSendText}
            aria-label="Send message"
            data-testid="input-bar-send-button"
          >
            &#10148;
          </button>
        </div>
      </Show>
    </footer>
  );
};
