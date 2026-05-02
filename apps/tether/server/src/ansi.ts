import stripAnsiLib from "strip-ansi";

export function stripAnsi(text: string): string {
  let cleaned = text;

  // Replace cursor-forward sequences with spaces (preserves word spacing)
  // eslint-disable-next-line no-control-regex
  cleaned = cleaned.replace(/\x1b\[(\d+)C/g, (_match, n) => {
    return " ".repeat(Number(n));
  });

  // eslint-disable-next-line no-control-regex
  cleaned = cleaned.replace(/\x1b\[C/g, " ");

  // Replace horizontal tabs with spaces
  cleaned = cleaned.replace(/\t/g, "    ");

  // eslint-disable-next-line no-control-regex
  cleaned = cleaned.replace(/\x1b\].*?(?:\x1b\\|\x07)/gs, "");

  // eslint-disable-next-line no-control-regex
  cleaned = cleaned.replace(/\x1bP.*?(?:\x1b\\)/gs, "");

  // Use strip-ansi for standard CSI/SGR sequences
  cleaned = stripAnsiLib(cleaned);

  // eslint-disable-next-line no-control-regex
  cleaned = cleaned.replace(/\x1b\[[\x20-\x3f]*[\x40-\x7e]/g, "");

  // eslint-disable-next-line no-control-regex
  cleaned = cleaned.replace(/\x1b[^[\]P]/g, "");

  // eslint-disable-next-line no-control-regex
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
