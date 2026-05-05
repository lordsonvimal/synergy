package server

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	r.GET("/", ShowGameModes)
	r.GET("/game/:gameID", ShowGame)
	r.POST("/game", CreateGame)
	r.POST("/game/:gameID/select/:square", SelectSquare)
}
