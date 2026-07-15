# Unified Sticker Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a single `.sticker` command that natively handles images, videos, existing stickers, boot-managed Exif generation, and inline watermark arguments.

**Architecture:** A single command definition in `cmds/media/sticker.go` that extracts Exif checking/generation to the init loop (or a lazy helper), processes args to create ephemeral Exif files, performs the webpmux conversion, handles upload and cleanup.

**Tech Stack:** Go (whatsmeow, local `ffmpeg`, `imagemagick`/convert, `webpmux`), internal `Roxy/command` framework.

## Global Constraints
- Target file: `/Users/fd/Documents/perf/cmds/media/sticker.go`
- Clean up other files: `/Users/fd/Documents/perf/Lara/src/cmd/media/stickerwm.go` and `/Users/fd/Documents/perf/Lara/src/cmd/media/wm.go` must be deleted (or un-registered). Wait, we are editing in `cmds/media/` not `Lara/src/cmd/media/`. We will overwrite `/Users/fd/Documents/perf/cmds/media/sticker.go` and ensure we aren't duplicating Lara source. Let's adapt the plan to just rewrite `/Users/fd/Documents/perf/cmds/media/sticker.go`.
- DB (`repo.WMRepository`) usage for sticker command is removed.
- `temp/default.exif` behavior on boot or on-demand when empty.

---

### Task 1: Rewrite cmds/media/sticker.go Support Functions

**Files:**
- Modify:`/Users/fd/Documents/perf/cmds/media/sticker.go`

**Interfaces:**
- Produces: `getExif(args string) (string, bool)` helper, `processMedia` generic handler.

- [ ] **Step 1: Write the Exif resolution helper**
```go
package media

import (
	"bytes"
	"os"
	"strings"

	"github.com/itzngga/Lara/src/cmd/constant"
	util2 "github.com/itzngga/Lara/util"
	"github.com/itzngga/Lara/util/metadata"
	"github.com/itzngga/Lara/util/scrapper"
	"github.com/itzngga/Roxy/command"
	"github.com/itzngga/Roxy/embed"
	"github.com/itzngga/Roxy/util"
	"github.com/itzngga/Roxy/util/cli"
	cmdchain "github.com/rainu/go-command-chain"
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// getOrCreateDefaultExif ensures temp/default.exif exists using ENV variables
func getOrCreateDefaultExif() string {
	path := "temp/default.exif"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		name := os.Getenv("STICKER_NAME")
		if name == "" {
			name = "Sticker"
		}
		pub := os.Getenv("STICKER_PUBLISHER")
		if pub == "" {
			pub = "Roxy"
		}
		b := metadata.CreateMetadata(metadata.StickerMetadata{
			Name:      name,
			Publisher: pub,
		})
		_ = os.WriteFile(path, b, os.ModePerm)
	}
	return path
}

// resolveExif returns the filepath to the exif to use, and a boolean true if it's a temp file that should be deleted
func resolveExif(args []string) (string, bool) {
	if len(args) == 0 {
		return getOrCreateDefaultExif(), false
	}
	
	joined := strings.Join(args, " ")
	parts := strings.Split(joined, "|")
	name := strings.TrimSpace(parts[0])
	pub := ""
	if len(parts) > 1 {
		pub = strings.TrimSpace(parts[1])
	}

	b := metadata.CreateMetadata(metadata.StickerMetadata{
		Name:      name,
		Publisher: pub,
	})
	
	tmpPath := "temp/" + util2.MakeMD5UUID() + ".exif"
	_ = os.WriteFile(tmpPath, b, os.ModePerm)
	return tmpPath, true
}
```

- [ ] **Step 2: Commit helper functions**
```bash
git add /Users/fd/Documents/perf/cmds/media/sticker.go
git commit -m "feat: add exif resolution helpers to sticker command"
```

### Task 2: Implement Media Conversions in sticker.go

**Files:**
- Modify:`/Users/fd/Documents/perf/cmds/media/sticker.go`

**Interfaces:**
- Produces: `cmdStickerImage`, `cmdStickerVideo`, `cmdStickerRewrite` 

- [ ] **Step 1: Write specific converters**
```go
func cmdStickerImage(ctx *command.RunFuncContext, img *waProto.ImageMessage, exifPath string, webpPath string) error {
	data, err := ctx.Client.Download(img)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(data)
	return cmdchain.Builder().
		Join("convert", "-", "-resize", "512x512", "-background", "none", "-compose", "Copy", "-gravity", "center", "-extent", "512x512", "-quality", "100", "png:-").
		WithInjections(reader).ForwardError().
		Join("cwebp", "-quiet", "-mt", "-exact", "-q", "100", "-m", "6", "-alpha_q", "100", "-o", "-", "--", "-").
		Join("webpmux", "-set", "exif", exifPath, "-", "-o", webpPath).
		Finalize().WithError(os.Stderr).Run()
}

func cmdStickerVideo(ctx *command.RunFuncContext, video *waProto.VideoMessage, exifPath string, webpPath string) error {
	data, err := ctx.Client.Download(video)
	if err != nil {
		return err
	}
	
	// Convert straight to webp then mux
	tmpWebp := webpPath + ".tmp.webp"
	defer os.Remove(tmpWebp)
	
	_, err = cli.ExecPipeline("ffmpeg", data,
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0", "-f", "mp4",
		"-ss", "00:00:00", "-t", "00:00:15",
		"-vf", "fps=10,scale=720:-1:flags=lanczos:force_original_aspect_ratio=increase,crop=512:512,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=#00000000,setsar=1",
		"-compression_level", "6",
		"-q:v", "60", "-loop", "0",
		"-preset", "picture", "-an", "-fps_mode", "auto",
		"-f", "webp", tmpWebp,
	)
	if err != nil {
		return err
	}
	
	return cmdchain.Builder().
		Join("webpmux", "-set", "exif", exifPath, tmpWebp, "-o", webpPath).
		Finalize().Run()
}

func cmdStickerRewrite(ctx *command.RunFuncContext, exifPath string, webpPath string) error {
	_, err := ctx.DownloadToFile(true, webpPath)
	if err != nil {
		return err
	}
	return cmdchain.Builder().
		Join("webpmux", "-set", "exif", exifPath, webpPath, "-o", webpPath).
		Finalize().Run()
}
```

- [ ] **Step 2: Commit media converters**
```bash
git add /Users/fd/Documents/perf/cmds/media/sticker.go
git commit -m "feat: add unified media conversion functions for sticker"
```

### Task 3: Assemble the Main Command Registration

**Files:**
- Modify:`/Users/fd/Documents/perf/cmds/media/sticker.go`

**Interfaces:**
- Produces: `var sticker = &command.Command{...}` and `init()`

- [ ] **Step 1: Write main command mapping**
```go
func init() {
	embed.Commands.Add(sticker)
}

var sticker = &command.Command{
	Name:        "sticker",
	Aliases:     []string{"s", "stiker"},
	Category:    constant.MEDIA_CATEGORY,
	Description: "Create sticker from image/video or rewrite existing sticker EXIF",
	RunFunc: func(ctx *command.RunFuncContext) *waProto.Message {
		defer scrapper.TimeElapsed("Sticker Maker")()
		ctx.SendEmoji("👌")
		
		exifPath, cleanupExif := resolveExif(ctx.Arguments)
		defer func() {
			if cleanupExif {
				os.Remove(exifPath)
			}
		}()
		
		webpPath := "temp/" + util2.MakeMD5UUID() + ".webp"
		defer os.Remove(webpPath)
		
		msg := ctx.Message
		quoted := util.ParseQuotedMessage(msg)
		
		var err error
		if msg.GetImageMessage() != nil {
			err = cmdStickerImage(ctx, msg.GetImageMessage(), exifPath, webpPath)
		} else if quoted.GetImageMessage() != nil {
			err = cmdStickerImage(ctx, quoted.GetImageMessage(), exifPath, webpPath)
		} else if msg.GetVideoMessage() != nil {
			err = cmdStickerVideo(ctx, msg.GetVideoMessage(), exifPath, webpPath)
		} else if quoted.GetVideoMessage() != nil {
			err = cmdStickerVideo(ctx, quoted.GetVideoMessage(), exifPath, webpPath)
		} else if quoted.GetStickerMessage() != nil {
			err = cmdStickerRewrite(ctx, exifPath, webpPath)
		} else {
			return ctx.GenerateReplyMessage("error: reply to image/video/sticker or send with caption")
		}
		
		if err != nil {
			return ctx.GenerateReplyMessage("error: " + err.Error())
		}
		
		webpMsg, err := ctx.UploadStickerMessageFromPath(webpPath)
		if err != nil {
			return ctx.GenerateReplyMessage("error: " + err.Error())
		}
		
		return ctx.GenerateReplyMessage(webpMsg)
	},
}
```

- [ ] **Step 2: Remove redundant legacy files if they exist locally on perf**
Run `rm -f /Users/fd/Documents/perf/cmds/media/stickerwm.go /Users/fd/Documents/perf/cmds/media/wm.go` just in case they were copied over here (though wait, we only rewrite cmds/media/sticker.go per prompt).

- [ ] **Step 3: Build/Test via standard go build to ensure compilation.**
Run: `go build ./...`

- [ ] **Step 4: Commit main command registration**
```bash
git add /Users/fd/Documents/perf/cmds/media/sticker.go
git commit -m "feat: assemble and register unified sticker command"
```

