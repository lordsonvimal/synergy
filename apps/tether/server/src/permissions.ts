export interface PermissionPrompt {
  action: string;
  options: string[];
}

const PERMISSION_PATTERN = /Allow\s+(.+?)\?\s*\[y\/n\/a\]/i;

export function detectPermission(text: string): PermissionPrompt | null {
  const match = PERMISSION_PATTERN.exec(text);
  if (!match?.[1]) {
    return null;
  }

  return {
    action: match[1].trim(),
    options: ["yes", "no", "always"]
  };
}
