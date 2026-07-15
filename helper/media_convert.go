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

func InjectExif(webpData []byte, exifPath string) ([]byte, error) {
	webpPath := filepath.Join("temp", fmt.Sprintf("%s.webp", uuid.New().String()))
	defer os.Remove(webpPath)

	tmpWebp := webpPath + ".tmp.webp"
	if err := os.WriteFile(tmpWebp, webpData, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(tmpWebp)

	webpmuxCmd := exec.Command("webpmux", "-set", "exif", exifPath, tmpWebp, "-o", webpPath)
	if err := webpmuxCmd.Run(); err != nil {
		return nil, fmt.Errorf("webpmux failed: %w", err)
	}

	return os.ReadFile(webpPath)
}
