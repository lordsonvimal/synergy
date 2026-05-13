package server

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	r.GET("/", ShowGameModes)
	r.GET("/ping", PingHandler)
	r.GET("/game/:gameID", ShowGame)
	r.GET("/game/:gameID/events", GameEventsHandler)
	r.GET("/game/:gameID/board-at/:halfMoveIdx", BoardAtHistoryHandler)
	r.POST("/game", CreateGame)
	r.POST("/game/:gameID/select/:square", SelectSquare)
	r.POST("/game/:gameID/history/navigate", NavigateHistoryHandler)

	// Online play routes.
	r.POST("/play", CreatePlay)
	r.GET("/play/:gameID", ShowPlayGame)
	r.GET("/play/:gameID/events", PlayEventsHandler)
	r.POST("/play/:gameID/select/:square", PlaySelectSquare)
	r.POST("/play/:gameID/claim-victory", ClaimVictory)
	r.POST("/play/:gameID/rematch", ProposeRematch)
	r.POST("/play/:gameID/rematch/accept", AcceptRematch)
	r.POST("/play/:gameID/rematch/decline", DeclineRematch)
	r.POST("/play/:gameID/abandon", AbandonPlayGame)
}
