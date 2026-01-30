package payment

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	paymentGroup := r.Group("/fees")
	{
		paymentGroup.GET("/", handleGetFees)
	}
}
