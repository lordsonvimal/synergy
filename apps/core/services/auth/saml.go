package auth

// import (
// 	"net/http"
// 	"os"

// 	"github.com/crewjam/saml"
// 	"github.com/gorilla/sessions"
// )

// type SAMLAuthenticator struct {
// 	serviceProvider *saml.ServiceProvider
// 	store           *sessions.CookieStore
// }

// func NewSAMLAuthenticator() *SAMLAuthenticator {
// 	return &SAMLAuthenticator{
// 		serviceProvider: &saml.ServiceProvider{
// 			EntityID:    os.Getenv("SAML_ENTITY_ID"),
// 			MetadataURL: os.Getenv("SAML_METADATA_URL"),
// 		},
// 		store: sessions.NewCookieStore([]byte("saml-session-secret")),
// 	}
// }

// func (s *SAMLAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
// 	url, err := s.serviceProvider.MakeRedirectAuthenticationRequest("relay-state")
// 	if err != nil {
// 		http.Error(w, "Failed to generate SAML request", http.StatusInternalServerError)
// 		return
// 	}
// 	http.Redirect(w, r, url, http.StatusFound)
// }

// func (s *SAMLAuthenticator) Callback(w http.ResponseWriter, r *http.Request) {
// 	w.Write([]byte(`{"message": "SAML authentication successful"}`))
// }

// func (s *SAMLAuthenticator) Logout(w http.ResponseWriter, r *http.Request) {
// 	session, _ := s.store.Get(r, "saml-session")
// 	delete(session.Values, "user_id")
// 	session.Save(r, w)
// 	w.Write([]byte(`{"message": "SAML logout successful"}`))
// }

// func (s *SAMLAuthenticator) Authenticate(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		http.Error(w, "SAML authentication not implemented for middleware", http.StatusNotImplemented)
// 	})
// }
