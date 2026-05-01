import { Component } from "solid-js";
import { useConnection } from "../context/connection.js";
import { MicButton } from "./MicButton.js";

export const TerminalToolbar: Component = () => {
  const { send } = useConnection();

  const sendKey = (data: string): void => {
    send({ type: "key", data });
  };

  const handleVoiceSend = (text: string): void => {
    send({ type: "text", data: text });
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
    <div
      class="shrink-0 flex flex-col items-center gap-2 px-3 py-2 bg-surface border-t border-edge"
      data-testid="terminal-toolbar"
    >
      <div class="flex items-center gap-1.5 overflow-x-auto w-full">
        {keys.map(k => (
          <button
            class="bg-muted border border-edge text-ink px-3 py-1.5 rounded-md text-xs font-mono cursor-pointer hover:bg-surface-raised active:scale-95 transition-all whitespace-nowrap"
            onPointerDown={e => {
              e.preventDefault();
              sendKey(k.data);
            }}
          >
            {k.label}
          </button>
        ))}
      </div>
      <MicButton onSend={handleVoiceSend} />
    </div>
  );
};
