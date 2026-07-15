package download

import (
	"fmt"
	"regexp"

	"neo/api"
	hc "neo/context"
	"neo/core"
	"neo/helper"
)

type tiktokResult struct {
	Description string   `json:"description"`
	ImageURL    []string `json:"image_url"`
	Username    string   `json:"username"`
	VideoURL    []string `json:"video_url"`
}

func init() { core.Default.Register(TikTok) }

var TikTok = &core.Command{
	Name:        "tiktok",
	Aliases:     []string{"tt", "tk"},
	Category:    "download",
	Description: "Download TikTok video",
	Run: func(ctx *hc.Ctx) {
		url := ctx.Arg(0)
		if url == "" {
			ctx.Reply("Usage: !tiktok <url>")
			return
		}

		ttRegex := regexp.MustCompile(`(?i)(?:tiktok\.com)\/(?:@[\w.-]+\/video\/|v\/|t\/)([0-9]+)`)
		if !ttRegex.MatchString(url) {
			ctx.Reply("Invalid TikTok URL")
			return
		}

		ctx.Send("Downloading...")
		var res tiktokResult
		if err := api.Default().Get("/tiktok", map[string]string{"url": url}, &res); err != nil {
			ctx.Reply(fmt.Sprintf("Failed to fetch: %v", err))
			return
		}

		mediaURL := ""
		if len(res.VideoURL) > 0 {
			mediaURL = res.VideoURL[0]
		} else if len(res.ImageURL) > 0 {
			mediaURL = res.ImageURL[0]
		}
		if mediaURL == "" {
			ctx.Reply("No media found")
			return
		}

		data, mime, err := api.Download(mediaURL)
		if err != nil {
			ctx.Reply(fmt.Sprintf("Download failed: %v", err))
			return
		}

		caption := fmt.Sprintf("@%s\n%s", res.Username, res.Description)
		helper.SendMedia(ctx, data, mime, caption)
	},
}
