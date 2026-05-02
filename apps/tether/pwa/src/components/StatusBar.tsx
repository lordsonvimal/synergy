import { Component, createSignal, onMount, Show } from "solid-js";
import { useConnection } from "../context/connection.js";
import { SettingsPanel } from "./SettingsPanel.js";

interface StatusBarProps {
  onSearchOpen: () => void;
}

export const StatusBar: Component<StatusBarProps> = (props) => {
  const { connected, onMessage } = useConnection();
  const [battery, setBattery] = createSignal<number | null>(null);
  const [charging, setCharging] = createSignal(false);
  const [settingsOpen, setSettingsOpen] = createSignal(false);

  onMount(() => {
    onMessage((data) => {
      const msg = data as { type: string; level?: number; charging?: boolean };
      if (msg.type === "battery") {
        setBattery(msg.level ?? null);
        setCharging(msg.charging ?? false);
      }
    });
  });

  return (
    <>
      <header
        class="flex items-center h-14 px-4 bg-surface border-b border-edge sticky top-0 z-3"
        data-testid="status-bar"
      >
        <button
          class="bg-transparent border-none text-ink text-xl cursor-pointer p-2 rounded-md hover:bg-muted transition-colors"
          aria-label="Open settings"
          onClick={() => setSettingsOpen(true)}
          data-testid="status-bar-menu"
        >
          &#9776;
        </button>
        <span class="text-sm font-semibold text-ink ml-2">
          Tether
        </span>
        <span class="flex-1" />
        <button
          class="bg-transparent border-none text-ink-secondary cursor-pointer p-2 rounded-md hover:bg-muted hover:text-ink transition-colors"
          aria-label="Search tabs"
          onClick={props.onSearchOpen}
          data-testid="status-bar-search"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
        </button>
        <span
          class={`w-2.5 h-2.5 rounded-full ${
            connected() ? "bg-success" : "bg-error"
          }`}
          role="status"
          aria-label={connected() ? "Connected" : "Disconnected"}
        />
        <Show when={battery() !== null}>
          <span class="text-xs text-ink-secondary ml-3">
            {charging() ? "⚡" : "🔋"} {battery()}%
          </span>
        </Show>
      </header>
      <SettingsPanel
        open={settingsOpen()}
        onClose={() => setSettingsOpen(false)}
      />
    </>
  );
};
