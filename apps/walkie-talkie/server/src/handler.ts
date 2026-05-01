import { IPty } from "node-pty";
import { writeToTerminal, stopTerminal } from "./terminal.js";

interface TextMessage {
  type: "text";
  data: string;
}

interface PermissionResponseMessage {
  type: "permission_response";
  value: "yes" | "no" | "always";
}

interface StopMessage {
  type: "stop";
}

type ClientMessage = TextMessage | PermissionResponseMessage | StopMessage;

const PERMISSION_MAP: Record<string, string> = {
  yes: "y",
  no: "n",
  always: "a"
};

export function handleMessage(message: ClientMessage, pty: IPty): void {
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
  }
}
