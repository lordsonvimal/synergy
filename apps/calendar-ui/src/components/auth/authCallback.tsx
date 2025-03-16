import { loginFromCallback } from "../../stores/authStore";

export function AuthCallback() {
  loginFromCallback();
  return <p>Authenticating...</p>;
}
