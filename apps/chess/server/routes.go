package server

import "github.com/gin-gonic/gin"

func InitRoutes(r *gin.Engine) {
	r.GET("/", ShowGameModes)
	r.GET("/ping", PingHandler)

	// Solo (single-player, controls both colours).
	r.GET("/solo/:gameID", ShowSolo)
	r.GET("/solo/:gameID/events", SoloEventsHandler)
	r.GET("/solo/:gameID/board-at/:halfMoveIdx", BoardAtHistoryHandler)
	r.POST("/solo", CreateSolo)
	r.POST("/solo/:gameID/select/:square", SoloSelectSquare)
	r.POST("/solo/:gameID/history/navigate", NavigateHistoryHandler)

	// Online play routes.
	r.POST("/play", CreatePlay)
	r.GET("/play/:gameID", ShowPlayGame)
	r.GET("/play/:gameID/events", PlayEventsHandler)
	r.GET("/play/:gameID/board-at/:halfMoveIdx", BoardAtHistoryHandler)
	r.POST("/play/:gameID/select/:square", PlaySelectSquare)
	r.POST("/play/:gameID/history/navigate", NavigateHistoryHandler)
	r.POST("/play/:gameID/claim-victory", ClaimVictory)
	r.POST("/play/:gameID/rematch", ProposeRematch)
	r.POST("/play/:gameID/rematch/accept", AcceptRematch)
	r.POST("/play/:gameID/rematch/decline", DeclineRematch)
	r.POST("/play/:gameID/abandon", AbandonPlayGame)
}
