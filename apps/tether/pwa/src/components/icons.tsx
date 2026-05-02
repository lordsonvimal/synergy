import { Component } from "solid-js";

interface IconProps {
  class?: string;
}

export const MicIcon: Component<IconProps> = (props) => (
  <svg class={props.class ?? "w-5 h-5"} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <rect x="9" y="1" width="6" height="13" rx="3" />
    <path d="M5 10a7 7 0 0 0 14 0" />
    <line x1="12" y1="17" x2="12" y2="21" />
    <line x1="8" y1="21" x2="16" y2="21" />
  </svg>
);

export const StopIcon: Component<IconProps> = (props) => (
  <svg class={props.class ?? "w-5 h-5"} viewBox="0 0 24 24" fill="currentColor">
    <rect x="6" y="6" width="12" height="12" rx="2" />
  </svg>
);
