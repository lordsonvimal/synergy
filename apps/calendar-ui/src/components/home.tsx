import { urls } from "../services/apiRoutes";
import { httpGet } from "../services/httpService";

function redirect() {
  httpGet(urls.authRedirect)
    .then(data => {
      window.location.href = data.url;
    })
    .catch(e => {
      console.error(e);
    });
}

export function Home() {
  return <button on:click={redirect}>Login</button>;
}
