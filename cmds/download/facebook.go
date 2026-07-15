package download

import (
	"fmt"
	"regexp"

	"neo/api"
	hc "neo/context"
	"neo/core"
	"neo/helper"
)

type facebookResult struct {
	HD        string `json:"hd"`
	SD        string `json:"sd"`
	Thumbnail string `json:"thumbnail"`
}

func init() { core.Default.Register(Facebook) }

var Facebook = &core.Command{
	Name:        "facebook",
	Aliases:     []string{"fb"},
	Category:    "download",
	Description: "Download Facebook video",
	Run: func(ctx *hc.Ctx) {
		url := ctx.Arg(0)
		if url == "" {
			ctx.Reply("Usage: !fb <url>")
			return
		}

		fbRegex := regexp.MustCompile(`(?i)(?:facebook\.com|fb\.com|fb\.watch)\/(?:share\/[pr]\/|reel\/|watch\/?\?v=|.*\/videos\/)?([a-zA-Z0-9_\-]+)`)
		if !fbRegex.MatchString(url) {
			ctx.Reply("Invalid Facebook URL")
			return
		}

		ctx.Send("Downloading...")
		var res facebookResult
		if err := api.Default().Get("/fb", map[string]string{"url": url}, &res); err != nil {
			ctx.Reply(fmt.Sprintf("Failed to fetch: %v", err))
			return
		}

		downloadURL := res.HD
		if downloadURL == "" {
			downloadURL = res.SD
		}
		if downloadURL == "" {
			ctx.Reply("No video found")
			return
		}

		data, mime, err := api.Download(downloadURL)
		if err != nil {
			ctx.Reply(fmt.Sprintf("Download failed: %v", err))
			return
		}

		helper.SendMedia(ctx, data, mime, "")
	},
}
