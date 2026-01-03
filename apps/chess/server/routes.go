package server

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	r.GET("/", ShowGameModes)
	r.POST("/game", CreateGame)
	r.POST("/game/:gameID/select", SelectSquare)
	r.GET("/game/live/:gameID", LiveChessUpdates)
}
