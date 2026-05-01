import { Component, For, createEffect } from "solid-js";
import { useChat } from "../context/chat.js";
import { ChatBubble } from "./ChatBubble.js";

export const ChatArea: Component = () => {
  const { messages } = useChat();
  let chatRef: HTMLElement | undefined;

  createEffect(() => {
    messages();
    chatRef?.scrollTo({ top: chatRef.scrollHeight, behavior: "smooth" });
  });

  return (
    <main class="flex-1 overflow-y-auto p-4 flex flex-col gap-3" ref={chatRef}>
      <For each={messages()}>
        {(message) => <ChatBubble message={message} />}
      </For>
    </main>
  );
};
