import { Component, createSignal, onMount, Show } from "solid-js";
import { useConnection } from "../context/connection.js";
import { getBatteryStatus } from "../lib/battery.js";

export const StatusBar: Component = () => {
  const { connected } = useConnection();
  const [battery, setBattery] = createSignal<number | null>(null);

  onMount(async () => {
    const status = await getBatteryStatus();
    if (status.supported) {
      setBattery(status.level);
    }
  });

  return (
    <header class="flex items-center h-11 px-4 bg-bg border-b border-border sticky top-0 z-10">
      <button class="bg-transparent border-none text-text-primary text-xl cursor-pointer px-2 py-1">
        &#9776;
      </button>
      <span class="flex-1 text-center text-sm font-semibold text-text-primary">
        Walkie Talkie
      </span>
      <span
        class={`w-2 h-2 rounded-full ml-2 ${
          connected() ? "bg-connected" : "bg-error"
        }`}
      />
      <Show when={battery() !== null}>
        <span class="text-xs text-text-secondary ml-3">
          {battery()}%
        </span>
      </Show>
    </header>
  );
};
