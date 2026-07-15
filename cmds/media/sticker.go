package media

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
	hc "neo/context"
	"neo/core"
	"neo/helper"

	waProto "go.mau.fi/whatsmeow/binary/proto"
)

func cmdStickerImage(ctx *hc.Ctx, img *waProto.ImageMessage, exifPath string, webpPath string) error {
	data, err := ctx.Client().Download(context.Background(), img)
	if err != nil {
		return err
	}
	
	convertCmd := exec.Command("convert", "-", "-resize", "512x512", "-background", "none", "-compose", "Copy", "-gravity", "center", "-extent", "512x512", "-quality", "100", "png:-")
	convertCmd.Stdin = bytes.NewReader(data)
	var convertBuf bytes.Buffer
	convertCmd.Stdout = &convertBuf
	if err := convertCmd.Run(); err != nil {
		return fmt.Errorf("convert failed: %w", err)
	}

	cwebpCmd := exec.Command("cwebp", "-quiet", "-mt", "-exact", "-q", "100", "-m", "6", "-alpha_q", "100", "-o", "-", "--", "-")
	cwebpCmd.Stdin = &convertBuf
	var cwebpBuf bytes.Buffer
	cwebpCmd.Stdout = &cwebpBuf
	if err := cwebpCmd.Run(); err != nil {
		return fmt.Errorf("cwebp failed: %w", err)
	}

	tmpWebp := webpPath + ".tmp.webp"
	if err := os.WriteFile(tmpWebp, cwebpBuf.Bytes(), 0644); err != nil {
		return err
	}
	defer os.Remove(tmpWebp)

	webpmuxCmd := exec.Command("webpmux", "-set", "exif", exifPath, tmpWebp, "-o", webpPath)
	if err := webpmuxCmd.Run(); err != nil {
		return fmt.Errorf("webpmux failed: %w", err)
	}
	
	return nil
}

func cmdStickerVideo(ctx *hc.Ctx, video *waProto.VideoMessage, exifPath string, webpPath string) error {
	data, err := ctx.Client().Download(context.Background(), video)
	if err != nil {
		return err
	}
	
	tmpWebp := webpPath + ".tmp.webp"
	defer os.Remove(tmpWebp)
	
	ffmpegCmd := exec.Command("ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0", "-f", "mp4",
		"-ss", "00:00:00", "-t", "00:00:15",
		"-vf", "fps=10,scale=720:-1:flags=lanczos:force_original_aspect_ratio=increase,crop=512:512,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=#00000000,setsar=1",
		"-compression_level", "6",
		"-q:v", "60", "-loop", "0",
		"-preset", "picture", "-an", "-fps_mode", "auto",
		"-f", "webp", tmpWebp,
	)
	ffmpegCmd.Stdin = bytes.NewReader(data)
	if err := ffmpegCmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}
	
	webpmuxCmd := exec.Command("webpmux", "-set", "exif", exifPath, tmpWebp, "-o", webpPath)
	if err := webpmuxCmd.Run(); err != nil {
		return fmt.Errorf("webpmux failed: %w", err)
	}
	
	return nil
}

func cmdStickerRewrite(ctx *hc.Ctx, sticker *waProto.StickerMessage, exifPath string, webpPath string) error {
	data, err := ctx.Client().Download(context.Background(), sticker)
	if err != nil {
		return err
	}
	
	tmpWebp := webpPath + ".tmp.webp"
	if err := os.WriteFile(tmpWebp, data, 0644); err != nil {
		return err
	}
	defer os.Remove(tmpWebp)

	webpmuxCmd := exec.Command("webpmux", "-set", "exif", exifPath, tmpWebp, "-o", webpPath)
	if err := webpmuxCmd.Run(); err != nil {
		return fmt.Errorf("webpmux failed: %w", err)
	}
	
	return nil
}

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

		webpPath := filepath.Join("temp", fmt.Sprintf("%s.webp", uuid.New().String()))
		defer os.Remove(webpPath)

		msg := ctx.Message()
		quoted := hc.GetQuotedMessage(msg)

		var err error
		if msg.GetImageMessage() != nil {
			err = cmdStickerImage(ctx, msg.GetImageMessage(), exifPath, webpPath)
		} else if quoted != nil && quoted.GetImageMessage() != nil {
			err = cmdStickerImage(ctx, quoted.GetImageMessage(), exifPath, webpPath)
		} else if msg.GetVideoMessage() != nil {
			err = cmdStickerVideo(ctx, msg.GetVideoMessage(), exifPath, webpPath)
		} else if quoted != nil && quoted.GetVideoMessage() != nil {
			err = cmdStickerVideo(ctx, quoted.GetVideoMessage(), exifPath, webpPath)
		} else if quoted != nil && quoted.GetStickerMessage() != nil {
			err = cmdStickerRewrite(ctx, quoted.GetStickerMessage(), exifPath, webpPath)
		} else {
			ctx.Reply("error: reply to image/video/sticker or send with caption")
			return
		}

		if err != nil {
			ctx.Reply(fmt.Sprintf("error: %v", err))
			return
		}

		stickerData, err := os.ReadFile(webpPath)
		if err != nil {
			ctx.Reply(fmt.Sprintf("error: failed to read temp sticker: %v", err))
			return
		}

		ctx.SendSticker(stickerData)
	},
}
