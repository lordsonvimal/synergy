import { Component, createSignal, onMount, Show } from "solid-js";
import { useConnection } from "../context/connection.js";
import { useSettings } from "../context/settings.js";
import { getBatteryStatus } from "../lib/battery.js";

export const StatusBar: Component = () => {
  const { connected } = useConnection();
  const { settings, updateSettings } = useSettings();
  const [battery, setBattery] = createSignal<number | null>(null);

  onMount(async () => {
    const status = await getBatteryStatus();
    if (status.supported) {
      setBattery(status.level);
    }
  });

  const toggleTheme = (): void => {
    const next = settings().theme === "dark" ? "light" : "dark";
    updateSettings({ theme: next });
  };

  return (
    <header
      class="flex items-center h-14 px-4 bg-surface border-b border-edge sticky top-0 z-3"
      data-testid="status-bar"
    >
      <button
        class="bg-transparent border-none text-ink text-xl cursor-pointer p-2 rounded-md hover:bg-muted"
        aria-label="Open menu"
      >
        &#9776;
      </button>
      <span class="flex-1 text-center text-sm font-semibold text-ink">
        Walkie Talkie
      </span>
      <button
        class="bg-transparent border-none text-ink text-lg cursor-pointer p-2 rounded-md hover:bg-muted"
        onClick={toggleTheme}
        aria-label={`Switch to ${settings().theme === "dark" ? "light" : "dark"} mode`}
        data-testid="status-bar-theme-toggle"
      >
        {settings().theme === "dark" ? "☀️" : "🌙"}
      </button>
      <span
        class={`w-2.5 h-2.5 rounded-full ml-2 ${
          connected() ? "bg-success" : "bg-error"
        }`}
        role="status"
        aria-label={connected() ? "Connected" : "Disconnected"}
      />
      <Show when={battery() !== null}>
        <span class="text-xs text-ink-secondary ml-3">
          {battery()}%
        </span>
      </Show>
    </header>
  );
};
