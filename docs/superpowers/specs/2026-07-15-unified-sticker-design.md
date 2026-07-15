# Unified Sticker Maker Design

## 1. Boot Exif
Generates temp/default.exif on boot using STICKER_NAME and STICKER_PUBLISHER environment variables if missing.

## 2. Parameter Parsing
- Empty: uses default.exif
- 'name': Custom exif with given name
- 'name|publisher': Custom exif with parsed name and publisher.

## 3. Media Capabilities
Detects Images, Video, and existing Stickers via Message or QuotedMessage. Converts with webpmux and uploads.

## 4. Cleanup
Custom exifs are deleted on action deferred. Default exif persists.

