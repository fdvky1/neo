package minigame

import (
	"fmt"
	"os"

	hc "neo/context"
	"neo/core"
)

func init() { core.Default.Register(Minesweeper) }

var Minesweeper = &core.Command{
	Name:        "minesweeper",
	Aliases:     []string{"minesweeper", "ms"},
	Category:    "minigame",
	Description: "Play Minesweeper in WhatsApp",
	Run: func(ctx *hc.Ctx) {
		htmlBytes, err := os.ReadFile("cmds/minigame/minesweeper.html")
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
