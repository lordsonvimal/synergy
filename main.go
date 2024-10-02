package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	dsig "github.com/russellhaering/goxmldsig"
)

var samlSP *samlsp.Middleware // Declare samlSP globally

var jwtSecret []byte

// Initialize environment variables and SAML SP
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET is not set in the environment")
	}
}

func main() {
	r := gin.Default()

	// SAML Service Provider setup
	rootURL, err := url.Parse("http://127.0.0.1:8080") // Base URL for your service provider
	if err != nil {
		log.Fatalf("Invalid root URL: %v", err)
	}

	// Get metadata
	metadataFile := os.Getenv("SAML_GOOGLE_IDP_METADATA_PATH")
	metadataBytes, err := os.ReadFile(metadataFile)
	if err != nil {
		log.Fatalf("Failed to read IdP metadata file: %v", err)
	}

	// Parse the XML metadata into a saml.EntityDescriptor
	var idpMetadata saml.EntityDescriptor
	err = xml.Unmarshal(metadataBytes, &idpMetadata)
	if err != nil {
		log.Fatalf("Failed to parse IdP metadata: %v", err)
	}

	httpsCert := os.Getenv("HTTPS_SERVER_CERT")
	httpsKey := os.Getenv("HTTPS_SERVER_KEY")

	// Load the X.509 key pair
	tlsCert, err := tls.LoadX509KeyPair(httpsCert, httpsKey)

	if err != nil {
		log.Fatalf("failed to load key pair: %v", err)
	}

	// Perform a type assertion to ensure the private key is *rsa.PrivateKey
	rsaPrivateKey, ok := tlsCert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		log.Fatalf("Private key is not of type *rsa.PrivateKey")
	}

	// Parse the PEM-encoded certificate to get an *x509.Certificate
	certBytes, err := os.ReadFile(httpsCert)
	if err != nil {
		log.Fatalf("failed to read certificate file: %v", err)
	}

	block, _ := pem.Decode(certBytes)
	if block == nil {
		log.Fatalf("failed to decode PEM block containing the certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatalf("failed to parse certificate: %v", err)
	}

	// Create the SAML middleware with options
	samlSP, err := samlsp.New(samlsp.Options{
		URL:         *rootURL,
		Key:         rsaPrivateKey,
		Certificate: cert,
		IDPMetadata: &idpMetadata,
	})
	if err != nil {
		log.Fatalf("Error creating SAML Service Provider: %v", err)
	}

	// Public route
	r.GET("/api", IndexHandler)

	// SAML Auth route (protect this route using RequireAccount)
	r.GET("/saml/login", func(c *gin.Context) {
		samlSP.RequireAccount(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next() // Just call Next, no need to touch c.Writer
		})).ServeHTTP(c.Writer, c.Request)
	})

	// SAML callback (after authentication)
	// ACS route - handles the SAML response
	r.POST("/saml/acs", func(c *gin.Context) {
		// Extract the SAML response from the POST request
		req := c.Request
		writer := c.Writer

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

		// samlResponse, err := samlSP.ServiceProvider.ParseResponse(req, []string{})
		// if err != nil {
		// 	http.Error(writer, err.Error(), http.StatusBadRequest)
		// 	return
		// }

		// Get the authenticated user's information from the SAML assertion
		// userEmail := ""
		// for _, statement := range samlResponse.AttributeStatements {
		// 	for _, attr := range statement.Attributes {
		// 		if attr.Name == "email" {
		// 			userEmail = attr.Values[0].Value
		// 			break
		// 		}
		// 	}
		// }

		// Set up a session for the user (simplified)
		// session, err := samlSP.Session.GetSession(r)
		// if err != nil {
		// 	http.Error(w, err.Error(), http.StatusBadRequest)
		// 	return
		// }

		// if session == nil {
		// 	// Create a new session
		// 	session = samlSP.Session.CreateSession(w, r, samlResponse)
		// }

		// sessionIndex := samlResponse.Subject.SubjectConfirmations[0].SubjectConfirmationData.SessionIndex
		// sessionID := samlResponse.Subject.NameID.Value
		// session.Set(r, w, samlSP.ServiceProvider.SessionProvider, samlSP.ServiceProvider.SessionIDPrefix+sessionIndex, sessionID, samlSP.ServiceProvider.SessionLifetime)

		// Redirect to a protected page after login
		c.JSON(http.StatusOK, gin.H{"message": "SAML Response processed successfully"})
	})

	// Protected route using JWT middleware
	r.GET("/api/protected", JWTAuthMiddleware(), ProtectedHandler)

	// Start the server with HTTPS
	err = r.RunTLS(":8080", httpsCert, httpsKey)
	if err != nil {
		log.Fatal(err)
	}
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

// IndexHandler: Public API endpoint
func IndexHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to the SAML + JWT API",
	})
}

// ProtectedHandler: Example of a protected route
func ProtectedHandler(c *gin.Context) {
	userEmail := c.MustGet("email").(string)
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Hello, %s! You have access to this protected resource.", userEmail),
	})
}

// JWTAuthMiddleware: Middleware to protect routes using JWT
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the JWT token from the Authorization header
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			c.Abort()
			return
		}

		// Validate the token
		claims, err := validateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set email in context
		c.Set("email", claims.Email)
		c.Next()
	}
}

// generateJWT: Generates a new JWT token after successful SAML authentication
func generateJWT(email string) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute) // Token valid for 15 minutes

	claims := &Claims{
		Email: email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Create the JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// validateJWT: Validates the JWT token and extracts claims
func validateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}

// Claims represents the structure of the JWT claims
type Claims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

// mustReadFile is a utility function to read key/cert files
func mustReadFile(filePath string) []byte {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", filePath, err)
	}
	return data
}
