import { Component, createSignal, createEffect } from "solid-js";
import { useSettings } from "../context/settings.js";
import { useConnection } from "../context/connection.js";

export const ConnectScreen: Component = () => {
  const { settings, updateSettings } = useSettings();
  const { connect, connected } = useConnection();
  const [connecting, setConnecting] = createSignal(false);
  const [error, setError] = createSignal("");

  createEffect(() => {
    if (connecting() && !connected()) {
      const timeout = setTimeout(() => {
        setConnecting(false);
        setError("Unable to connect. Check that the server is running and the IP/port are correct.");
      }, 5000);
      return () => clearTimeout(timeout);
    }
  });

  const handleConnect = (): void => {
    setError("");
    setConnecting(true);
    connect();
  };

  const toggleTheme = (): void => {
    const next = settings().theme === "dark" ? "light" : "dark";
    updateSettings({ theme: next });
  };

  return (
    <div
      class="flex flex-col h-full bg-canvas"
      data-testid="connect-screen"
    >
      <header class="flex items-center justify-end h-14 px-4 shrink-0">
        <button
          class="bg-transparent border-none text-ink text-lg cursor-pointer p-2 rounded-md hover:bg-muted"
          onClick={toggleTheme}
          aria-label={`Switch to ${settings().theme === "dark" ? "light" : "dark"} mode`}
          data-testid="connect-screen-theme-toggle"
        >
          {settings().theme === "dark" ? "☀️" : "🌙"}
        </button>
      </header>

      <main class="flex-1 flex items-center justify-center px-6 pb-16">
        <section class="w-full max-w-md bg-surface-raised border border-edge rounded-xl p-8 shadow-lg">
          <div class="text-center mb-8">
            <span class="text-5xl block mb-4" aria-hidden="true">&#127908;</span>
            <h1 class="text-2xl font-semibold text-ink">Walkie Talkie</h1>
            <p class="text-sm text-ink-secondary mt-2">
              Connect to your Mac to get started
            </p>
          </div>

          <fieldset class="flex flex-col gap-4 border-none p-0 m-0 mb-6">
            <label class="flex flex-col gap-1.5 text-sm text-ink-secondary font-medium">
              Mac IP
              <input
                class="w-full bg-canvas border border-edge-strong text-ink p-3 rounded-md text-[15px] outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim transition-colors"
                type="text"
                placeholder="192.168.1.100"
                value={settings().host}
                onInput={(e) => updateSettings({ host: e.currentTarget.value })}
                data-testid="connect-screen-host-input"
              />
            </label>
            <label class="flex flex-col gap-1.5 text-sm text-ink-secondary font-medium">
              Port
              <input
                class="w-full bg-canvas border border-edge-strong text-ink p-3 rounded-md text-[15px] outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim transition-colors"
                type="number"
                placeholder="5100"
                value={settings().port}
                onInput={(e) =>
                  updateSettings({ port: Number(e.currentTarget.value) })
                }
                data-testid="connect-screen-port-input"
              />
            </label>
          </fieldset>

          {error() && (
            <div
              class="mb-4 p-3 bg-error-subtle border border-error-200 rounded-md text-sm text-error"
              role="alert"
            >
              <p class="font-medium mb-1">Connection failed</p>
              <p class="text-error/80">{error()}</p>
            </div>
          )}

          <button
            class="w-full py-3.5 bg-primary border-none text-on-primary text-base font-semibold rounded-md cursor-pointer hover:bg-primary-hover active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed transition-all"
            onClick={handleConnect}
            disabled={connecting()}
            data-testid="connect-screen-connect-button"
          >
            {connecting() ? "Connecting..." : error() ? "Retry" : "Connect"}
          </button>
        </section>
      </main>
    </div>
  );
};
