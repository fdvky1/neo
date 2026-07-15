package media

import (
	"context"
	"fmt"
	"os"

	hc "neo/context"
	"neo/core"
	"neo/helper"
)

func init() { core.Default.Register(Sticker) }

var Sticker = &core.Command{
	Name:        "sticker",
	Aliases:     []string{"s", "stiker"},
	Category:    "media",
	Description: "Create sticker from image/video or rewrite existing sticker EXIF",
	Run: func(ctx *hc.Ctx) {
		ctx.React("👌")

		exifPath, cleanupExif := helper.ResolveExif(ctx.Arguments())
		defer func() {
			if cleanupExif {
				os.Remove(exifPath)
			}
		}()

		msg := ctx.Message()
		quoted := hc.GetQuotedMessage(msg)

		var err error
		var stickerData []byte
		var mediaData []byte

		if img := msg.GetImageMessage(); img != nil {
			mediaData, err = ctx.Client().Download(context.Background(), img)
			if err == nil {
				stickerData, err = helper.ConvertImageToSticker(mediaData, exifPath)
			}
		} else if img := quoted.GetImageMessage(); img != nil && quoted != nil {
			mediaData, err = ctx.Client().Download(context.Background(), img)
			if err == nil {
				stickerData, err = helper.ConvertImageToSticker(mediaData, exifPath)
			}
		} else if vid := msg.GetVideoMessage(); vid != nil {
			mediaData, err = ctx.Client().Download(context.Background(), vid)
			if err == nil {
				stickerData, err = helper.ConvertVideoToSticker(mediaData, exifPath)
			}
		} else if vid := quoted.GetVideoMessage(); vid != nil && quoted != nil {
			mediaData, err = ctx.Client().Download(context.Background(), vid)
			if err == nil {
				stickerData, err = helper.ConvertVideoToSticker(mediaData, exifPath)
			}
		} else if smsg := quoted.GetStickerMessage(); smsg != nil && quoted != nil {
			mediaData, err = ctx.Client().Download(context.Background(), smsg)
			if err == nil {
				stickerData, err = helper.RewriteStickerExif(mediaData, exifPath)
			}
		} else {
			ctx.Reply("error: reply to image/video/sticker or send with caption")
			return
		}

		if err != nil {
			ctx.Reply(fmt.Sprintf("error: %v", err))
			return
		}

		ctx.SendSticker(stickerData)
	},
}
