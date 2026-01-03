package server

import "github.com/gin-gonic/gin"

type SquareRequest struct {
	Square uint8 `json:"square"`
}

func parseSquare(c *gin.Context) uint8 {
	var req SquareRequest

	// 1. Try to bind the JSON body to the struct
	if err := c.ShouldBindJSON(&req); err != nil {
		// Log error or handle cases where square is 0/missing
		return 0
	}

	return req.Square
}
