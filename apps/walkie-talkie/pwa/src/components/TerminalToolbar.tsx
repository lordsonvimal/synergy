import { Component, createSignal, Show } from "solid-js";
import { useConnection } from "../context/connection.js";
import { VoiceInput } from "./VoiceInput.js";

export const TerminalToolbar: Component = () => {
  const { send } = useConnection();
  const [keysOpen, setKeysOpen] = createSignal(false);
  const [isReviewing, setIsReviewing] = createSignal(false);

  const sendKey = (data: string): void => {
    send({ type: "key", data });
  };

  const handleVoiceSend = (text: string): void => {
    send({ type: "text", data: text });
  };

  const keys = [
    { label: "Enter", data: "\r" },
    { label: "Tab", data: "\t" },
    { label: "Esc", data: "\x1b" },
    { label: "↑", data: "\x1b[A" },
    { label: "↓", data: "\x1b[B" },
    { label: "←", data: "\x1b[D" },
    { label: "→", data: "\x1b[C" },
    { label: "Ctrl+C", data: "\x03" },
  ];

  return (
    <div
      class="shrink-0 flex flex-col gap-1.5 px-3 py-2 bg-surface border-t border-edge"
      data-testid="terminal-toolbar"
    >
      <Show when={keysOpen() && !isReviewing()}>
        <div class="flex items-center gap-1.5 overflow-x-auto">
          {keys.map(k => (
            <button
              class="bg-muted border border-edge text-ink px-3 py-1.5 rounded-md text-xs font-mono cursor-pointer hover:bg-surface-raised active:scale-95 transition-all whitespace-nowrap"
              onPointerDown={e => {
                e.preventDefault();
                sendKey(k.data);
              }}
            >
              {k.label}
            </button>
          ))}
        </div>
      </Show>
      <div class="flex items-center gap-2 w-full">
        <button
          class={`shrink-0 border w-10 h-10 rounded-md cursor-pointer transition-colors flex items-center justify-center ${
            keysOpen()
              ? "bg-primary-subtle border-primary text-primary"
              : "bg-muted border-edge text-ink hover:bg-surface-raised"
          }`}
          onClick={() => setKeysOpen(!keysOpen())}
          aria-label={keysOpen() ? "Hide keys" : "Show keys"}
          data-testid="toolbar-keys-toggle"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="4" width="20" height="16" rx="2" />
            <path d="M6 8h.01M10 8h.01M14 8h.01M18 8h.01M6 12h.01M10 12h.01M14 12h.01M18 12h.01M8 16h8" />
          </svg>
        </button>
        <div class="flex-1 flex items-center justify-end">
          <VoiceInput
            onSend={handleVoiceSend}
            onReviewChange={setIsReviewing}
          />
        </div>
      </div>
    </div>
  );
};
