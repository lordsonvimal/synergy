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
      class={`max-w-[90%] px-4 py-3 animate-fade-in ${
        isUser()
          ? "self-end bg-primary text-on-primary max-w-[85%] rounded-lg rounded-br-sm"
          : "self-start bg-surface border border-edge rounded-lg rounded-bl-sm"
      }`}
      data-testid="chat-bubble"
    >
      <div class="flex gap-2 items-center mb-1">
        <span class="text-xs font-semibold">
          {isUser() ? "You" : "Claude"}
        </span>
        <span
          class={`text-[11px] ${
            isUser() ? "text-on-primary/70" : "text-ink-secondary"
          }`}
        >
          {formatTime(props.message.timestamp)}
        </span>
      </div>
      <div
        class="leading-relaxed whitespace-pre-wrap break-words [&_pre]:bg-muted [&_pre]:p-3 [&_pre]:rounded-md [&_pre]:overflow-x-auto [&_pre]:my-2 [&_code]:font-mono [&_code]:text-[13px] [&_hr]:border-edge [&_hr]:my-3"
        innerHTML={
          isUser() ? props.message.text : renderMarkdown(props.message.text)
        }
      />
      <Show when={props.message.streaming}>
        <span class="inline-block w-2 h-4 bg-primary ml-0.5 animate-blink" />
      </Show>
    </div>
  );
};
