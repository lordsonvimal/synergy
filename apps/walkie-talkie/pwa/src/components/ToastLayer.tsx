import { Component, For } from "solid-js";
import { Portal } from "solid-js/web";
import { getToasts, removeToast, Toast } from "../lib/toast.js";

const typeStyles: Record<string, string> = {
  error: "bg-error-subtle text-error border-error",
  warning: "bg-warning-subtle text-warning border-warning",
  info: "bg-info-subtle text-info border-info",
  success: "bg-success-subtle text-success border-success",
};

const ToastItem: Component<{ toast: Toast }> = (props) => {
  return (
    <div
      class={`flex items-center gap-2 px-3 py-2 rounded-lg border text-sm shadow-md animate-fade-in ${
        typeStyles[props.toast.type] ?? typeStyles.error
      }`}
      role="alert"
    >
      <span class="flex-1">{props.toast.message}</span>
      <button
        class="bg-transparent border-none text-current cursor-pointer text-base leading-none p-1 rounded-md opacity-70 hover:opacity-100 hover:bg-black/10 transition-all"
        onClick={() => removeToast(props.toast.id)}
        aria-label="Dismiss"
      >
        ✕
      </button>
    </div>
  );
};

export const ToastLayer: Component = () => {
  return (
    <Portal mount={document.getElementById("toast-layer")!}>
      <div class="fixed bottom-16 right-3 left-3 flex flex-col gap-2 pointer-events-none">
        <For each={getToasts()}>
          {toast => (
            <div class="pointer-events-auto ml-auto max-w-[90%]">
              <ToastItem toast={toast} />
            </div>
          )}
        </For>
      </div>
    </Portal>
  );
};
