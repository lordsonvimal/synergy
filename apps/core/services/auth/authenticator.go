package auth

var authenticator *OAuthAuthenticator

// SetAuthenticator initializes the authenticator (called in main)
func SetAuthenticator(a *OAuthAuthenticator) {
	authenticator = a
}

func GetAuthenticator() *OAuthAuthenticator {
	return authenticator
}
