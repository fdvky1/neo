package misc

import (
	"fmt"
	"time"

	hc "neo/context"
	"neo/core"
)

func init() { core.Default.Register(Ping) }

var Ping = &core.Command{
	Name:        "ping",
	Aliases:     []string{"test", "tes", "p"},
	Category:    "general",
	Description: "Check bot response time",
	// Permissions: []core.Permission{core.PermissionGroupOnly},
	Run: func(ctx *hc.Ctx) {
		t := ctx.Event().Info.Timestamp
		speed := time.Since(t).Milliseconds()
		ctx.Reply(fmt.Sprintf("Pong! 🏓\n\nSpeed: %d ms", speed))
	},
}
