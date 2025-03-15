package cookie

import "net/http"

func SetCookie(w http.ResponseWriter, key string, value string) {
	// Store state in a cookie (SPA retrieves it later)
	http.SetCookie(w, &http.Cookie{
		Name:     key,
		Value:    value,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
		Path:     "/",
	})
}
