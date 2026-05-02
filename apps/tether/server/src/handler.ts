import {
  TerminalManager,
  writeToTerminal,
  stopTerminal,
  resizeTerminal
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

export function handleMessage(
  message: ClientMessage,
  manager: TerminalManager
): void {
  switch (message.type) {
    case "create-tab":
      manager.createTab(message.tabId);
      break;
    case "close-tab":
      manager.closeTab(message.tabId);
      break;
    default: {
      const pty = manager.getTab(message.tabId);
      if (!pty) return;

      switch (message.type) {
        case "text":
          writeToTerminal(pty, message.data);
          break;
        case "permission_response":
          writeToTerminal(pty, PERMISSION_MAP[message.value] ?? "n");
          break;
        case "stop":
          stopTerminal(pty);
          break;
        case "resize":
          resizeTerminal(pty, message.cols, message.rows);
          break;
        case "key":
          pty.write(message.data);
          break;
      }
    }
  }
}
