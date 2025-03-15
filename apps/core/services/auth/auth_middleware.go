package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware validates the JWT token from cookies
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get JWT from HttpOnly Cookie
		tokenString, err := c.Cookie("access_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Parse and validate JWT
		claims, err := authenticator.Authenticate(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Check expiration
		if claims.ExpiresAt.Time.Before(time.Now()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			return
		}

		// Attach userID to context for future handlers
		c.Set("user_id", claims.ID)

		fmt.Println("User token validation successful")
		// Proceed with request
		c.Next()
	}
}
