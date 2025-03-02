package sso

import (
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
)

// Middleware to add samlSP to the context
func SamlSPMiddleware(samlSP *samlsp.Middleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("samlSP", samlSP)
		c.Next()
	}
}
