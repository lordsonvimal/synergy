import { Component, Show, For, createSignal, createEffect } from "solid-js";
import { Portal } from "solid-js/web";
import { useSettings, Shortcut } from "../context/settings.js";
import { useConnection } from "../context/connection.js";
import {
  FONT_SIZE_OPTIONS,
  reorderShortcuts,
  flipTheme,
  themeButtonLabel,
  fontSizeButtonClass,
  toggleBgClass,
  toggleTranslate,
  closingClass,
  modeLabel
} from "./settings-panel-utils.js";

interface SettingsPanelProps {
  open: boolean;
  onClose: () => void;
}

function useAnimatedMount(open: () => boolean): {
  mounted: () => boolean;
  closing: () => boolean;
  handleAnimationEnd: () => void;
} {
  const [mounted, setMounted] = createSignal(false);
  const [closing, setClosing] = createSignal(false);
  createEffect(() => {
    if (open()) { setMounted(true); setClosing(false); }
    else if (mounted()) { setClosing(true); }
  });
  const handleAnimationEnd = (): void => {
    if (closing()) { setMounted(false); setClosing(false); }
  };
  return { mounted, closing, handleAnimationEnd };
}

function useShortcutEditor(settings: () => ReturnType<ReturnType<typeof useSettings>["settings"]>, updateSettings: ReturnType<typeof useSettings>["updateSettings"]): {
  editingShortcut: () => string | null;
  editLabel: () => string;
  editCommand: () => string;
  addingNew: () => boolean;
  setEditLabel: (v: string) => void;
  setEditCommand: (v: string) => void;
  startEdit: (shortcut: Shortcut) => void;
  startAdd: () => void;
  cancelEdit: () => void;
  saveEdit: () => void;
  deleteShortcut: (id: string) => void;
} {
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

  const buildUpdatedShortcuts = (label: string, command: string): Shortcut[] => {
    const shortcuts = [...settings().shortcuts];
    if (addingNew()) {
      shortcuts.push({ id: `s${Date.now()}`, label, command });
    } else {
      const idx = shortcuts.findIndex(s => s.id === editingShortcut());
      const existing = shortcuts[idx];
      if (idx !== -1 && existing) shortcuts[idx] = { ...existing, label, command };
    }
    return shortcuts;
  };

  const saveEdit = (): void => {
    const label = editLabel().trim();
    const command = editCommand().trim();
    if (!label || !command) return;
    updateSettings({ shortcuts: buildUpdatedShortcuts(label, command) });
    setEditingShortcut(null);
    setAddingNew(false);
  };

  const deleteShortcut = (id: string): void => {
    updateSettings({ shortcuts: settings().shortcuts.filter(s => s.id !== id) });
    if (editingShortcut() === id) setEditingShortcut(null);
  };

  return { editingShortcut, editLabel, editCommand, addingNew, setEditLabel, setEditCommand, startEdit, startAdd, cancelEdit, saveEdit, deleteShortcut };
}

function useShortcutDrag(
  settings: () => ReturnType<ReturnType<typeof useSettings>["settings"]>,
  updateSettings: ReturnType<typeof useSettings>["updateSettings"],
  isEditing: () => boolean
): {
  dragId: () => string | null;
  dragY: () => number;
  dragLabel: () => string;
  dragCommand: () => string;
  dropIndex: () => number;
  handleDragStart: (e: PointerEvent, id: string) => void;
  itemClass: (id: string) => string;
  itemStyle: (id: string) => Record<string, string>;
  listRef: (el: HTMLDivElement) => void;
} {
  const [dragId, setDragId] = createSignal<string | null>(null);
  const [dragY, setDragY] = createSignal(0);
  const [dragLabel, setDragLabel] = createSignal("");
  const [dragCommand, setDragCommand] = createSignal("");
  const [dropIndex, setDropIndex] = createSignal(-1);
  let listEl: HTMLDivElement | undefined;
  let pointerId = -1;
  let itemOffsetY = 0;
  let itemHeight = 0;

  const listRef = (el: HTMLDivElement): void => { listEl = el; };

  const computeDropIndex = (clientY: number): number => {
    if (!listEl) return -1;
    const listRect = listEl.getBoundingClientRect();
    if (clientY < listRect.top || clientY > listRect.bottom) return -1;
    const items = listEl.querySelectorAll<HTMLElement>("[data-shortcut-id]");
    const currentId = dragId();
    let idx = 0;
    for (const item of items) {
      const id = item.getAttribute("data-shortcut-id");
      if (id === currentId) continue;
      const rect = item.getBoundingClientRect();
      if (clientY > rect.top + rect.height / 2) idx++;
    }
    return idx;
  };

  const handlePointerMove = (e: PointerEvent): void => {
    setDragY(e.clientY - itemOffsetY);
    setDropIndex(computeDropIndex(e.clientY));
  };

  const cleanup = (): void => {
    document.removeEventListener("pointermove", handlePointerMove);
    document.removeEventListener("pointerup", handlePointerUp);
    document.removeEventListener("pointercancel", handlePointerUp);
    document.removeEventListener("keydown", handleKeyDown);
    if (listEl && pointerId >= 0) {
      try { listEl.releasePointerCapture(pointerId); } catch {}
    }
    pointerId = -1;
  };

  const handlePointerUp = (): void => {
    const sourceId = dragId();
    const idx = dropIndex();
    cleanup();
    setDragId(null);
    setDragY(0);
    setDropIndex(-1);
    if (!sourceId || idx === -1) return;
    const shortcuts = settings().shortcuts;
    const sourceIdx = shortcuts.findIndex(s => s.id === sourceId);
    if (sourceIdx === -1) return;
    const adjustedIdx = idx > sourceIdx ? idx : idx;
    if (adjustedIdx === sourceIdx) return;
    const overId = adjustedIdx === 0
      ? "__before_first__"
      : shortcuts.filter(s => s.id !== sourceId)[adjustedIdx - 1]?.id ?? null;
    const result = reorderShortcuts(shortcuts, sourceId, overId);
    if (result) updateSettings({ shortcuts: result });
  };

  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Escape") {
      cleanup();
      setDragId(null);
      setDragY(0);
      setDropIndex(-1);
    }
  };

  const handleDragStart = (e: PointerEvent, id: string): void => {
    if (isEditing()) return;
    e.preventDefault();
    pointerId = e.pointerId;
    if (listEl) listEl.setPointerCapture(e.pointerId);
    const row = (e.currentTarget as HTMLElement).closest("[data-shortcut-id]") as HTMLElement | null;
    if (row) {
      const rect = row.getBoundingClientRect();
      itemOffsetY = e.clientY - rect.top;
      itemHeight = rect.height;
      setDragY(rect.top);
    }
    const shortcut = settings().shortcuts.find(s => s.id === id);
    setDragLabel(shortcut?.label ?? "");
    setDragCommand(shortcut?.command ?? "");
    setDragId(id);
    setDropIndex(-1);
    document.addEventListener("pointermove", handlePointerMove);
    document.addEventListener("pointerup", handlePointerUp);
    document.addEventListener("pointercancel", handlePointerUp);
    document.addEventListener("keydown", handleKeyDown);
  };

  const sourceIndex = (): number => {
    const id = dragId();
    if (!id) return -1;
    return settings().shortcuts.findIndex(s => s.id === id);
  };

  const itemClass = (id: string): string => {
    if (!dragId()) return "";
    if (dragId() === id) return "opacity-0 h-0 overflow-hidden m-0 p-0";
    return "";
  };

  const itemStyle = (id: string): Record<string, string> => {
    const currentDrag = dragId();
    if (!currentDrag || currentDrag === id) return {};
    const idx = dropIndex();
    const src = sourceIndex();
    if (idx === -1 || src === -1) return {};
    const shortcuts = settings().shortcuts;
    const myIdx = shortcuts.findIndex(s => s.id === id);
    if (myIdx === -1) return {};
    const gap = itemHeight + 4;
    if (src < idx) {
      if (myIdx > src && myIdx <= idx) {
        return { transform: `translateY(-${gap}px)`, transition: "transform 150ms ease" };
      }
    } else {
      if (myIdx >= idx && myIdx < src) {
        return { transform: `translateY(${gap}px)`, transition: "transform 150ms ease" };
      }
    }
    return { transition: "transform 150ms ease" };
  };

  return { dragId, dragY, dragLabel, dragCommand, dropIndex, handleDragStart, itemClass, itemStyle, listRef };
}

export const SettingsPanel: Component<SettingsPanelProps> = props => {
  const { settings, updateSettings } = useSettings();
  const { disconnect } = useConnection();
  const { mounted, closing, handleAnimationEnd } = useAnimatedMount(() => props.open);
  const editor = useShortcutEditor(settings, updateSettings);
  const drag = useShortcutDrag(settings, updateSettings, () => !!editor.editingShortcut() || editor.addingNew());

  const toggleTheme = (): void => updateSettings({ theme: flipTheme(settings().theme) });
  const toggleChime = (): void => updateSettings({ chimeEnabled: !settings().chimeEnabled });

  return (
    <Show when={mounted()}>
      <Portal mount={document.getElementById("settings-layer") as HTMLElement}>
        <div
          class="fixed inset-0"
          data-testid="settings-panel-backdrop"
        >
          <div
            class={`absolute inset-0 bg-canvas/60 transition-opacity duration-200 ${
              closingClass(closing(), "opacity-100", "opacity-0")
            }`}
            onClick={props.onClose}
          />
          <aside
            class={`absolute top-0 left-0 bottom-0 w-full sm:w-80 bg-surface border-r border-edge shadow-xl flex flex-col ${
              closingClass(closing(), "animate-slide-in-left", "animate-slide-out-left")
            }`}
            role="dialog"
            aria-label="Settings"
            data-testid="settings-panel"
            onAnimationEnd={handleAnimationEnd}
          >
            <header class="flex items-center justify-between h-14 px-5 border-b border-edge shrink-0">
              <h2 class="text-base font-semibold text-ink">Settings</h2>
              <button
                class="flex items-center justify-center w-8 h-8 bg-transparent border-none text-ink text-xl cursor-pointer rounded-md hover:bg-muted transition-colors"
                onClick={props.onClose}
                aria-label="Close settings"
                data-testid="settings-panel-close"
              >
                &times;
              </button>
            </header>

            <div class="flex-1 overflow-y-auto px-5 py-5 flex flex-col gap-0">
              {/* Appearance */}
              <section class="pb-6 mb-6 border-b border-edge">
                <h3 class="text-xs font-semibold text-ink-secondary uppercase tracking-wide mb-4">
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
                      {themeButtonLabel(settings().theme)}
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
                          class={fontSizeButtonClass(settings().fontSize === opt.value)}
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

              {/* Notifications */}
              <section class="pb-6 mb-6 border-b border-edge">
                <h3 class="text-xs font-semibold text-ink-secondary uppercase tracking-wide mb-4">
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
                      toggleBgClass(settings().chimeEnabled)
                    }`}
                    role="switch"
                    aria-checked={settings().chimeEnabled}
                    onClick={toggleChime}
                    data-testid="settings-chime-toggle"
                  >
                    <span
                      class="inline-block w-5 h-5 rounded-full bg-white shadow-sm transition-transform"
                      style={{
                        transform: toggleTranslate(settings().chimeEnabled)
                      }}
                    />
                  </button>
                </div>
              </section>

              {/* Shortcuts */}
              <section class="pb-6 mb-6 border-b border-edge">
                <div class="flex items-center justify-between mb-4">
                  <h3 class="text-xs font-semibold text-ink-secondary uppercase tracking-wide">
                    Shortcuts
                  </h3>
                  <button
                    class="text-xs text-primary cursor-pointer hover:text-primary-hover transition-colors bg-transparent border-none font-medium"
                    onClick={editor.startAdd}
                    data-testid="settings-shortcut-add"
                  >
                    + Add
                  </button>
                </div>

                <Show when={editor.addingNew()}>
                  <div class="flex flex-col gap-2 mb-3 p-3 bg-muted rounded-lg border border-edge">
                    <input
                      type="text"
                      class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-2.5 py-1.5 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim"
                      placeholder="Label"
                      value={editor.editLabel()}
                      onInput={e => editor.setEditLabel(e.currentTarget.value)}
                      autofocus
                      data-testid="settings-shortcut-new-label"
                    />
                    <input
                      type="text"
                      class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-2.5 py-1.5 font-mono outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim"
                      placeholder="Command"
                      value={editor.editCommand()}
                      onInput={e => editor.setEditCommand(e.currentTarget.value)}
                      data-testid="settings-shortcut-new-command"
                    />
                    <div class="flex gap-2">
                      <button
                        class="flex-1 py-1.5 text-xs bg-primary text-on-primary border-none rounded-md cursor-pointer hover:bg-primary-hover transition-colors font-medium"
                        onClick={editor.saveEdit}
                        data-testid="settings-shortcut-new-save"
                      >
                        Save
                      </button>
                      <button
                        class="flex-1 py-1.5 text-xs bg-surface text-ink border border-edge rounded-md cursor-pointer hover:bg-muted transition-colors"
                        onClick={editor.cancelEdit}
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                </Show>

                <Show
                  when={settings().shortcuts.length > 0}
                  fallback={
                    <div class="flex flex-col items-center py-6 text-center">
                      <span class="text-ink-dim text-sm mb-2">No shortcuts yet</span>
                      <button
                        class="text-xs text-primary cursor-pointer hover:text-primary-hover transition-colors bg-transparent border-none font-medium"
                        onClick={editor.startAdd}
                      >
                        + Add your first shortcut
                      </button>
                    </div>
                  }
                >
                  <div class="flex flex-col gap-1 max-h-52 overflow-y-auto" data-shortcut-list ref={drag.listRef}>
                    <For each={settings().shortcuts}>
                      {(shortcut) => (
                        <Show
                          when={editor.editingShortcut() === shortcut.id}
                          fallback={
                            <div
                              class={`flex items-center gap-1 rounded-md ${
                                drag.itemClass(shortcut.id)
                              }`}
                              style={drag.itemStyle(shortcut.id)}
                              data-shortcut-id={shortcut.id}
                            >
                              <div
                                class="flex items-center justify-center w-5 cursor-grab text-ink-dim hover:text-ink active:cursor-grabbing shrink-0 touch-none"
                                onPointerDown={e => drag.handleDragStart(e, shortcut.id)}
                              >
                                <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor">
                                  <circle cx="9" cy="6" r="1.5" />
                                  <circle cx="15" cy="6" r="1.5" />
                                  <circle cx="9" cy="12" r="1.5" />
                                  <circle cx="15" cy="12" r="1.5" />
                                  <circle cx="9" cy="18" r="1.5" />
                                  <circle cx="15" cy="18" r="1.5" />
                                </svg>
                              </div>
                              <button
                                class="flex-1 text-left bg-muted border border-edge rounded-md px-2.5 py-1.5 cursor-pointer hover:bg-surface-raised transition-colors min-w-0"
                                onClick={() => editor.startEdit(shortcut)}
                                data-testid={`settings-shortcut-${shortcut.id}`}
                              >
                                <span class="text-xs text-ink block font-medium truncate">{shortcut.label}</span>
                                <span class="text-[11px] text-ink-dim font-mono block truncate">{shortcut.command}</span>
                              </button>
                              <button
                                class="flex items-center justify-center w-6 h-6 bg-transparent border-none text-ink-dim text-xs cursor-pointer hover:text-error rounded-md hover:bg-muted transition-colors shrink-0"
                                onClick={() => editor.deleteShortcut(shortcut.id)}
                                aria-label={`Delete ${shortcut.label}`}
                                data-testid={`settings-shortcut-delete-${shortcut.id}`}
                              >
                                &times;
                              </button>
                            </div>
                          }
                        >
                          <div
                            class="flex flex-col gap-2 p-2.5 bg-muted rounded-md border border-primary"
                            data-shortcut-id={shortcut.id}
                          >
                            <input
                              type="text"
                              class="w-full bg-canvas border border-edge-strong rounded-md text-xs text-ink px-2 py-1 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25"
                              value={editor.editLabel()}
                              onInput={e => editor.setEditLabel(e.currentTarget.value)}
                              autofocus
                            />
                            <input
                              type="text"
                              class="w-full bg-canvas border border-edge-strong rounded-md text-xs text-ink px-2 py-1 font-mono outline-none focus:border-primary focus:ring-2 focus:ring-primary/25"
                              value={editor.editCommand()}
                              onInput={e => editor.setEditCommand(e.currentTarget.value)}
                            />
                            <div class="flex gap-2">
                              <button
                                class="flex-1 py-1 text-xs bg-primary text-on-primary border-none rounded-md cursor-pointer hover:bg-primary-hover transition-colors font-medium"
                                onClick={editor.saveEdit}
                              >
                                Save
                              </button>
                              <button
                                class="flex-1 py-1 text-xs bg-surface text-ink border border-edge rounded-md cursor-pointer hover:bg-muted transition-colors"
                                onClick={editor.cancelEdit}
                              >
                                Cancel
                              </button>
                            </div>
                          </div>
                        </Show>
                      )}
                    </For>
                  </div>
                </Show>
              </section>

              {/* Connection info */}
              <section>
                <h3 class="text-xs font-semibold text-ink-secondary uppercase tracking-wide mb-4">
                  Connection
                </h3>
                <div class="flex flex-col gap-4">
                  <div class="flex items-center gap-2">
                    <span class="w-2 h-2 rounded-full bg-success shrink-0" />
                    <span class="text-sm text-ink font-mono">
                      {settings().host}:{settings().port}
                    </span>
                  </div>
                  <div class="flex items-center justify-between">
                    <div class="flex flex-col">
                      <label class="text-sm text-ink">Mode</label>
                      <span class="text-xs text-ink-dim mt-0.5">
                        {modeLabel(settings().mode)}
                      </span>
                    </div>
                    <span class="text-sm text-ink-secondary font-medium capitalize">
                      {settings().mode}
                    </span>
                  </div>
                </div>
              </section>
            </div>

            {/* Sticky footer */}
            <footer class="shrink-0 px-5 py-4 border-t border-edge">
              <button
                class="w-full py-2.5 px-3 bg-error-subtle border border-error/30 text-error text-sm font-medium rounded-md cursor-pointer hover:bg-error hover:text-on-primary transition-colors"
                onClick={() => {
                  disconnect();
                  props.onClose();
                }}
                data-testid="settings-disconnect"
              >
                Disconnect
              </button>
              <p class="text-center text-[11px] text-ink-dim mt-3">
                Tether v1.0
              </p>
            </footer>
          </aside>
          <Show when={drag.dragId()}>
            <div
              class="fixed left-5 pointer-events-none w-[calc(min(100vw,20rem)-2.5rem)] opacity-80"
              style={{ top: `${drag.dragY()}px`, transform: "rotate(1.5deg)" }}
            >
              <div class="flex items-center gap-1 bg-surface-raised border border-primary rounded-md shadow-xl px-2.5 py-1.5">
                <div class="flex items-center justify-center w-5 text-ink-dim shrink-0">
                  <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor">
                    <circle cx="9" cy="6" r="1.5" />
                    <circle cx="15" cy="6" r="1.5" />
                    <circle cx="9" cy="12" r="1.5" />
                    <circle cx="15" cy="12" r="1.5" />
                    <circle cx="9" cy="18" r="1.5" />
                    <circle cx="15" cy="18" r="1.5" />
                  </svg>
                </div>
                <div class="flex-1 min-w-0">
                  <span class="text-xs text-ink block font-medium truncate">{drag.dragLabel()}</span>
                  <span class="text-[11px] text-ink-dim font-mono block truncate">{drag.dragCommand()}</span>
                </div>
              </div>
            </div>
          </Show>
        </div>
      </Portal>
    </Show>
  );
};
