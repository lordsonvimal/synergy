import { Component, createSignal } from "solid-js";
import { useSettings } from "../context/settings.js";
import { useConnection } from "../context/connection.js";

export const ConnectScreen: Component = () => {
  const { settings, updateSettings } = useSettings();
  const { connect } = useConnection();
  const [connecting, setConnecting] = createSignal(false);

  const handleConnect = (): void => {
    setConnecting(true);
    connect();
  };

  return (
    <div class="flex flex-col items-center justify-center h-full p-6 gap-8">
      <div class="text-center">
        <span class="text-5xl block mb-3">&#127908;</span>
        <h1 class="text-2xl font-semibold text-text-primary">Walkie Talkie</h1>
      </div>
      <div class="w-full max-w-[300px] flex flex-col gap-4">
        <label class="flex flex-col gap-1 text-[13px] text-text-secondary">
          Mac IP
          <input
            class="bg-surface border border-border text-text-primary p-3 rounded-lg text-[15px] outline-none focus:border-accent"
            type="text"
            value={settings().host}
            onInput={(e) => updateSettings({ host: e.currentTarget.value })}
          />
        </label>
        <label class="flex flex-col gap-1 text-[13px] text-text-secondary">
          Port
          <input
            class="bg-surface border border-border text-text-primary p-3 rounded-lg text-[15px] outline-none focus:border-accent"
            type="number"
            value={settings().port}
            onInput={(e) =>
              updateSettings({ port: Number(e.currentTarget.value) })
            }
          />
        </label>
      </div>
      <button
        class="w-full max-w-[300px] py-3.5 bg-accent border-none text-white text-base font-semibold rounded-lg cursor-pointer hover:bg-accent-hover disabled:opacity-60 disabled:cursor-not-allowed"
        onClick={handleConnect}
        disabled={connecting()}
      >
        {connecting() ? "Connecting..." : "Connect"}
      </button>
    </div>
  );
};
