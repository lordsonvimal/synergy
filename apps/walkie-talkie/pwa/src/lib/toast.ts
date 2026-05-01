import { createSignal } from "solid-js";

export type ToastType = "error" | "warning" | "info" | "success";

export interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

let nextId = 0;

const [toasts, setToasts] = createSignal<Toast[]>([]);

export function addToast(message: string, type: ToastType = "error"): void {
  const id = nextId++;
  setToasts(prev => [...prev, { id, message, type }]);
}

export function removeToast(id: number): void {
  setToasts(prev => prev.filter(t => t.id !== id));
}

export function getToasts(): Toast[] {
  return toasts();
}
