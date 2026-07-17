package download

import (
	"fmt"
	"regexp"

	"neo/api"
	hc "neo/context"
	"neo/core"
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

		for _, video := range res.Videos {
			data, _, err := api.Download(video.Src)
			if err != nil {
				ctx.Reply(fmt.Sprintf("Download failed: %v", err))
				return
			}
			// if i == 0 {
			// 	ctx.SendVideo(data, fmt.Sprintf("Instagram Video\n%s", url))
			// } else {
			ctx.SendVideo(data, "")
			// }
		}

		for _, imgURL := range res.Images {
			data, _, err := api.Download(imgURL)
			if err != nil {
				ctx.Reply(fmt.Sprintf("Download failed: %v", err))
				return
			}
			// if i == 0 {
			// 	ctx.SendImage(data, fmt.Sprintf("Instagram Image\n%s", url))
			// } else {
			ctx.SendImage(data, "")
			// }
		}
	},
}
