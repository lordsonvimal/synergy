import { createStore } from "solid-js/store";
import { clientRoutes } from "../services/clientRoutes";
import { httpGet, httpPost } from "../services/httpService";
import { urls } from "../services/apiRoutes";
import { useNavigate } from "@solidjs/router";

type AuthStore = {
  checkingLoginStatus: boolean;
  userId: number | null;
};

export const [authStore, setAuthStore] = createStore<AuthStore>({
  checkingLoginStatus: true,
  userId: null
});

function setUserId(userId: number | null) {
  if (userId) localStorage.setItem("userId", userId.toString());
  else localStorage.removeItem("userId");
  setAuthStore("userId", userId);
}
export function getUserId() {
  const userId = localStorage.getItem("userId");
  return userId;
}

export function logout(cb: VoidFunction) {
  httpPost(urls.authLogout, {})
    .then(() => {
      setUserId(null);
      cb();
    })
    .catch(e => {
      console.error(e);
    });
}

export function login() {
  const userId = getUserId();
  if (!userId) {
    setAuthStore("checkingLoginStatus", false);
    return;
  }
  const nav = useNavigate();
  httpGet(urls.authLogin)
    .then(response => {
      setUserId(response.user_id);
      nav(clientRoutes.dashboard);
    })
    .catch(() => {
      console.log("Not logged in");
    })
    .finally(() => {
      setAuthStore("checkingLoginStatus", false);
    });
}

export function loginFromCallback() {
  const queryString = window.location.search;
  const urlParams = new URLSearchParams(queryString);
  const code = urlParams.get("code");
  const state = urlParams.get("state");
  const nav = useNavigate();

  if (code) {
    httpGet(`${urls.authCallback}?code=${code}&state=${state}`)
      .then(response => {
        setUserId(response.user_id);
        nav(clientRoutes.dashboard);
      })
      .catch(error => {
        console.error("Error:", error);
        nav(clientRoutes.home);
      });
  }
}

export const isLoggedIn = () => getUserId() !== null;

export function redirect() {
  httpGet(urls.authRedirect)
    .then(data => {
      window.location.href = data.url;
    })
    .catch(e => {
      console.error(e);
    });
}
