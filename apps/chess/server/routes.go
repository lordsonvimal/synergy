package server

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	r.GET("/", ShowGameModes)
	r.POST("/game", CreateGame)
	r.GET("/live/chess/:gameID", LiveChessUpdates)
}
