package sso

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	dsig "github.com/russellhaering/goxmldsig"
)

func Login(c *gin.Context) {
	samlSP, exists := c.MustGet("samlSP").(*samlsp.Middleware)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SAML Middleware not found"})
		return
	}

	samlSP.RequireAccount(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Request = r
		c.Next() // Just call Next, no need to touch c.Writer
	})).ServeHTTP(c.Writer, c.Request)
}

func ACS(c *gin.Context) {
	// Extract the SAML response from the POST request
	req := c.Request
	writer := c.Writer
	httpsCert := os.Getenv("HTTPS_SERVER_CERT")

	req.ParseForm()
	samlResponseRaw := req.FormValue("SAMLResponse")
	if samlResponseRaw == "" {
		fmt.Println("No SAMLResponse found in request")
		return
	}

	// Decode the Base64 SAML response for inspection
	samlResponseDecoded, err := base64.StdEncoding.DecodeString(samlResponseRaw)
	if err != nil {
		fmt.Println("Error decoding SAMLResponse:", err)
		return
	}

	// Parse the SAML response XML using etree
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(samlResponseDecoded); err != nil {
		fmt.Errorf("failed to parse XML: %v", err)
	}

	// Validate the SAML response
	// Parse the SAML response XML
	var response saml.Response
	if err := xml.Unmarshal(samlResponseDecoded, &response); err != nil {
		fmt.Errorf("failed to unmarshal SAML response: %v", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
	}

	// Load the certificate to validate the signature
	certPEM, err := os.ReadFile(httpsCert)
	if err != nil {
		fmt.Errorf("failed to read certificate: %v", err)
	}

	// Parse the certificate
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Create a dsig (digital signature) context for signature verification
	context := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{cert},
	})

	// Verify the SAML response signature
	_, err = context.Validate(doc.Root())
	if err != nil {
		fmt.Errorf("SAML response signature validation failed: %v", err)
	}

	// Check if the assertion is encrypted
	if response.EncryptedAssertion != nil {
		// At this point, we have an EncryptedAssertion but no built-in decryption
		// This is where you would perform custom decryption (not covered in this example)
		fmt.Errorf("encrypted assertion detected, decryption not implemented")
	} else if response.Assertion != nil {
		// If the assertion is not encrypted, validate it directly
		err := validateAssertion(response.Assertion, "https://127.0.0.1:8080")
		if err != nil {
			fmt.Errorf(err.Error())
		}
	}

	// Redirect to a protected page after login
	c.JSON(http.StatusOK, gin.H{"message": "SAML Response processed successfully"})
}

// Validate the assertion's conditions and audience
func validateAssertion(assertion *saml.Assertion, expectedAudience string) error {
	// Validate assertion's conditions (like timing and audience)
	if assertion.Conditions != nil {
		if err := validateTimeConditions(assertion.Conditions); err != nil {
			return err
		}

		// Validate the audience restriction
		for _, audienceRestriction := range assertion.Conditions.AudienceRestrictions {
			if audienceRestriction.Audience.Value != expectedAudience {
				return fmt.Errorf("invalid audience: %s", audienceRestriction.Audience.Value)
			}
		}
	}

	// Extract attributes from the assertion
	for _, attrStatement := range assertion.AttributeStatements {
		for _, attr := range attrStatement.Attributes {
			if len(attr.Values) == 1 {
				fmt.Printf("Attribute: %s = %s\n", attr.Name, attr.Values[0].Value)
			}
		}
	}

	return nil
}

// Validate the time-based conditions of the assertion
func validateTimeConditions(conditions *saml.Conditions) error {
	now := time.Now()
	if now.Before(conditions.NotBefore) {
		return fmt.Errorf("assertion is not yet valid")
	}
	if now.After(conditions.NotOnOrAfter) {
		return fmt.Errorf("assertion has expired")
	}
	return nil
}
