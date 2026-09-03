package minigame

import (
	"fmt"
	"os"

	hc "neo/context"
	"neo/core"
)

func init() { core.Default.Register(Tetris) }

var Tetris = &core.Command{
	Name:        "tetris",
	Aliases:     []string{"tetris"},
	Category:    "minigame",
	Description: "Play Tetris in WhatsApp",
	Run: func(ctx *hc.Ctx) {
		htmlBytes, err := os.ReadFile("cmds/minigame/tetris.html")
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
