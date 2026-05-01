import { Component } from "solid-js";
import { Portal } from "solid-js/web";
import { useConnection } from "../context/connection.js";

export const KeysOverlay: Component = () => {
  const { send } = useConnection();

  const sendKey = (data: string): void => {
    send({ type: "key", data });
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
    <Portal mount={document.getElementById("keys-layer")!}>
      <div class="fixed bottom-16 left-3 right-3 px-3 py-2 bg-surface border border-edge rounded-lg shadow-lg">
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
    </Portal>
  );
};
