// src/components/Button.tsx
import { createSignal, JSX, splitProps } from "solid-js";
import styles from "./Button.module.scss";

type ButtonProps = {
  children: JSX.Element;
  onClick?: (e: MouseEvent) => Promise<void> | void;
  beforeClick?: () => boolean | Promise<boolean>; // Validate before clicking
  afterClick?: () => void; // Called after click completes
  variant?: "primary" | "primary-alt" | "secondary" | "danger";
  size?: "small" | "medium" | "large";
  disabled?: boolean;
  loadingText?: string;
  preventMultipleClicks?: boolean; // Prevent double clicks
  debounceTime?: number; // Prevent rapid clicks
  type?: "button" | "submit"; // Form support
};

export function Button(props: ButtonProps) {
  const [loading, setLoading] = createSignal(false);
  const [lastClickTime, setLastClickTime] = createSignal(0);

  const [local, rest] = splitProps(props, [
    "children",
    "onClick",
    "beforeClick",
    "afterClick",
    "variant",
    "size",
    "disabled",
    "loadingText",
    "preventMultipleClicks",
    "debounceTime",
    "type"
  ]);

  const handleClick = async (e: MouseEvent) => {
    if (local.disabled || loading()) return;

    const now = Date.now();
    if (
      local.preventMultipleClicks &&
      now - lastClickTime() < (local.debounceTime || 500)
    ) {
      return;
    }
    setLastClickTime(now);

    if (local.beforeClick) {
      const isAllowed = await local.beforeClick();
      if (!isAllowed) return;
    }

    if (local.onClick) {
      try {
        const result = local.onClick(e);
        if (result instanceof Promise) {
          setLoading(true);
          await result;
        }
      } finally {
        setLoading(false);
      }
    }

    local.afterClick?.();
  };

  const colorClass = styles[`btn-${local.variant || "primary"}`];
  const sizeClass = styles[`btn-${local.size || "medium"}`];
  return (
    <button
      type={local.type || "button"}
      class={`${styles.btn} ${colorClass} ${sizeClass}`}
      onClick={handleClick}
      disabled={local.disabled || loading()}
      {...rest}
    >
      {loading() ? local.loadingText || "Loading..." : local.children}
    </button>
  );
}
