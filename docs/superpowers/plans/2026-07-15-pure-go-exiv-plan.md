# Pure-Go WebP EXIF RIFF Injector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Modify `helper/exif.go` and `helper/media_convert.go` to inject EXIF payloads directly into WebP RIFF byte slices entirely in Go memory, replacing `webpmux`.

**Architecture:** We will implement an `InjectExif(webpBytes []byte, packName, publisher string) ([]byte, error)` function inside `helper/exif.go` to handle raw RIFF parsing in place, and strip out the old Temp File logic. Webpmux invocation in `helper/media_convert.go` will be removed. 

**Tech Stack:** Go (encoding/binary)

## Global Constraints
- Target files: `helper/exif.go` and `helper/media_convert.go`
- Do not use CGO or any unstated 3rd party package for RIFF handling. Implement manually using `encoding/binary`.
- Format must be `VP8X` with the EXIF flag bit activated.

---

### Task 1: Clean Up media_convert.go

**Files:**
- Modify:`/Users/fd/Documents/perf/helper/media_convert.go`

**Interfaces:**
- Consumes: Nothing
- Produces: `ConvertImageToWebp`, `ConvertVideoToWebp`

- [ ] **Step 1: Simplify Convert methods**
Remove `InjectExif` from `media_convert.go` since it will be rewritten from scratch in `exif.go`.

```go
package helper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

func ConvertImageToWebp(imgData []byte) ([]byte, error) {
	convertCmd := exec.Command("convert", "-", "-resize", "512x512", "-background", "none", "-compose", "Copy", "-gravity", "center", "-extent", "512x512", "-quality", "100", "png:-")
	convertCmd.Stdin = bytes.NewReader(imgData)
	var convertBuf bytes.Buffer
	convertCmd.Stdout = &convertBuf
	if err := convertCmd.Run(); err != nil {
		return nil, fmt.Errorf("convert failed: %w", err)
	}

	cwebpCmd := exec.Command("cwebp", "-quiet", "-mt", "-exact", "-q", "100", "-m", "6", "-alpha_q", "100", "-o", "-", "--", "-")
	cwebpCmd.Stdin = &convertBuf
	var cwebpBuf bytes.Buffer
	cwebpCmd.Stdout = &cwebpBuf
	if err := cwebpCmd.Run(); err != nil {
		return nil, fmt.Errorf("cwebp failed: %w", err)
	}

	return cwebpBuf.Bytes(), nil
}

func ConvertVideoToWebp(vidData []byte) ([]byte, error) {
	webpPath := filepath.Join("temp", fmt.Sprintf("%s.webp", uuid.New().String()))
	defer os.Remove(webpPath)

	ffmpegCmd := exec.Command("ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0", "-f", "mp4",
		"-ss", "00:00:00", "-t", "00:00:15",
		"-vf", "fps=10,scale=720:-1:flags=lanczos:force_original_aspect_ratio=increase,crop=512:512,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=#00000000,setsar=1",
		"-compression_level", "6",
		"-q:v", "60", "-loop", "0",
		"-preset", "picture", "-an", "-fps_mode", "auto",
		"-f", "webp", webpPath,
	)
	ffmpegCmd.Stdin = bytes.NewReader(vidData)
	if err := ffmpegCmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	return os.ReadFile(webpPath)
}
```

- [ ] **Step 2: Check standard test suite**
Run: `go build ./...`

- [ ] **Step 3: Commit**
```bash
git add helper/media_convert.go
git commit -m "refactor: remove InjectExif shell wrapper from media_convert"
```

### Task 2: Implement Pure-Go RIFF Injector

**Files:**
- Modify:`/Users/fd/Documents/perf/helper/exif.go`

**Interfaces:**
- Produces: `InjectExif(webpData []byte, packName, publisher string) ([]byte, error)`
- Consumes: Nothing

- [ ] **Step 1: Write `helper/exif.go`**
Overwrite `/Users/fd/Documents/perf/helper/exif.go` with the completely new implementation. The previous dynamic `temp/default.exif` file logic is gone.

```go
package helper

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
)

type StickerMetadata struct {
	PackId    string   `json:"sticker-pack-id"`
	Name      string   `json:"sticker-pack-name"`
	Publisher string   `json:"sticker-pack-publisher"`
	Emojis    []string `json:"emojis,omitempty"`
}

func writeUIntLE(buffer []byte, value, offset, byteLength int64) {
	slice := make([]byte, byteLength)
	val := new(big.Int)
	val.SetUint64(uint64(value))
	valBytes := val.Bytes()

	tmp := make([]byte, len(valBytes))
	for i := range valBytes {
		tmp[i] = valBytes[len(valBytes)-1-i]
	}
	copy(slice, tmp)
	copy(buffer[offset:], slice)
}

func createMetadataBytes(packName, publisher string) []byte {
	exifObj := StickerMetadata{
		PackId:    "com.wa.bot.sticker",
		Name:      packName,
		Publisher: publisher,
		Emojis:    []string{"😀"},
	}

	exifJson, err := json.Marshal(exifObj)
	if err != nil {
		return nil
	}

	bit := []byte{0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x01, 0x00, 0x41, 0x57, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00}
	bit = append(bit, exifJson...)
	writeUIntLE(bit, int64(len(exifJson)), 14, 4)

	return bit
}

// InjectExif takes raw WebP bytes and constructs a new WebP RIFF byte slice containing the EXIF metadata
func InjectExif(webpData []byte, packName, publisher string) ([]byte, error) {
	if len(webpData) < 12 || string(webpData[0:4]) != "RIFF" || string(webpData[8:12]) != "WEBP" {
		return nil, fmt.Errorf("not a valid WEBP file")
	}

	var width, height uint32
	hasVP8X := false
	var vp8xFlags byte
	var vp8xBytes []byte
	
	var chunks [][]byte
	
	// Default defaults for VP8X canvas sizes 
	width = 512
	height = 512

	offset := 12
	for offset < len(webpData) {
		if offset+8 > len(webpData) {
			break
		}
		chunkID := string(webpData[offset : offset+4])
		chunkSize := binary.LittleEndian.Uint32(webpData[offset+4 : offset+8])

		paddedSize := chunkSize
		if paddedSize%2 != 0 {
			paddedSize++
		}

		if offset+8+int(paddedSize) > len(webpData) {
			break
		}

		chunkData := webpData[offset+8 : offset+8+int(paddedSize)]

		switch chunkID {
		case "VP8X":
			hasVP8X = true
			vp8xFlags = chunkData[0]
			vp8xBytes = chunkData
			// extract dimensions
			wBytes := []byte{chunkData[4], chunkData[5], chunkData[6], 0}
			hBytes := []byte{chunkData[7], chunkData[8], chunkData[9], 0}
			width = binary.LittleEndian.Uint32(wBytes) + 1
			height = binary.LittleEndian.Uint32(hBytes) + 1
		case "VP8 ":
			if !hasVP8X {
				// extract 14 bit scale block
				w := uint32(chunkData[6]) | (uint32(chunkData[7]) << 8)
				width = w & 0x3FFF
				h := uint32(chunkData[8]) | (uint32(chunkData[9]) << 8)
				height = h & 0x3FFF
			}
			chunks = append(chunks, webpData[offset:offset+8+int(paddedSize)])
		case "VP8L":
			if !hasVP8X {
				// Parse bits 1-28 for w/h
				b0 := uint32(chunkData[1])
				b1 := uint32(chunkData[2])
				b2 := uint32(chunkData[3])
				b3 := uint32(chunkData[4])
				width = 1 + (((b1 & 0x3F) << 8) | b0)
				height = 1 + (((b3 & 0xF) << 10) | (b2 << 2) | ((b1 & 0xC0) >> 6))
			}
			chunks = append(chunks, webpData[offset:offset+8+int(paddedSize)])
		default:
			if chunkID != "EXIF" {
				chunks = append(chunks, webpData[offset:offset+8+int(paddedSize)])
			}
		}
		offset += 8 + int(paddedSize)
	}

	exifMeta := createMetadataBytes(packName, publisher)
	if exifMeta == nil {
		return nil, fmt.Errorf("failed to create exif byte payload")
	}

	// Turn on EXIF flag (bit 3) in VP8X
	if !hasVP8X {
		vp8xBytes = make([]byte, 10)
		vp8xFlags = 0x08 // EXIF flag is bit 3
		vp8xBytes[0] = vp8xFlags
		w := width - 1
		h := height - 1
		vp8xBytes[4] = byte(w)
		vp8xBytes[5] = byte(w >> 8)
		vp8xBytes[6] = byte(w >> 16)
		vp8xBytes[7] = byte(h)
		vp8xBytes[8] = byte(h >> 8)
		vp8xBytes[9] = byte(h >> 16)
	} else {
		vp8xBytes[0] = vp8xFlags | 0x08 
	}

	var buf bytes.Buffer
	buf.WriteString("RIFF")
	buf.Write([]byte{0, 0, 0, 0}) // Total size placeholder
	buf.WriteString("WEBP")

	// Write VP8X
	buf.WriteString("VP8X")
	vp8xSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(vp8xSize, 10)
	buf.Write(vp8xSize)
	buf.Write(vp8xBytes)

	// Write existing chunks
	for _, chunk := range chunks {
		buf.Write(chunk)
	}

	// Write EXIF
	buf.WriteString("EXIF")
	exifSize := uint32(len(exifMeta))
	exifSizeSlice := make([]byte, 4)
	binary.LittleEndian.PutUint32(exifSizeSlice, exifSize)
	buf.Write(exifSizeSlice)
	buf.Write(exifMeta)
	if exifSize%2 != 0 {
		buf.WriteByte(0)
	}

	result := buf.Bytes()
	totalSize := uint32(len(result) - 8)
	binary.LittleEndian.PutUint32(result[4:8], totalSize)

	return result, nil
}
```

- [ ] **Step 2: Check test suite**
Run: `go build ./...`

- [ ] **Step 3: Commit**
```bash
git add helper/exif.go
git commit -m "feat: implement pure-go RIFF EXIF injector"
```

### Task 3: Update cmds/media/sticker.go Signature

**Files:**
- Modify:`/Users/fd/Documents/perf/cmds/media/sticker.go`

**Interfaces:**
- Consumes: `helper.InjectExif(rawWebp, packName, publisher)`

- [ ] **Step 1: Replace ResolveExif usages in `cmds/media/sticker.go`**
Read `ctx.Arguments()` directly to compute `packName` and `publisher` locally instead of calling `helper.ResolveExif`. Use the same ENV logic as fallback defaults if args are empty. Remove `helper.ResolveExif` code context cleanup.

```go
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
		ctx.React("👌")

		args := ctx.Arguments()
		packName := os.Getenv("STICKER_NAME")
		if packName == "" {
			packName = "Bot"
		}
		publisher := os.Getenv("STICKER_PUBLISHER")
		if publisher == "" {
			publisher = "Bot"
		}

		if len(args) > 0 {
			argStr := strings.Join(args, " ")
			parts := strings.Split(argStr, "|")
			packName = strings.TrimSpace(parts[0])
			publisher = "Bot"
			if len(parts) > 1 {
				publisher = strings.TrimSpace(parts[1])
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
```

- [ ] **Step 2: Verify Compilation**
Run: `go build ./...`

- [ ] **Step 3: Commit**
```bash
git add cmds/media/sticker.go
git commit -m "feat: use pure-go exif injector in sticker command"
```
