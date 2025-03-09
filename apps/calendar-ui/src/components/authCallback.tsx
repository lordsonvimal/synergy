import { urls } from "../services/apiRoutes";
import { httpGet } from "../services/httpService";

export function AuthCallback() {
  const queryString = window.location.search;
  const urlParams = new URLSearchParams(queryString);
  const code = urlParams.get("code");
  const state = urlParams.get("state");

  if (code) {
    httpGet(`${urls.authCallback}?code=${code}&state=${state}`)
      .then(data => {
        console.log(data);
        // if (data.token) {
        //   localStorage.setItem("jwt", data.token); // Store JWT for authentication
        //   window.location.href = "/dashboard"; // Redirect to dashboard
        // } else {
        //   console.error("Login failed", data.error);
        // }
      })
      .catch(error => console.error("Error:", error));
  }

  return <div>Authenticated from google</div>;
}
