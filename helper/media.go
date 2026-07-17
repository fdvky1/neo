package helper

import (
	"bytes"
	"os/exec"
	"strings"

	hc "neo/context"
)

func ConvertToMP3(data []byte) ([]byte, error) {
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "mp3", "-ab", "128k", "pipe:1")
	cmd.Stdin = bytes.NewReader(data)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func SendMedia(ctx *hc.Ctx, data []byte, mime, caption string) {
	switch {
	case strings.HasPrefix(mime, "video/mp4"):
		ctx.SendVideo(data, caption)
	case strings.HasPrefix(mime, "image/"):
		ctx.SendImage(data, caption)
	case strings.HasPrefix(mime, "audio/"):
		ctx.SendAudio(data)
	default:
		ctx.SendDocument(data, "media", "media"+extByMIME(mime))
	}
}

func extByMIME(mime string) string {
	switch {
	case strings.HasPrefix(mime, "video/mp4"):
		return ".mp4"
	case strings.HasPrefix(mime, "video/webm"):
		return ".webm"
	case strings.HasPrefix(mime, "video/quicktime"):
		return ".mov"
	case strings.HasPrefix(mime, "video/x-matroska"):
		return ".mkv"
	case strings.HasPrefix(mime, "video/"):
		return ".video"
	case strings.HasPrefix(mime, "image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(mime, "image/png"):
		return ".png"
	case strings.HasPrefix(mime, "image/gif"):
		return ".gif"
	case strings.HasPrefix(mime, "image/webp"):
		return ".webp"
	case strings.HasPrefix(mime, "audio/mpeg"), strings.HasPrefix(mime, "audio/mp3"):
		return ".mp3"
	case strings.HasPrefix(mime, "audio/"):
		return ".m4a"
	default:
		return ".bin"
	}
}
