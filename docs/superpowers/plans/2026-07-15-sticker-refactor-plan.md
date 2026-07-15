# Sticker Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move EXIF generation and WebP conversion helpers out of sticker.go into helper packages.

**Architecture:** 
1. Create `helper/exif.go` handling exif metadata extraction.
2. Create `helper/media_convert.go` abstracting commands like `cwebp` and `ffmpeg` wrappers to convert byte slices to webp.
3. Simplify `cmds/media/sticker.go` to depend on these new exports.

**Tech Stack:** Go, exec, Neo Context, `waProto`

## Global Constraints
- Only use standard library inside `helper/` where possible (no whatsapp dependencies except general bytes handling).
- The behavior of the `/Users/fd/Documents/perf/cmds/media/sticker.go` command should remain functionally identical.

---

### Task 1: Migrate Exif Support to Helper

**Files:**
- Create: `/Users/fd/Documents/perf/helper/exif.go`
- Modify: `/Users/fd/Documents/perf/cmds/media/sticker.go`

**Interfaces:**
- Produces: `helper.CreateDynamicExif(args []string) (exifPath string, cleanup bool)`

- [ ] **Step 1: Write `exif.go`**
Extract `exifData`, `webpExif`, `getOrCreateDefaultExif`, and `resolveExif` into `helper/exif.go`. Make `resolveExif` exported as `ResolveExif`. Ensure it uses the existing env logic. Keep `uuid` dependency.

- [ ] **Step 2: Update `sticker.go` EXIF usage**
Remove the exif functions/structs from `sticker.go` and update `Sticker.Run` to call `helper.ResolveExif(ctx.Arguments())`.

- [ ] **Step 3: Commit**

### Task 2: Abstract Conversion Functions

**Files:**
- Create: `/Users/fd/Documents/perf/helper/media_convert.go`
- Modify: `/Users/fd/Documents/perf/cmds/media/sticker.go`

**Interfaces:**
- Produces: 
  `ConvertImageToSticker(imgData []byte, exifPath string) ([]byte, error)`
  `ConvertVideoToSticker(vidData []byte, exifPath string) ([]byte, error)`
  `RewriteStickerExif(webpData []byte, exifPath string) ([]byte, error)`

- [ ] **Step 1: Create `media_convert.go`**
Implement the three conversion functions. Each should take bytes (downloaded media) and an `exifPath`, do the conversion via `exec.Command`, inject exif via `webpmux`, clean up its own internal temp files, and return the final `[]byte` of the webp ready to send. (They don't need Whatsapp Context).

- [ ] **Step 2: Update `sticker.go` media usage**
Remove `cmdStickerImage`, `cmdStickerVideo`, and `cmdStickerRewrite` from `sticker.go`. Instead, in `Sticker.Run`, download the bytes directly using `ctx.Client().Download(...)`, then pass those bytes to the `helper.ConvertX` functions, and `ctx.SendSticker()` the returned bytes. 

- [ ] **Step 3: Test build**
Run `go build ./...`

- [ ] **Step 4: Commit**
`git commit -m "refactor: extract media conversion logic to helper"`
