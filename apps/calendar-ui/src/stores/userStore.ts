import { createStore } from "solid-js/store";
import { getUserId, logout } from "./authStore";
import { ApiError, httpGet } from "../services/httpService";
import { urls } from "../services/apiRoutes";
import { useNavigate } from "@solidjs/router";
import { clientRoutes } from "../services/clientRoutes";

type User = {
  id: number;
  email: string;
  display_name: string;
  picture: string;
};

type Store = {
  user: User | null;
  profileUrl: string | null;
};

const [userStore, setUserStore] = createStore<Store>({
  user: null,
  profileUrl: null
});

export function loadUser() {
  const nav = useNavigate();
  const userId = getUserId();
  if (!userId) {
    console.error("User id not found");
    nav(clientRoutes.home);
    return;
  }

  httpGet(urls.userShow(userId))
    .then(res => {
      console.log(res);

      setUserStore("user", res.user);
    })
    .catch(err => {
      if (err instanceof ApiError) {
        if (err.status === 401) {
          logout(() => nav(clientRoutes.home));
          return;
        }

        if (err.json.error) {
          logout(() => nav(clientRoutes.home));
          return;
        }
      }

      console.error(err);
    });
}

export function getUserStore() {
  return userStore;
}
