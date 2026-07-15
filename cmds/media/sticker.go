package media

import (
	"context"
	"fmt"
	"os"
	"strings"

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

		finalSticker, err := helper.InjectExif(rawWebp, packName, publisher)
		if err != nil {
			ctx.Reply(fmt.Sprintf("error injecting EXIF: %v", err))
			return
		}

		ctx.SendSticker(finalSticker)
	},
}
