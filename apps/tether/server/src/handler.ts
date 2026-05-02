import {
  TerminalManager,
  writeToTerminal,
  stopTerminal
} from "./terminal.js";

interface TextMessage {
  type: "text";
  tabId: string;
  data: string;
}

interface PermissionResponseMessage {
  type: "permission_response";
  tabId: string;
  value: "yes" | "no" | "always";
}

interface StopMessage {
  type: "stop";
  tabId: string;
}

interface ResizeMessage {
  type: "resize";
  tabId: string;
  cols: number;
  rows: number;
}

interface KeyMessage {
  type: "key";
  tabId: string;
  data: string;
}

interface CreateTabMessage {
  type: "create-tab";
  tabId: string;
  cols?: number;
  rows?: number;
}

interface CloseTabMessage {
  type: "close-tab";
  tabId: string;
}

type ClientMessage =
  | TextMessage
  | PermissionResponseMessage
  | StopMessage
  | ResizeMessage
  | KeyMessage
  | CreateTabMessage
  | CloseTabMessage;

const PERMISSION_MAP: Record<string, string> = {
  yes: "y",
  no: "n",
  always: "a"
};

function handleTextOrKey(
  message: TextMessage | KeyMessage | PermissionResponseMessage,
  pty: ReturnType<TerminalManager["getTab"]> & {}
): void {
  if (message.type === "text") {
    writeToTerminal(pty, message.data);
  } else if (message.type === "permission_response") {
    writeToTerminal(pty, PERMISSION_MAP[message.value] ?? "n");
  } else {
    pty.write(message.data);
  }
}

function handleTabMessage(
  message: TextMessage | PermissionResponseMessage | StopMessage | ResizeMessage | KeyMessage,
  manager: TerminalManager
): void {
  const pty = manager.getTab(message.tabId);
  if (!pty) return;

  if (message.type === "stop") {
    stopTerminal(pty);
  } else if (message.type === "resize") {
    manager.resizeTab(message.tabId, message.cols, message.rows);
  } else {
    handleTextOrKey(message, pty);
  }
}

export function handleMessage(
  message: ClientMessage,
  manager: TerminalManager
): void {
  switch (message.type) {
    case "create-tab":
      manager.createTab(message.tabId, message.cols, message.rows);
      break;
    case "close-tab":
      manager.closeTab(message.tabId);
      break;
    default:
      handleTabMessage(message, manager);
  }
}
