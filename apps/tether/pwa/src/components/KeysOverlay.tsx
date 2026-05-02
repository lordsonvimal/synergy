import { Component, Show, createSignal, createEffect } from "solid-js";
import { Portal } from "solid-js/web";
import { useConnection } from "../context/connection.js";
import { usePanes } from "../context/panes.js";

interface KeysOverlayProps {
  open: boolean;
  onClose: () => void;
}

export const KeysOverlay: Component<KeysOverlayProps> = props => {
  const { send } = useConnection();
  const { activeTabId } = usePanes();
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

  const sendKey = (data: string): void => {
    send({ type: "key", tabId: activeTabId(), data });
  };

  const keys = [
    { label: "Enter", data: "\r" },
    { label: "Tab", data: "\t" },
    { label: "Esc", data: "\x1b" },
    { label: "↑", data: "\x1b[A" },
    { label: "↓", data: "\x1b[B" },
    { label: "←", data: "\x1b[D" },
    { label: "→", data: "\x1b[C" },
    { label: "Ctrl+C", data: "\x03" },
  ];

  return (
    <Show when={mounted()}>
      <Portal mount={document.getElementById("keys-layer")!}>
        <div class="fixed inset-0" data-testid="keys-overlay-backdrop">
          <div
            class="absolute inset-0"
            onClick={props.onClose}
          />
          <div
            class={`absolute bottom-16 left-3 right-3 px-3 py-2 bg-surface border border-edge rounded-lg shadow-lg ${
              closing() ? "animate-slide-down" : "animate-slide-up"
            }`}
            onAnimationEnd={handleAnimationEnd}
          >
            <div class="flex items-center gap-1.5 overflow-x-auto">
              {keys.map(k => (
                <button
                  class="bg-muted border border-edge text-ink px-3 py-1.5 rounded-md text-xs font-mono cursor-pointer hover:bg-surface-raised hover:border-edge-strong hover:text-primary active:scale-95 transition-all whitespace-nowrap"
                  onPointerDown={e => {
                    e.preventDefault();
                    sendKey(k.data);
                  }}
                >
                  {k.label}
                </button>
              ))}
            </div>
          </div>
        </div>
      </Portal>
    </Show>
  );
};
