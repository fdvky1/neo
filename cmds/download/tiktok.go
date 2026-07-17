package download

import (
	"fmt"
	"regexp"

	"neo/api"
	hc "neo/context"
	"neo/core"
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

		ttRegex := regexp.MustCompile(`(?i)^https?://(?:[\w-]+\.)?tiktok\.com(?:/.*)?$`)
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

		caption := fmt.Sprintf("@%s\n%s", res.Username, res.Description)
		if len(res.VideoURL) > 0 {
			data, _, err := api.Download(res.VideoURL[0])
			if err != nil {
				ctx.Reply(fmt.Sprintf("Download failed: %v", err))
				return
			}
			ctx.SendVideo(data, caption)
		} else if len(res.ImageURL) > 0 {
			go func() {
				for _, imgURL := range res.ImageURL {
					data, _, err := api.Download(imgURL)
					if err != nil {
						ctx.Reply(fmt.Sprintf("Download failed: %v", err))
						return
					}
					ctx.SendImage(data, caption)
				}
			}()
		} else {
			ctx.Reply("No media found")
			return
		}
	},
}
