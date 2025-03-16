import { onMount, Show } from "solid-js";
import { authStore, login, redirect } from "../stores/authStore";

export function Home() {
  onMount(() => {
    login();
  });

  return (
    <Show
      when={!authStore.checkingLoginStatus}
      fallback={<div>Loading...</div>}
    >
      <button on:click={redirect}>Login</button>
    </Show>
  );
}
