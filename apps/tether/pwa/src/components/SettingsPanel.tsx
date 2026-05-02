import { Component, Show, For, createSignal, createEffect } from "solid-js";
import { Portal } from "solid-js/web";
import { useSettings, Shortcut } from "../context/settings.js";
import { useConnection } from "../context/connection.js";

interface SettingsPanelProps {
  open: boolean;
  onClose: () => void;
}

const FONT_SIZE_OPTIONS: { value: "small" | "medium" | "large"; label: string }[] = [
  { value: "small", label: "Small" },
  { value: "medium", label: "Medium" },
  { value: "large", label: "Large" }
];

export const SettingsPanel: Component<SettingsPanelProps> = props => {
  const { settings, updateSettings } = useSettings();
  const { disconnect } = useConnection();
  const [mounted, setMounted] = createSignal(false);
  const [closing, setClosing] = createSignal(false);

  createEffect(() => {
    if (props.open) {
      setMounted(true);
      setClosing(false);
    } else if (mounted()) {
      setClosing(true);
    }
  });

  const handleAnimationEnd = (): void => {
    if (closing()) {
      setMounted(false);
      setClosing(false);
    }
  };

  const toggleTheme = (): void => {
    const next = settings().theme === "dark" ? "light" : "dark";
    updateSettings({ theme: next });
  };

  const toggleChime = (): void => {
    updateSettings({ chimeEnabled: !settings().chimeEnabled });
  };

  const [editingShortcut, setEditingShortcut] = createSignal<string | null>(null);
  const [editLabel, setEditLabel] = createSignal("");
  const [editCommand, setEditCommand] = createSignal("");
  const [addingNew, setAddingNew] = createSignal(false);

  const startEdit = (shortcut: Shortcut): void => {
    setEditingShortcut(shortcut.id);
    setEditLabel(shortcut.label);
    setEditCommand(shortcut.command);
    setAddingNew(false);
  };

  const startAdd = (): void => {
    setEditingShortcut(null);
    setEditLabel("");
    setEditCommand("");
    setAddingNew(true);
  };

  const cancelEdit = (): void => {
    setEditingShortcut(null);
    setAddingNew(false);
  };

  const saveEdit = (): void => {
    const label = editLabel().trim();
    const command = editCommand().trim();
    if (!label || !command) return;

    const shortcuts = [...settings().shortcuts];
    if (addingNew()) {
      shortcuts.push({
        id: `s${Date.now()}`,
        label,
        command
      });
    } else {
      const idx = shortcuts.findIndex(s => s.id === editingShortcut());
      if (idx !== -1) {
        shortcuts[idx] = { ...shortcuts[idx]!, label, command };
      }
    }
    updateSettings({ shortcuts });
    setEditingShortcut(null);
    setAddingNew(false);
  };

  const deleteShortcut = (id: string): void => {
    const shortcuts = settings().shortcuts.filter(s => s.id !== id);
    updateSettings({ shortcuts });
    if (editingShortcut() === id) {
      setEditingShortcut(null);
    }
  };

  const moveShortcut = (id: string, direction: -1 | 1): void => {
    const shortcuts = [...settings().shortcuts];
    const idx = shortcuts.findIndex(s => s.id === id);
    const target = idx + direction;
    if (idx === -1 || target < 0 || target >= shortcuts.length) return;
    [shortcuts[idx], shortcuts[target]] = [shortcuts[target]!, shortcuts[idx]!];
    updateSettings({ shortcuts });
  };

  return (
    <Show when={mounted()}>
      <Portal mount={document.getElementById("settings-layer")!}>
        <div
          class="fixed inset-0"
          data-testid="settings-panel-backdrop"
        >
          <div
            class={`absolute inset-0 bg-canvas/60 transition-opacity duration-200 ${
              closing() ? "opacity-0" : "opacity-100"
            }`}
            onClick={props.onClose}
          />
          <aside
            class={`absolute top-0 left-0 bottom-0 w-72 bg-surface border-r border-edge shadow-xl flex flex-col ${
              closing() ? "animate-slide-out-left" : "animate-slide-in-left"
            }`}
            role="dialog"
            aria-label="Settings"
            data-testid="settings-panel"
            onAnimationEnd={handleAnimationEnd}
          >
            <header class="flex items-center justify-between h-14 px-4 border-b border-edge shrink-0">
              <h2 class="text-base font-semibold text-ink">Settings</h2>
              <button
                class="bg-transparent border-none text-ink text-xl cursor-pointer p-2 rounded-md hover:bg-muted transition-colors"
                onClick={props.onClose}
                aria-label="Close settings"
                data-testid="settings-panel-close"
              >
                &times;
              </button>
            </header>

            <div class="flex-1 overflow-y-auto p-4 flex flex-col gap-6">
              <section>
                <h3 class="text-xs font-semibold text-ink-secondary uppercase tracking-wide mb-3">
                  Appearance
                </h3>
                <div class="flex flex-col gap-4">
                  <div class="flex items-center justify-between">
                    <label class="text-sm text-ink" for="settings-theme">
                      Theme
                    </label>
                    <button
                      id="settings-theme"
                      class="flex items-center gap-2 px-3 py-1.5 bg-muted border border-edge rounded-md text-sm text-ink cursor-pointer hover:bg-surface-raised transition-colors"
                      onClick={toggleTheme}
                      data-testid="settings-theme-toggle"
                    >
                      {settings().theme === "dark" ? "☀️ Light" : "🌙 Dark"}
                    </button>
                  </div>

                  <div class="flex items-center justify-between">
                    <label class="text-sm text-ink" id="settings-font-label">
                      Font size
                    </label>
                    <div
                      class="flex border border-edge rounded-md overflow-hidden"
                      role="radiogroup"
                      aria-labelledby="settings-font-label"
                    >
                      {FONT_SIZE_OPTIONS.map(opt => (
                        <button
                          class={`px-3 py-1.5 text-xs border-none cursor-pointer transition-colors ${
                            settings().fontSize === opt.value
                              ? "bg-primary text-on-primary"
                              : "bg-muted text-ink hover:bg-surface-raised"
                          }`}
                          role="radio"
                          aria-checked={settings().fontSize === opt.value}
                          onClick={() => updateSettings({ fontSize: opt.value })}
                          data-testid={`settings-font-${opt.value}`}
                        >
                          {opt.label}
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              </section>

              <section>
                <h3 class="text-xs font-semibold text-ink-secondary uppercase tracking-wide mb-3">
                  Notifications
                </h3>
                <div class="flex items-center justify-between">
                  <div class="flex flex-col">
                    <label class="text-sm text-ink" for="settings-chime">
                      Completion chime
                    </label>
                    <span class="text-xs text-ink-dim mt-0.5">
                      Play sound when terminal goes idle
                    </span>
                  </div>
                  <button
                    id="settings-chime"
                    class={`relative inline-flex items-center w-11 h-6 rounded-full cursor-pointer transition-colors shrink-0 ${
                      settings().chimeEnabled
                        ? "bg-primary"
                        : "bg-muted border border-edge"
                    }`}
                    role="switch"
                    aria-checked={settings().chimeEnabled}
                    onClick={toggleChime}
                    data-testid="settings-chime-toggle"
                  >
                    <span
                      class="inline-block w-5 h-5 rounded-full bg-white shadow-sm transition-transform"
                      style={{
                        transform: settings().chimeEnabled
                          ? "translateX(22px)"
                          : "translateX(2px)"
                      }}
                    />
                  </button>
                </div>
              </section>

              <section>
                <div class="flex items-center justify-between mb-3">
                  <h3 class="text-xs font-semibold text-ink-secondary uppercase tracking-wide">
                    Shortcuts
                  </h3>
                  <button
                    class="text-xs text-primary cursor-pointer hover:text-primary-hover transition-colors bg-transparent border-none"
                    onClick={startAdd}
                    data-testid="settings-shortcut-add"
                  >
                    + Add
                  </button>
                </div>

                <Show when={addingNew()}>
                  <div class="flex flex-col gap-2 mb-3 p-2 bg-muted rounded-md border border-edge">
                    <input
                      type="text"
                      class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-2 py-1 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25"
                      placeholder="Label"
                      value={editLabel()}
                      onInput={e => setEditLabel(e.currentTarget.value)}
                      autofocus
                      data-testid="settings-shortcut-new-label"
                    />
                    <input
                      type="text"
                      class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-2 py-1 font-mono outline-none focus:border-primary focus:ring-2 focus:ring-primary/25"
                      placeholder="Command"
                      value={editCommand()}
                      onInput={e => setEditCommand(e.currentTarget.value)}
                      data-testid="settings-shortcut-new-command"
                    />
                    <div class="flex gap-2">
                      <button
                        class="flex-1 py-1 text-xs bg-primary text-on-primary border-none rounded-md cursor-pointer hover:bg-primary-hover transition-colors"
                        onClick={saveEdit}
                        data-testid="settings-shortcut-new-save"
                      >
                        Save
                      </button>
                      <button
                        class="flex-1 py-1 text-xs bg-muted text-ink border border-edge rounded-md cursor-pointer hover:bg-surface-raised transition-colors"
                        onClick={cancelEdit}
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                </Show>

                <div class="flex flex-col gap-1">
                  <For each={settings().shortcuts}>
                    {(shortcut, idx) => (
                      <Show
                        when={editingShortcut() === shortcut.id}
                        fallback={
                          <div class="flex items-center gap-1 group">
                            <div class="flex flex-col shrink-0">
                              <button
                                class="bg-transparent border-none text-ink-dim text-[10px] leading-none cursor-pointer hover:text-ink p-0 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                                onClick={() => moveShortcut(shortcut.id, -1)}
                                disabled={idx() === 0}
                                aria-label="Move up"
                              >
                                &#9650;
                              </button>
                              <button
                                class="bg-transparent border-none text-ink-dim text-[10px] leading-none cursor-pointer hover:text-ink p-0 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                                onClick={() => moveShortcut(shortcut.id, 1)}
                                disabled={idx() === settings().shortcuts.length - 1}
                                aria-label="Move down"
                              >
                                &#9660;
                              </button>
                            </div>
                            <button
                              class="flex-1 text-left bg-muted border border-edge rounded-md px-2 py-1.5 cursor-pointer hover:bg-surface-raised transition-colors"
                              onClick={() => startEdit(shortcut)}
                              data-testid={`settings-shortcut-${shortcut.id}`}
                            >
                              <span class="text-sm text-ink block">{shortcut.label}</span>
                              <span class="text-xs text-ink-dim font-mono block">{shortcut.command}</span>
                            </button>
                            <button
                              class="bg-transparent border-none text-ink-dim text-sm cursor-pointer hover:text-error p-1 rounded-md hover:bg-muted transition-colors shrink-0"
                              onClick={() => deleteShortcut(shortcut.id)}
                              aria-label={`Delete ${shortcut.label}`}
                              data-testid={`settings-shortcut-delete-${shortcut.id}`}
                            >
                              &times;
                            </button>
                          </div>
                        }
                      >
                        <div class="flex flex-col gap-2 p-2 bg-muted rounded-md border border-primary">
                          <input
                            type="text"
                            class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-2 py-1 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25"
                            value={editLabel()}
                            onInput={e => setEditLabel(e.currentTarget.value)}
                            autofocus
                          />
                          <input
                            type="text"
                            class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-2 py-1 font-mono outline-none focus:border-primary focus:ring-2 focus:ring-primary/25"
                            value={editCommand()}
                            onInput={e => setEditCommand(e.currentTarget.value)}
                          />
                          <div class="flex gap-2">
                            <button
                              class="flex-1 py-1 text-xs bg-primary text-on-primary border-none rounded-md cursor-pointer hover:bg-primary-hover transition-colors"
                              onClick={saveEdit}
                            >
                              Save
                            </button>
                            <button
                              class="flex-1 py-1 text-xs bg-muted text-ink border border-edge rounded-md cursor-pointer hover:bg-surface-raised transition-colors"
                              onClick={cancelEdit}
                            >
                              Cancel
                            </button>
                          </div>
                        </div>
                      </Show>
                    )}
                  </For>
                </div>
              </section>

              <section>
                <h3 class="text-xs font-semibold text-ink-secondary uppercase tracking-wide mb-3">
                  Connection
                </h3>
                <div class="flex flex-col gap-3">
                  <div class="flex flex-col gap-1">
                    <span class="text-sm text-ink">
                      {settings().host}:{settings().port}
                    </span>
                    <span class="text-xs text-ink-dim">
                      Disconnect to change connection settings
                    </span>
                  </div>
                  <button
                    class="w-full py-2 px-3 bg-error-subtle border border-error/30 text-error text-sm font-medium rounded-md cursor-pointer hover:bg-error hover:text-on-primary transition-colors"
                    onClick={() => {
                      disconnect();
                      props.onClose();
                    }}
                    data-testid="settings-disconnect"
                  >
                    Disconnect
                  </button>
                </div>
              </section>
            </div>
          </aside>
        </div>
      </Portal>
    </Show>
  );
};
