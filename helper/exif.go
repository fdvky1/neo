package helper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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

func ResolveExif(args []string) (string, bool) {
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
