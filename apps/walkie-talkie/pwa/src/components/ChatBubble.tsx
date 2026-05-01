import { Component, Show } from "solid-js";
import { ChatMessage } from "../context/chat.js";
import { renderMarkdown } from "../lib/renderer.js";

interface ChatBubbleProps {
  message: ChatMessage;
}

function formatTime(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit"
  });
}

export const ChatBubble: Component<ChatBubbleProps> = (props) => {
  const isUser = () => props.message.role === "user";

  return (
    <div
      class={`max-w-[90%] px-4 py-3 rounded-2xl animate-fade-in ${
        isUser()
          ? "self-end bg-accent text-white max-w-[85%] rounded-br-sm"
          : "self-start bg-surface rounded-bl-sm"
      }`}
    >
      <div class="flex gap-2 items-center mb-1">
        <span class="text-xs font-semibold">
          {isUser() ? "You" : "Claude"}
        </span>
        <span class="text-[11px] text-text-secondary">
          {formatTime(props.message.timestamp)}
        </span>
      </div>
      <div
        class="leading-relaxed [&_pre]:bg-bg [&_pre]:p-3 [&_pre]:rounded-lg [&_pre]:overflow-x-auto [&_pre]:my-2 [&_code]:font-mono [&_code]:text-[13px]"
        innerHTML={
          isUser() ? props.message.text : renderMarkdown(props.message.text)
        }
      />
      <Show when={props.message.streaming}>
        <span class="inline-block w-2 h-4 bg-accent ml-0.5 animate-blink" />
      </Show>
    </div>
  );
};
