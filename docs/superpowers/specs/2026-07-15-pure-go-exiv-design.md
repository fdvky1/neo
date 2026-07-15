# Pure-Go WebP EXIF RIFF Injector

## 1. Goal
Completely remove file-based IO and the `webpmux` dependency by weaving the EXIF chunk into the WebP RIFF byte struct physically in memory.

## 2. Architecture
Instead of using `webpmux` passing paths around, `helper/exif.go` will implement `InjectExif(webpBytes []byte, packName, publisher string) ([]byte, error)`.

## 3. RIFF Parsing Logic
1. Validate `RIFF` and `WEBP` signatures.
2. Read all chunks to extract Canvas Width/Height (from `VP8X`, `VP8L`, or `VP8 `).
3. Identify if `VP8X` is missing, and if so, construct it. We must ensure the `EXIF` bit flag is set.
4. Construct the custom `EXIF` chunk (which holds the exact binary array and JSON metadata we validated via F-BOT_GO earlier).
5. Append `EXIF` to the end of the WebP chunks, update the main `RIFF` file size pointer, and return the aggregated `[]byte` slice.

## 4. Main Command Refactor
`cmds/media/sticker.go` will stop requesting files via `ResolveExif()`. It will simply pass the raw Webp bytes received from `ConvertImageToWebp` or `ConvertVideoToWebp` into `InjectExif` and instantly upload the resulting byte array.
