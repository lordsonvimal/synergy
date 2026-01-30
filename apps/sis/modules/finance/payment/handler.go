package payment

import "github.com/gin-gonic/gin"

// Table: payments
func handleGetFees(c *gin.Context) {
	RenderFeesPage().Render(c, c.Writer)
}
