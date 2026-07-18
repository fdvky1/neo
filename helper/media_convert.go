package helper

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/davidbyttow/govips/v2/vips"
)

func ConvertImageToWebp(imgData []byte) ([]byte, error) {
	img, err := vips.NewImageFromBuffer(imgData)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %w", err)
	}
	defer img.Close()

	if !img.HasAlpha() {
		if err := img.AddAlpha(); err != nil {
			return nil, fmt.Errorf("failed to add alpha: %w", err)
		}
	}

	if err := img.ThumbnailWithSize(512, 512, vips.InterestingNone, vips.SizeBoth); err != nil {
		return nil, fmt.Errorf("failed to resize: %w", err)
	}

	// Center in 512x512 canvas with transparent background (fit: contain)
	w, h := img.Width(), img.Height()
	if w != 512 || h != 512 {
		left := (512 - w) / 2
		top := (512 - h) / 2
		if err := img.EmbedBackgroundRGBA(left, top, 512, 512, &vips.ColorRGBA{R: 0, G: 0, B: 0, A: 0}); err != nil {
			return nil, fmt.Errorf("failed to embed: %w", err)
		}
	}

	webpBuf, _, err := img.ExportWebp(&vips.WebpExportParams{
		Quality:         100,
		Lossless:        false,
		ReductionEffort: 6,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to export webp: %w", err)
	}

	return webpBuf, nil
}

func ConvertVideoToWebp(vidData []byte) ([]byte, error) {
	ffmpegCmd := exec.Command("ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-ss", "00:00:00", "-t", "00:00:15",
		"-vcodec", "libwebp",
		"-vf", "fps=10,scale=720:-1:flags=lanczos:force_original_aspect_ratio=increase,crop=512:512,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=#00000000,setsar=1",
		"-loop", "0",
		"-preset", "default",
		"-an",
		"-f", "webp", "pipe:1",
	)

	ffmpegCmd.Stdin = bytes.NewReader(vidData)
	var outBuf bytes.Buffer
	var stderr bytes.Buffer
	ffmpegCmd.Stdout = &outBuf
	ffmpegCmd.Stderr = &stderr

	if err := ffmpegCmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg direct convert failed: %w, stderr: %s", err, stderr.String())
	}

	return outBuf.Bytes(), nil
}
