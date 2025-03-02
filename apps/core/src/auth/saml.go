package auth

import (
	"log"
	"net/http"

	"github.com/crewjam/saml/samlsp"
)

type SAMLAuthenticator struct {
	spMiddleware *samlsp.Middleware
}

func NewSAMLAuthenticator() *SAMLAuthenticator {
	samlSP, err := samlsp.New(samlsp.Options{
		URL:            *samlsp.MustParseURL("http://localhost:8080"),
		Key:            samlsp.DefaultKey,
		Certificate:    samlsp.DefaultCertificate,
		IDPMetadataURL: samlsp.MustParseURL("https://your-idp.com/metadata"),
	})
	if err != nil {
		log.Fatalf("Failed to initialize SAML SP: %v", err)
	}

	return &SAMLAuthenticator{spMiddleware: samlSP}
}

func (s *SAMLAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
	s.spMiddleware.RequireAccount(w, r)
}

func (s *SAMLAuthenticator) Callback(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "SAML authentication successful"}`))
}

func (s *SAMLAuthenticator) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Logged out successfully."}`))
}
