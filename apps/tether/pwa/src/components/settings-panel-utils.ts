import { Shortcut } from "../context/settings.js";

export const FONT_SIZE_OPTIONS: {
  value: "small" | "medium" | "large";
  label: string;
}[] = [
  { value: "small", label: "Small" },
  { value: "medium", label: "Medium" },
  { value: "large", label: "Large" }
];

export function findDragTargetId(
  listEl: Element,
  dragId: string,
  currentCenter: number
): string | null {
  const items = listEl.querySelectorAll<HTMLElement>(
    "[data-shortcut-id]"
  );
  let targetId: string | null = null;
  for (const item of items) {
    const itemId = item.getAttribute("data-shortcut-id");
    if (itemId === dragId) continue;
    const itemRect = item.getBoundingClientRect();
    const itemCenter = itemRect.top + itemRect.height / 2;
    if (currentCenter > itemCenter) targetId = itemId;
  }
  return targetId;
}

function resolveTargetIdx(
  shortcuts: Shortcut[],
  overId: string | null
): number {
  if (overId === "__before_first__" || overId === null) return 0;
  return shortcuts.findIndex(s => s.id === overId) + 1;
}

function isReorderNoop(
  sourceIdx: number,
  targetIdx: number
): boolean {
  return sourceIdx === targetIdx || sourceIdx + 1 === targetIdx;
}

export function reorderShortcuts(
  shortcuts: Shortcut[],
  sourceId: string,
  overId: string | null
): Shortcut[] | null {
  const result = [...shortcuts];
  const sourceIdx = result.findIndex(s => s.id === sourceId);
  if (sourceIdx === -1) return null;
  const targetIdx = resolveTargetIdx(result, overId);
  if (isReorderNoop(sourceIdx, targetIdx)) return null;
  const [removed] = result.splice(sourceIdx, 1);
  if (!removed) return null;
  const insertAt =
    targetIdx > sourceIdx ? targetIdx - 1 : targetIdx;
  result.splice(insertAt, 0, removed);
  return result;
}

export function flipTheme(current: string): "dark" | "light" {
  return current === "dark" ? "light" : "dark";
}

export function themeButtonLabel(theme: string): string {
  return theme === "dark" ? "☀️ Light" : "🌙 Dark";
}

export function fontSizeButtonClass(active: boolean): string {
  const base =
    "px-3 py-1.5 text-xs border-none cursor-pointer transition-colors";
  return active
    ? `${base} bg-primary text-on-primary`
    : `${base} bg-muted text-ink hover:bg-surface-raised`;
}

export function toggleBgClass(enabled: boolean): string {
  return enabled ? "bg-primary" : "bg-muted border border-edge";
}

export function toggleTranslate(enabled: boolean): string {
  return enabled ? "translateX(22px)" : "translateX(2px)";
}

export function closingClass(
  isClosing: boolean,
  openClass: string,
  closedClass: string
): string {
  return isClosing ? closedClass : openClass;
}

export function modeLabel(mode: string): string {
  return mode === "mirror"
    ? "Shared sessions across devices"
    : "Separate sessions per device";
}
