import { urls } from "../services/apiRoutes";
import { httpPost } from "../services/httpService";

function login() {
  httpPost(urls.authLogin, {})
    .then(data => {
      console.log(data);
      window.location.href = data.url;
    })
    .catch(e => {
      console.log(e);
    });
}

export function Home() {
  return <button on:click={login}>Login</button>;
}
