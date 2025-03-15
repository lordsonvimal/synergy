import { createStore } from "solid-js/store";
import { clientRoutes } from "../services/clientRoutes";
import { httpGet, httpPost } from "../services/httpService";
import { urls } from "../services/apiRoutes";
import { useNavigate } from "@solidjs/router";

const [authStore, setAuthStore] = createStore<{ userId: number | null }>({
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
  // if (code) {
  //   httpGet(`${urls.authCallback}?code=${code}&state=${state}`)
  //     .then(response => {
  //       console.log(response);
  //       setUserId(response.userId);
  //       nav(clientRoutes.dashboard);
  //     })
  //     .catch(error => {
  //       console.error("Error:", error);
  //       nav(clientRoutes.home);
  //     });
  // }
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
