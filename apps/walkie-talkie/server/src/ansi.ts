import stripAnsiLib from "strip-ansi";

export function stripAnsi(text: string): string {
  return stripAnsiLib(text);
}
