import {
  Component,
  JSX,
  createContext,
  createSignal,
  useContext
} from "solid-js";

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  text: string;
  timestamp: number;
  streaming?: boolean;
}

interface ChatContextValue {
  messages: () => ChatMessage[];
  addMessage: (role: "user" | "assistant", text: string) => string;
  updateMessage: (id: string, text: string) => void;
  completeMessage: (id: string) => void;
  clearMessages: () => void;
}

const ChatContext = createContext<ChatContextValue>();

let messageId = 0;

function generateId(): string {
  messageId += 1;
  return `msg-${messageId}`;
}

export const ChatProvider: Component<{ children: JSX.Element }> = (props) => {
  const [messages, setMessages] = createSignal<ChatMessage[]>([]);

  const addMessage = (role: "user" | "assistant", text: string): string => {
    const id = generateId();
    const message: ChatMessage = {
      id,
      role,
      text,
      timestamp: Date.now(),
      streaming: role === "assistant"
    };
    setMessages((prev) => [...prev, message]);
    return id;
  };

  const updateMessage = (id: string, text: string): void => {
    setMessages((prev) =>
      prev.map((msg) => (msg.id === id ? { ...msg, text } : msg))
    );
  };

  const completeMessage = (id: string): void => {
    setMessages((prev) =>
      prev.map((msg) =>
        msg.id === id ? { ...msg, streaming: false } : msg
      )
    );
  };

  const clearMessages = (): void => {
    setMessages([]);
  };

  return (
    <ChatContext.Provider
      value={{ messages, addMessage, updateMessage, completeMessage, clearMessages }}
    >
      {props.children}
    </ChatContext.Provider>
  );
};

export function useChat(): ChatContextValue {
  const ctx = useContext(ChatContext);
  if (!ctx) {
    throw new Error("useChat must be used within ChatProvider");
  }
  return ctx;
}
