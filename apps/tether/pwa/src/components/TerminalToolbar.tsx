import { Component, createSignal } from "solid-js";
import { useConnection } from "../context/connection.js";
import { usePanes } from "../context/panes.js";
import { VoiceInput } from "./VoiceInput.js";
import { KeysOverlay } from "./KeysOverlay.js";
import { ShortcutPanel } from "./ShortcutPanel.js";

function toggleButtonClass(active: boolean): string {
  if (active) {
    return "shrink-0 border w-10 h-10 rounded-md cursor-pointer transition-all flex items-center justify-center bg-primary-subtle border-primary text-primary hover:bg-primary-100";
  }
  return "shrink-0 border w-10 h-10 rounded-md cursor-pointer transition-all flex items-center justify-center bg-muted border-edge text-ink hover:bg-surface-raised hover:border-edge-strong";
}

export const TerminalToolbar: Component = () => {
  const { send } = useConnection();
  const { activeTabId } = usePanes();
  const [keysOpen, setKeysOpen] = createSignal(false);
  const [shortcutsOpen, setShortcutsOpen] = createSignal(false);
  const [isReviewing, setIsReviewing] = createSignal(false);

  const handleVoiceSend = (text: string): void => {
    send({ type: "text", tabId: activeTabId(), data: text });
  };

  const handleShortcutSend = (command: string): void => {
    send({ type: "text", tabId: activeTabId(), data: command });
  };

  return (
    <div
      class="shrink-0 px-3 py-2 bg-surface border-t border-edge"
      data-testid="terminal-toolbar"
    >
      <KeysOverlay
        open={keysOpen() && !isReviewing()}
        onClose={() => setKeysOpen(false)}
      />
      <ShortcutPanel
        open={shortcutsOpen() && !isReviewing()}
        onClose={() => setShortcutsOpen(false)}
        onSend={handleShortcutSend}
      />
      <div class="flex items-center gap-2 w-full">
        <button
          class={toggleButtonClass(keysOpen())}
          onClick={() => setKeysOpen(!keysOpen())}
          aria-label={keysOpen() ? "Hide keys" : "Show keys"}
          data-testid="toolbar-keys-toggle"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="4" width="20" height="16" rx="2" />
            <path d="M6 8h.01M10 8h.01M14 8h.01M18 8h.01M6 12h.01M10 12h.01M14 12h.01M18 12h.01M8 16h8" />
          </svg>
        </button>
        <button
          class={toggleButtonClass(shortcutsOpen())}
          onClick={() => setShortcutsOpen(!shortcutsOpen())}
          aria-label={shortcutsOpen() ? "Hide shortcuts" : "Show shortcuts"}
          data-testid="toolbar-shortcuts-toggle"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M13 2 L3 14h9l-1 8 10-12h-9l1-8z" />
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
