package minigame

import (
	"fmt"
	"os"

	hc "neo/context"
	"neo/core"
)

func init() { core.Default.Register(Game2048) }

var Game2048 = &core.Command{
	Name:        "2048",
	Aliases:     []string{"2048"},
	Category:    "minigame",
	Description: "Play 2048 in WhatsApp",
	Run: func(ctx *hc.Ctx) {
		htmlBytes, err := os.ReadFile("cmds/minigame/2048.html")
		if err != nil {
			ctx.Reply(fmt.Sprintf("Failed to read HTML file: %v", err))
			return
		}
		_, err = ctx.SendInteractiveHTML(string(htmlBytes))
		if err != nil {
			ctx.Reply(fmt.Sprintf("Failed to send game: %v", err))
		}
	},
}
