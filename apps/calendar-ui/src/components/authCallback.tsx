import { urls } from "../services/apiRoutes";
import { httpPost } from "../services/httpService";

export function AuthCallback() {
  const queryString = window.location.search;
  const urlParams = new URLSearchParams(queryString);
  const code = urlParams.get("code");

  if (code) {
    httpPost(urls.authCallback, { code })
      .then(data => {
        console.log(data);
        if (data.token) {
          localStorage.setItem("jwt", data.token); // Store JWT for authentication
          window.location.href = "/dashboard"; // Redirect to dashboard
        } else {
          console.error("Login failed", data.error);
        }
      })
      .catch(error => console.error("Error:", error));
  }

  return <div>Authenticated from google</div>;
}
