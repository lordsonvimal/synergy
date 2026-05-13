package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/chess/game"
	"github.com/lordsonvimal/synergy/apps/chess/ui/components"
	"github.com/lordsonvimal/synergy/apps/chess/ui/ui_store"
	"github.com/starfederation/datastar-go/datastar"
)

// broadcastBoard sends an updated board + notation panel HTML and signals patch
// via DataStar SSE to the requesting player. Route prefix is derived from PlayMeta.
func broadcastBoard(c *gin.Context, g *game.Game, signals *ui_store.ChessBoardSignals) error {
	sse := datastar.NewSSE(c.Writer, c.Request)
	ctx := c.Request.Context()

	routePrefix := routePrefixFor(g)
	buf := new(strings.Builder)

	components.RenderChessBoard(g, routePrefix).Render(ctx, buf)
	sse.PatchElements(buf.String())

	buf.Reset()
	components.MoveNotationPanel(g.ID, g.History).Render(ctx, buf)
	sse.PatchElements(buf.String())

	return broadcastSignals(c, signals)
}

// broadcastSignals sends a DataStar signal patch via SSE to the requesting player.
func broadcastSignals(c *gin.Context, signals *ui_store.ChessBoardSignals) error {
	sse := datastar.NewSSE(c.Writer, c.Request)

	b, err := json.Marshal(signals)
	if err != nil {
		return err
	}

	sse.PatchSignals(b)
	return nil
}

// hubBroadcastBoard pushes board + notation + signals as DataStar SSE frames to
// every hub subscriber. Called after a move is applied in play mode so the
// opponent's board updates in real time via the /play/:id/board-stream endpoint.
func hubBroadcastBoard(ctx context.Context, g *game.Game, signals *ui_store.ChessBoardSignals) {
	var all []byte

	buf := new(strings.Builder)
	components.RenderChessBoard(g, "/play").Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	buf.Reset()
	components.MoveNotationPanel(g.ID, g.History).Render(ctx, buf)
	all = append(all, datastarPatchElementsFrame(buf.String())...)

	if b, err := json.Marshal(signals); err == nil {
		all = append(all, datastarPatchSignalsFrame(b)...)
	}

	g.Hub.Broadcast(all)
}

// routePrefixFor returns "/play" for online games, "/game" for solo games.
func routePrefixFor(g *game.Game) string {
	if g.PlayMeta != nil {
		return "/play"
	}
	return "/game"
}

// datastarPatchElementsFrame formats an HTML string as a DataStar patch-elements SSE frame.
func datastarPatchElementsFrame(html string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "event: %s\n", datastar.EventTypePatchElements)
	for _, line := range strings.Split(html, "\n") {
		fmt.Fprintf(&buf, "data: %s%s\n", datastar.ElementsDatalineLiteral, line)
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}

// datastarPatchSignalsFrame formats JSON as a DataStar patch-signals SSE frame.
func datastarPatchSignalsFrame(b []byte) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "event: %s\n", datastar.EventTypePatchSignals)
	for _, line := range strings.Split(string(b), "\n") {
		fmt.Fprintf(&buf, "data: %s%s\n", datastar.SignalsDatalineLiteral, line)
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}
