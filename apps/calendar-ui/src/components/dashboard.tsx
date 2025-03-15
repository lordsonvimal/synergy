import { onMount, Show } from "solid-js";
import { getUserStore, loadUser } from "../stores/userStore";
import { logout } from "../stores/authStore";
import { useNavigate } from "@solidjs/router";
import { clientRoutes } from "../services/clientRoutes";

export function Dashboard() {
  onMount(() => {
    loadUser();
  });

  const store = getUserStore();
  const nav = useNavigate();

  const onLogout = () => {
    nav(clientRoutes.home);
  };

  return (
    <Show when={store.user} fallback={<div>User not present</div>}>
      {user => {
        console.log(user().picture);

        return (
          <>
            <img
              src={`${user().picture.replace("https", "http")}`}
              alt="User Profile"
            />
            <div>{user().display_name}</div>
            <button on:click={() => logout(onLogout)}>Logout</button>
          </>
        );
      }}
    </Show>
  );
}
