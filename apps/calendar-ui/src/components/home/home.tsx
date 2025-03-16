import { onMount, Show } from "solid-js";
import { authStore, login, redirect } from "../../stores/authStore";
import { Button } from "../button/button";
import { Section } from "../layouts/Section";
import { Column } from "../layouts/column";

export function Home() {
  onMount(() => {
    login();
  });

  return (
    <Show
      when={!authStore.checkingLoginStatus}
      fallback={<div>Loading...</div>}
    >
      <Section align="center" minHeight="100%" height="100%">
        <Column
          gap="1rem"
          justifyContent="center"
          start={4}
          span={6}
          textAlign="center"
        >
          <h1 class="title">Synergy</h1>
          <Button onClick={redirect} size="medium" variant="primary-alt">
            <img
              src="https://www.svgrepo.com/show/355037/google.svg"
              alt="Google Logo"
              class="google-icon"
              height={30}
            />
            &nbsp;&nbsp;&nbsp;Login&nbsp;with&nbsp;<b>Google</b>
          </Button>
        </Column>
      </Section>
    </Show>
  );
}
