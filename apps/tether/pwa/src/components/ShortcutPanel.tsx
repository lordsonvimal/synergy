import { Component, Show, For, createSignal, createEffect } from "solid-js";
import { Portal } from "solid-js/web";
import { useSettings, Shortcut } from "../context/settings.js";

interface ShortcutPanelProps {
  open: boolean;
  onClose: () => void;
  onSend: (command: string) => void;
}

const LONG_PRESS_MS = 500;

export const ShortcutPanel: Component<ShortcutPanelProps> = props => {
  const { settings } = useSettings();
  const [editing, setEditing] = createSignal<Shortcut | null>(null);
  const [editValue, setEditValue] = createSignal("");
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
  let pressTimer: ReturnType<typeof setTimeout> | undefined;
  let didLongPress = false;

  const handlePointerDown = (shortcut: Shortcut): void => {
    didLongPress = false;
    pressTimer = setTimeout(() => {
      didLongPress = true;
      setEditValue(shortcut.command);
      setEditing(shortcut);
    }, LONG_PRESS_MS);
  };

  const handlePointerUp = (shortcut: Shortcut): void => {
    clearTimeout(pressTimer);
    if (!didLongPress) {
      props.onSend(shortcut.command);
      props.onClose();
    }
  };

  const handlePointerLeave = (): void => {
    clearTimeout(pressTimer);
  };

  const handleEditSend = (): void => {
    const cmd = editValue().trim();
    if (cmd) {
      props.onSend(cmd);
      setEditing(null);
      setEditValue("");
      props.onClose();
    }
  };

  const handleEditKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleEditSend();
    } else if (e.key === "Escape") {
      e.preventDefault();
      setEditing(null);
      setEditValue("");
    }
  };

  const handleClose = (): void => {
    setEditing(null);
    setEditValue("");
    props.onClose();
  };

  return (
    <Show when={mounted()}>
      <Portal mount={document.getElementById("shortcuts-layer")!}>
        <div class="fixed inset-0" data-testid="shortcut-panel-backdrop">
          <div
            class="absolute inset-0"
            onClick={handleClose}
          />
          <div
            class={`absolute bottom-16 left-3 right-3 bg-surface border border-edge rounded-lg shadow-lg ${
              closing() ? "animate-slide-down" : "animate-slide-up"
            }`}
            data-testid="shortcut-panel"
            onAnimationEnd={handleAnimationEnd}
          >
            <Show
              when={!editing()}
              fallback={
                <div class="p-3 flex items-center gap-2" data-testid="shortcut-edit-prompt">
                  <span class="text-xs text-ink-secondary shrink-0">
                    {editing()!.label}:
                  </span>
                  <input
                    type="text"
                    class="flex-1 bg-canvas border border-edge-strong rounded-md text-sm text-ink px-3 py-1.5 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25"
                    value={editValue()}
                    onInput={e => setEditValue(e.currentTarget.value)}
                    onKeyDown={handleEditKeyDown}
                    autofocus
                    data-testid="shortcut-edit-input"
                  />
                  <button
                    class="bg-primary border-none text-on-primary px-3 py-1.5 rounded-md text-sm cursor-pointer hover:bg-primary-hover active:scale-95 transition-all"
                    onClick={handleEditSend}
                    data-testid="shortcut-edit-send"
                  >
                    Send
                  </button>
                </div>
              }
            >
              <div class="p-3">
                <div class="flex items-center justify-between mb-2">
                  <span class="text-xs text-ink-dim">
                    Tap to run, hold to edit
                  </span>
                </div>
                <div class="flex flex-wrap gap-2">
                  <For each={settings().shortcuts}>
                    {shortcut => (
                      <button
                        class="bg-muted border border-edge text-ink px-3 py-1.5 rounded-md text-sm font-mono cursor-pointer hover:bg-surface-raised hover:border-edge-strong active:scale-95 transition-all select-none touch-manipulation"
                        onPointerDown={() => handlePointerDown(shortcut)}
                        onPointerUp={() => handlePointerUp(shortcut)}
                        onPointerLeave={handlePointerLeave}
                        data-testid={`shortcut-${shortcut.id}`}
                      >
                        {shortcut.label}
                      </button>
                    )}
                  </For>
                </div>
              </div>
            </Show>
          </div>
        </div>
      </Portal>
    </Show>
  );
};
