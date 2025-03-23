package db

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ContainerMiddleware injects the repository container into the Gin context
func ContainerMiddleware() gin.HandlerFunc {
	container := GetContainer()
	return func(c *gin.Context) {
		// Add the container to the Gin context
		c.Set("container", container)

		// Inject into Go request context
		reqCtx := context.WithValue(c.Request.Context(), ContainerKey, container)
		c.Request = c.Request.WithContext(reqCtx)

		c.Next()
	}
}
