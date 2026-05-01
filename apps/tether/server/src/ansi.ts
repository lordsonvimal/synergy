import stripAnsiLib from "strip-ansi";

export function stripAnsi(text: string): string {
  let cleaned = text;

  // Replace cursor-forward sequences with spaces (preserves word spacing)
  // \x1b[<n>C = move cursor forward n columns
  cleaned = cleaned.replace(/\x1b\[(\d+)C/g, (_match, n) => {
    return " ".repeat(Number(n));
  });

  // Replace single cursor-forward (no number = 1)
  cleaned = cleaned.replace(/\x1b\[C/g, " ");

  // Replace horizontal tabs with spaces
  cleaned = cleaned.replace(/\t/g, "    ");

  // Remove OSC sequences: \x1b] ... (ST or BEL terminated)
  cleaned = cleaned.replace(/\x1b\].*?(?:\x1b\\|\x07)/gs, "");

  // Remove DCS sequences: \x1bP ... ST
  cleaned = cleaned.replace(/\x1bP.*?(?:\x1b\\)/gs, "");

  // Use strip-ansi for standard CSI/SGR sequences
  cleaned = stripAnsiLib(cleaned);

  // Remove CSI sequences that strip-ansi missed (private modes, cursor shape)
  cleaned = cleaned.replace(/\x1b\[[\x20-\x3f]*[\x40-\x7e]/g, "");

  // Remove remaining escape sequences
  cleaned = cleaned.replace(/\x1b[^[\]P]/g, "");

  // Remove leftover control characters except newline, carriage return, and tab
  cleaned = cleaned.replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]/g, "");

  // Handle carriage returns (terminal line overwrites — keep last write)
  cleaned = cleaned.split("\n").map(line => {
    if (line.includes("\r")) {
      const parts = line.split("\r");
      return parts[parts.length - 1];
    }
    return line;
  }).join("\n");

  // Remove cursor shape fragments (e.g. "0q", "u1u4;2m")
  cleaned = cleaned.replace(/^\s*\d*\s*q\s*$/gm, "");
  cleaned = cleaned.replace(/^[a-z]\d+[a-z]\d*;?\d*[a-z]?\s*$/gm, "");

  // Collapse multiple blank lines into max two
  cleaned = cleaned.replace(/\n{3,}/g, "\n\n");

  return cleaned.trim();
}
