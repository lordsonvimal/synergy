const CLIENT_ID =
  "624406138435-h07mq719hflrah516ogoubi47elnjrde.apps.googleusercontent.com";
const REDIRECT_URI = "http://localhost:3001/callback";
const AUTH_SERVER = "https://accounts.google.com/o/oauth2/v2/auth";

function base64UrlEncode(arrayBuffer: any) {
  return btoa(String.fromCharCode(...new Uint8Array(arrayBuffer)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, ""); // URL-safe base64
}

function generateCodeVerifier() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);

  return Array.from(
    array,
    byte => ("0" + byte.toString(16)).slice(-2) // Convert to hex
  ).join("");
}

async function generateCodeChallenge(codeVerifier: string | undefined) {
  const encoder = new TextEncoder();
  const data = encoder.encode(codeVerifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return base64UrlEncode(digest);
}

async function generatePKCE() {
  const codeVerifier = generateCodeVerifier();
  const codeChallenge = await generateCodeChallenge(codeVerifier);

  localStorage.setItem("code_verifier", codeVerifier); // Store for later

  return codeChallenge;
}

// Usage example
generatePKCE().then(codeChallenge => {
  console.log("Code Challenge:", codeChallenge);
});

async function login() {
  const codeChallenge = await generatePKCE();
  const state = generateCodeVerifier();

  const authUrl = `${AUTH_SERVER}?client_id=${CLIENT_ID}&redirect_uri=${encodeURIComponent(
    REDIRECT_URI
  )}&response_type=code&scope=openid%20email%20profile&code_challenge=${codeChallenge}&code_challenge_method=S256&state=${state}`;

  console.log(authUrl);

  window.location.href = authUrl; // Redirect user to Google login
}

export function Home() {
  return <button on:click={login}>Login</button>;
}
