package media

import (
	"context"
	"fmt"
	"os"
	"strings"

	hc "neo/context"
	"neo/core"
	"neo/helper"

	"go.mau.fi/whatsmeow"
)

func init() { core.Default.Register(Sticker) }

var Sticker = &core.Command{
	Name:        "sticker",
	Aliases:     []string{"s", "stiker"},
	Category:    "media",
	Description: "Create sticker from image/video or rewrite existing sticker EXIF",
	Run: func(ctx *hc.Ctx) {
		// go ctx.React("👌")

		var packName, publisher string
		args := ctx.Arguments()

		packName = os.Getenv("STICKER_NAME")
		if packName == "" {
			packName = "Bot"
		}
		publisher = os.Getenv("STICKER_PUBLISHER")
		if publisher == "" {
			publisher = "Bot"
		}

		if len(args) > 0 {
			argsStr := strings.Join(args, " ")
			parts := strings.Split(argsStr, "|")
			packName = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				publisher = strings.TrimSpace(parts[1])
			} else {
				publisher = ""
			}
		}

		msg := ctx.Message()
		quoted := hc.GetQuotedMessage(msg)

		var err error
		var rawWebp []byte
		var mediaData []byte
		var downloadable whatsmeow.DownloadableMessage
		var mediaType string

		switch {
		case msg.GetImageMessage() != nil:
			downloadable, mediaType = msg.GetImageMessage(), "image"
		case quoted != nil && quoted.GetImageMessage() != nil:
			downloadable, mediaType = quoted.GetImageMessage(), "image"
		case msg.GetVideoMessage() != nil:
			downloadable, mediaType = msg.GetVideoMessage(), "video"
		case quoted != nil && quoted.GetVideoMessage() != nil:
			downloadable, mediaType = quoted.GetVideoMessage(), "video"
		case quoted != nil && quoted.GetStickerMessage() != nil:
			downloadable, mediaType = quoted.GetStickerMessage(), "sticker"
		}

		if downloadable != nil {
			mediaData, err = ctx.Client().Download(context.Background(), downloadable)
			if err == nil {
				switch mediaType {
				case "image":
					rawWebp, err = helper.ConvertImageToWebp(mediaData)
				case "video":
					rawWebp, err = helper.ConvertVideoToWebp(mediaData)
				case "sticker":
					rawWebp = mediaData
				}
			}
		} else {
			ctx.Reply("error: reply to image/video/sticker or send with caption")
			return
		}

		if err != nil {
			ctx.Reply(fmt.Sprintf("error processing media: %v", err))
			return
		}

		finalSticker, err := helper.InjectExif(rawWebp, packName, publisher)
		if err != nil {
			ctx.Reply(fmt.Sprintf("error injecting EXIF: %v", err))
			return
		}

		ctx.SendSticker(finalSticker)
	},
}
