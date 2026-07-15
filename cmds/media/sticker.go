package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	hc "neo/context"

	waProto "go.mau.fi/whatsmeow/binary/proto"
)

type exifData struct {
	StickerPackId        string `json:"sticker-pack-id"`
	StickerPackName      string `json:"sticker-pack-name"`
	StickerPackPublisher string `json:"sticker-pack-publisher"`
	IsAvatarSticker      int    `json:"is-avatar-sticker"`
}

type webpExif struct {
	AppId string   `json:"app-id"`
	Ext   exifData `json:"ext"`
}

func getOrCreateDefaultExif() string {
	defaultExifPath := "temp/default.exif"

	// Create temp directory if it doesn't exist
	if err := os.MkdirAll("temp", 0755); err != nil {
		log.Printf("Failed to create temp directory: %v", err)
		return ""
	}

	if _, err := os.Stat(defaultExifPath); os.IsNotExist(err) {
		packName := os.Getenv("STICKER_NAME")
		if packName == "" {
			packName = "Bot"
		}

		publisher := os.Getenv("STICKER_PUBLISHER")
		if publisher == "" {
			publisher = "Bot"
		}

		exifObj := webpExif{
			AppId: "com.whatsapp.app",
			Ext: exifData{
				StickerPackId:        "com.whatsapp.app",
				StickerPackName:      packName,
				StickerPackPublisher: publisher,
				IsAvatarSticker:      0,
			},
		}

		exifJson, err := json.Marshal(exifObj)
		if err != nil {
			log.Printf("Failed to marshal default EXIF: %v", err)
			return ""
		}

		// WebP EXIF JSON string representation inside the file
		exifContent := append([]byte{0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x01, 0x00, 0x41, 0x57, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00}, exifJson...)

		if err := os.WriteFile(defaultExifPath, exifContent, 0644); err != nil {
			log.Printf("Failed to write default EXIF: %v", err)
			return ""
		}
	}
	return defaultExifPath
}

func resolveExif(args []string) (string, bool) {
	if len(args) == 0 {
		return getOrCreateDefaultExif(), false
	}

	argStr := strings.Join(args, " ")
	parts := strings.Split(argStr, "|")

	packName := strings.TrimSpace(parts[0])
	publisher := "Bot"
	if len(parts) > 1 {
		publisher = strings.TrimSpace(parts[1])
	}

	if packName == "" && publisher == "Bot" {
		return getOrCreateDefaultExif(), false
	}

	exifObj := webpExif{
		AppId: "com.whatsapp.app",
		Ext: exifData{
			StickerPackId:        "com.whatsapp.app",
			StickerPackName:      packName,
			StickerPackPublisher: publisher,
			IsAvatarSticker:      0,
		},
	}

	exifJson, err := json.Marshal(exifObj)
	if err != nil {
		log.Printf("Failed to marshal dynamic EXIF: %v", err)
		return getOrCreateDefaultExif(), false
	}

	exifContent := append([]byte{0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x01, 0x00, 0x41, 0x57, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00}, exifJson...)

	id := uuid.New().String()
	tempPath := filepath.Join("temp", fmt.Sprintf("%s.exif", id))

	if err := os.WriteFile(tempPath, exifContent, 0644); err != nil {
		log.Printf("Failed to write dynamic EXIF: %v", err)
		return getOrCreateDefaultExif(), false
	}

	return tempPath, true
}

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
