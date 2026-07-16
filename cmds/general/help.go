package general

import (
	"fmt"
	"strings"
	"sync"
	"time"

	hc "neo/context"
	"neo/core"
)

func init() {
	core.Default.Register(Help)
}

var (
	cachedMenu string
	menuOnce   sync.Once
)

// buildMenuCache generates the static portion of the help menu
func buildMenuCache(prefix string) {
	var sb strings.Builder

	cmdMap := make(map[string][]string)
	for _, cmd := range core.Default.GetCommands() {
		cat := cmd.Category
		if cat == "" {
			cat = "general"
		}
		cmdMap[cat] = append(cmdMap[cat], cmd.Name)
	}

	for cat, cmds := range cmdMap {
		sb.WriteString(fmt.Sprintf("\n*📃%s*\n", strings.ToUpper(cat)))
		for i, cmdName := range cmds {
			branch := "├"
			if i == len(cmds)-1 {
				branch = "└"
			}
			sb.WriteString(fmt.Sprintf("%s %s%s\n", branch, prefix, cmdName))
		}
	}

	cachedMenu = sb.String()
}

var Help = &core.Command{
	Name:        "help",
	Aliases:     []string{"?", "h", "menu"},
	Category:    "general",
	Description: "Shows the bot's features and commands",
	Run: func(ctx *hc.Ctx) {
		loc, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			loc = time.FixedZone("UTC+7", 7*3600)
		}

		now := time.Now().In(loc)
		hour := now.Hour()

		var greeting string
		switch {
		case hour >= 4 && hour < 12:
			greeting = "Ohayō"
		case hour >= 12 && hour < 18:
			greeting = "Konnichiwa"
		default:
			greeting = "Konbanwa"
		}

		pushName := ctx.Event().Info.PushName
		if pushName == "" {
			pushName = "User"
		}

		prefix := ctx.Prefix()
		if prefix == "" {
			prefix = "."
		}

		// Calculate the static command layout part only once
		menuOnce.Do(func() {
			buildMenuCache(prefix)
		})

		header := fmt.Sprintf("%s *%s*👋\n📬 Need help? Here are all of my commands\n", greeting, pushName)

		ctx.Reply(header + cachedMenu)
	},
}
