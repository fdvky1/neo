package download

import (
	"fmt"
	"regexp"

	"neo/api"
	hc "neo/context"
	"neo/core"
	"neo/helper"
)

type instagramVideo struct {
	Src       string `json:"src"`
	Thumbnail string `json:"thumbnail"`
}

type instagramResult struct {
	Images []string         `json:"images"`
	Videos []instagramVideo `json:"videos"`
}

func init() { core.Default.Register(Instagram) }

var Instagram = &core.Command{
	Name:        "instagram",
	Aliases:     []string{"ig"},
	Category:    "download",
	Description: "Download Instagram media",
	Run: func(ctx *hc.Ctx) {
		url := ctx.Arg(0)
		if url == "" {
			ctx.Reply("Usage: !ig <url>")
			return
		}

		igRegex := regexp.MustCompile(`(?i)(?:instagram\.com|instagr\.am)\/(?:p|reel|tv)\/([a-zA-Z0-9_\-]+)`)
		if !igRegex.MatchString(url) {
			ctx.Reply("Invalid Instagram URL")
			return
		}

		ctx.Send("Downloading...")
		var res instagramResult
		if err := api.Default().Get("/ig", map[string]string{"url": url}, &res); err != nil {
			ctx.Reply(fmt.Sprintf("Failed to fetch: %v", err))
			return
		}

		mediaURL := ""
		if len(res.Videos) > 0 {
			mediaURL = res.Videos[0].Src
		} else if len(res.Images) > 0 {
			mediaURL = res.Images[0]
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

		helper.SendMedia(ctx, data, mime, "")
	},
}
