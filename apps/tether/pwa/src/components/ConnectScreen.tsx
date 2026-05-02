import { Component, createSignal, createEffect, Show } from "solid-js";
import { useSettings } from "../context/settings.js";
import { useConnection } from "../context/connection.js";
import { usePanes } from "../context/panes.js";

const INPUT_BASE = "w-full bg-canvas text-ink p-3 rounded-md text-[15px] outline-none placeholder:text-ink-dim transition-colors";
const INPUT_DEFAULT = `${INPUT_BASE} border border-edge-strong focus:border-primary focus:ring-2 focus:ring-primary/25`;
const INPUT_ERROR = `${INPUT_BASE} border border-error focus:ring-2 focus:ring-error/25`;

function validateHost(host: string, touched: boolean): string {
  if (!touched) return "";
  if (!host.trim()) return "IP address is required";
  return "";
}

function validatePort(port: number, touched: boolean): string {
  if (!touched) return "";
  if (!port) return "Port is required";
  if (port < 1 || port > 65535) return "Port must be between 1 and 65535";
  return "";
}

function getButtonLabel(connecting: boolean, hasError: boolean): string {
  if (connecting) return "Connecting...";
  if (hasError) return "Retry";
  return "Connect";
}

function themeIcon(theme: string): string {
  return theme === "dark" ? "☀️" : "🌙";
}

function themeLabel(theme: string): string {
  return theme === "dark" ? "light" : "dark";
}

function inputClass(hasError: boolean): string {
  return hasError ? INPUT_ERROR : INPUT_DEFAULT;
}

function modeButtonClass(active: boolean): string {
  const base = "flex-1 px-3 py-2.5 text-sm border-none cursor-pointer transition-colors";
  return active
    ? `${base} bg-primary text-on-primary`
    : `${base} bg-muted text-ink hover:bg-surface-raised`;
}

function modeDescription(mode: string): string {
  return mode === "mirror"
    ? "All devices share the same terminal sessions"
    : "Each device gets its own terminal sessions";
}

function useConnectionTimeout(
  connecting: () => boolean,
  connected: () => boolean,
  setConnecting: (v: boolean) => void,
  setError: (v: string) => void
): void {
  createEffect(() => {
    if (!connecting() || connected()) return;
    const timeout = setTimeout(() => {
      setConnecting(false);
      setError("Unable to connect. Check that the server is running and the IP/port are correct.");
    }, 5000);
    return () => clearTimeout(timeout);
  });
}

export const ConnectScreen: Component = () => {
  const { settings, updateSettings } = useSettings();
  const { connect, connected } = useConnection();
  const { getAllTabIds } = usePanes();
  const [connecting, setConnecting] = createSignal(false);
  const [error, setError] = createSignal("");
  const [hostTouched, setHostTouched] = createSignal(false);
  const [portTouched, setPortTouched] = createSignal(false);

  const hostError = (): string =>
    validateHost(settings().host, hostTouched());

  const portError = (): string =>
    validatePort(settings().port, portTouched());

  useConnectionTimeout(connecting, connected, setConnecting, setError);

  const handleSubmit = (e: SubmitEvent): void => {
    e.preventDefault();
    setHostTouched(true);
    setPortTouched(true);
    if (hostError() || portError()) return;
    setError("");
    setConnecting(true);
    connect(getAllTabIds);
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
          aria-label={`Switch to ${themeLabel(settings().theme)} mode`}
          data-testid="connect-screen-theme-toggle"
        >
          {themeIcon(settings().theme)}
        </button>
      </header>

      <main class="flex-1 flex items-center justify-center px-6 pb-16">
        <section class="w-full max-w-md bg-surface-raised border border-edge rounded-xl p-8 shadow-lg">
          <div class="text-center mb-8">
            <svg class="w-20 h-20 mx-auto mb-4" viewBox="0 0 80 80" fill="none" aria-hidden="true">
              <circle cx="40" cy="40" r="36" stroke="currentColor" stroke-width="2.5" class="text-ink" opacity="0.15" />
              <path d="M22 34 L32 40 L22 46" stroke="currentColor" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" class="text-ink" />
              <rect x="36" y="44" width="12" height="3.5" rx="1.5" fill="currentColor" class="text-ink" />
              <circle cx="56" cy="34" r="3.5" fill="currentColor" class="text-primary" />
              <circle cx="56" cy="34" r="7.5" stroke="currentColor" stroke-width="2" class="text-primary" opacity="0.5" />
              <circle cx="56" cy="34" r="12" stroke="currentColor" stroke-width="1.5" class="text-primary" opacity="0.25" />
            </svg>
            <h1 class="text-2xl font-semibold text-ink">Tether</h1>
            <p class="text-sm text-ink-secondary mt-2">
              Connect to your Mac to get started
            </p>
          </div>

          <form onSubmit={handleSubmit} data-testid="connect-screen-form">
            <fieldset class="flex flex-col gap-4 border-none p-0 m-0 mb-6">
              <label class="flex flex-col gap-1.5 text-sm text-ink-secondary font-medium">
                <span>Mac IP <span class="text-error">*</span></span>
                <input
                  class={inputClass(!!hostError())}
                  type="text"
                  placeholder="192.168.1.100"
                  value={settings().host}
                  onInput={(e) => updateSettings({ host: e.currentTarget.value })}
                  onBlur={() => setHostTouched(true)}
                  required
                  data-testid="connect-screen-host-input"
                />
                <Show when={hostError()}>
                  <span class="text-sm text-error mt-1">{hostError()}</span>
                </Show>
              </label>
              <label class="flex flex-col gap-1.5 text-sm text-ink-secondary font-medium">
                <span>Port <span class="text-error">*</span></span>
                <input
                  class={inputClass(!!portError())}
                  type="number"
                  placeholder="5100"
                  value={settings().port}
                  onInput={(e) =>
                    updateSettings({ port: Number(e.currentTarget.value) })
                  }
                  onBlur={() => setPortTouched(true)}
                  required
                  data-testid="connect-screen-port-input"
                />
                <Show when={portError()}>
                  <span class="text-sm text-error mt-1">{portError()}</span>
                </Show>
              </label>
              <label class="flex flex-col gap-1.5 text-sm text-ink-secondary font-medium">
                <span>Secret</span>
                <input
                  class={INPUT_DEFAULT}
                  type="password"
                  placeholder="Leave blank if not set"
                  value={settings().secret}
                  onInput={(e) => updateSettings({ secret: e.currentTarget.value })}
                  autocomplete="off"
                  data-testid="connect-screen-secret-input"
                />
                <span class="text-xs text-ink-dim">
                  Must match TETHER_SECRET on the server
                </span>
              </label>
              <div class="flex flex-col gap-1.5 text-sm text-ink-secondary font-medium">
                <span>Mode</span>
                <div
                  class="flex border border-edge rounded-md overflow-hidden"
                  role="radiogroup"
                  aria-label="Connection mode"
                >
                  <button
                    type="button"
                    class={modeButtonClass(settings().mode === "independent")}
                    role="radio"
                    aria-checked={settings().mode === "independent"}
                    onClick={() => updateSettings({ mode: "independent" })}
                    data-testid="connect-screen-mode-independent"
                  >
                    Independent
                  </button>
                  <button
                    type="button"
                    class={modeButtonClass(settings().mode === "mirror")}
                    role="radio"
                    aria-checked={settings().mode === "mirror"}
                    onClick={() => updateSettings({ mode: "mirror" })}
                    data-testid="connect-screen-mode-mirror"
                  >
                    Mirror
                  </button>
                </div>
                <span class="text-xs text-ink-dim">
                  {modeDescription(settings().mode)}
                </span>
              </div>
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
              type="submit"
              class="w-full py-3.5 bg-primary border-none text-on-primary text-base font-semibold rounded-md cursor-pointer hover:bg-primary-hover active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed transition-all"
              disabled={connecting()}
              data-testid="connect-screen-connect-button"
            >
              {getButtonLabel(connecting(), !!error())}
            </button>
          </form>
        </section>
      </main>
    </div>
  );
};
