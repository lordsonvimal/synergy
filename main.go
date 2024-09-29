package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/crewjam/saml/samlsp"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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
	rootURL, err := url.Parse("http://localhost:8080") // Base URL for your service provider
	if err != nil {
		log.Fatalf("Invalid root URL: %v", err)
	}

	// Fetch IdP metadata (correct way)
	idpMetadataURL, err := url.Parse(os.Getenv("SAML_IDP_METADATA_URL"))
	if err != nil {
		log.Fatalf("Invalid IdP Metadata URL: %v", err)
	}

	// Create a context and HTTP client for fetching metadata
	ctx := context.Background()
	httpClient := &http.Client{}

	// Fetch and parse IdP metadata
	idpMetadata, err := samlsp.FetchMetadata(ctx, httpClient, *idpMetadataURL)
	if err != nil {
		log.Fatalf("Error fetching IdP metadata: %v", err)
	}

	// Create the SAML middleware with options
	samlMiddleware, err := samlsp.New(samlsp.Options{
		URL:         *rootURL,
		Key:         mustReadFile("certs/key.pem"),
		Certificate: mustReadFile("certs/cert.pem"),
		IDPMetadata: idpMetadata, // Use the fetched metadata here
	})
	if err != nil {
		log.Fatalf("Error creating SAML Service Provider: %v", err)
	}

	// Public route
	r.GET("/api", IndexHandler)

	// SAML Auth route (protect this route using RequireAccount)
	r.GET("/saml/login", func(c *gin.Context) {
		samlMiddleware.RequireAccount(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next() // Just call Next, no need to touch c.Writer
		})).ServeHTTP(c.Writer, c.Request)
	})

	// SAML callback (after authentication)
	r.GET("/saml/callback", func(c *gin.Context) {
		samlMiddleware.RequireAccount(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)

		// Extract SAML attributes
		session := samlsp.SessionFromContext(c.Request.Context())
		if session == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			return
		}

		// Extract the user's email or any attribute you want to use
		attributes := session.(samlsp.JWTSessionClaims).StandardClaims
		userEmail := attributes.Subject

		// Generate JWT for the authenticated user
		jwtToken, err := generateJWT(userEmail)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
			return
		}

		// Return the JWT token
		c.JSON(http.StatusOK, gin.H{
			"token": jwtToken,
		})
	})

	// Protected route using JWT middleware
	r.GET("/api/protected", JWTAuthMiddleware(), ProtectedHandler)

	httpsCert := os.Getenv("HTTPS_SERVER_CERT")
	httpsKey := os.Getenv("HTTPS_SERVER_KEY")

	// Start the server with HTTPS
	err = r.RunTLS(":8080", httpsCert, httpsKey)
	if err != nil {
		log.Fatal(err)
	}
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
