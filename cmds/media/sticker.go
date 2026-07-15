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
		var rawWebp []byte
		var mediaData []byte

		if img := msg.GetImageMessage(); img != nil {
			mediaData, err = ctx.Client().Download(context.Background(), img)
			if err == nil {
				rawWebp, err = helper.ConvertImageToWebp(mediaData)
			}
		} else if img := quoted.GetImageMessage(); img != nil && quoted != nil {
			mediaData, err = ctx.Client().Download(context.Background(), img)
			if err == nil {
				rawWebp, err = helper.ConvertImageToWebp(mediaData)
			}
		} else if vid := msg.GetVideoMessage(); vid != nil {
			mediaData, err = ctx.Client().Download(context.Background(), vid)
			if err == nil {
				rawWebp, err = helper.ConvertVideoToWebp(mediaData)
			}
		} else if vid := quoted.GetVideoMessage(); vid != nil && quoted != nil {
			mediaData, err = ctx.Client().Download(context.Background(), vid)
			if err == nil {
				rawWebp, err = helper.ConvertVideoToWebp(mediaData)
			}
		} else if smsg := quoted.GetStickerMessage(); smsg != nil && quoted != nil {
			rawWebp, err = ctx.Client().Download(context.Background(), smsg)
		} else {
			ctx.Reply("error: reply to image/video/sticker or send with caption")
			return
		}

		if err != nil {
			ctx.Reply(fmt.Sprintf("error processing media: %v", err))
			return
		}

		finalSticker, err := helper.InjectExif(rawWebp, exifPath)
		if err != nil {
			ctx.Reply(fmt.Sprintf("error injecting EXIF: %v", err))
			return
		}

		ctx.SendSticker(finalSticker)
	},
}
