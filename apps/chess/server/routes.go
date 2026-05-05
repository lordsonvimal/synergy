package server

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	r.GET("/", ShowGameModes)
	r.GET("/ping", PingHandler)
	r.GET("/game/:gameID", ShowGame)
	r.GET("/game/:gameID/events", GameEventsHandler)
	r.POST("/game", CreateGame)
	r.POST("/game/:gameID/select/:square", SelectSquare)
}
