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
	var stderr bytes.Buffer
	convertCmd.Stdout = &convertBuf
	convertCmd.Stderr = &stderr
	if err := convertCmd.Run(); err != nil {
		return nil, fmt.Errorf("convert failed: %w, stderr: %s", err, stderr.String())
	}

	cwebpCmd := exec.Command("cwebp", "-quiet", "-mt", "-exact", "-q", "100", "-m", "6", "-alpha_q", "100", "-o", "-", "--", "-")
	cwebpCmd.Stdin = &convertBuf
	var cwebpBuf bytes.Buffer
	var cwebStderr bytes.Buffer
	cwebpCmd.Stdout = &cwebpBuf
	cwebpCmd.Stderr = &cwebStderr
	if err := cwebpCmd.Run(); err != nil {
		return nil, fmt.Errorf("cwebp failed: %w, stderr: %s", err, cwebStderr.String())
	}

	return cwebpBuf.Bytes(), nil
}

func ConvertVideoToWebp(vidData []byte) ([]byte, error) {
	id := uuid.New().String()
	gifPath := filepath.Join("temp", fmt.Sprintf("%s.gif", id))
	webpPath := filepath.Join("temp", fmt.Sprintf("%s.webp", id))

	defer os.Remove(gifPath)
	defer os.Remove(webpPath)

	ffmpegCmd := exec.Command("ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0",
		"-ss", "00:00:00", "-t", "00:00:15",
		"-vf", "fps=10,scale=720:-1:flags=lanczos:force_original_aspect_ratio=increase,crop=512:512,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=#00000000,setsar=1",
		"-loop", "0",
		"-an",
		"-f", "gif", gifPath,
	)
	var stderr bytes.Buffer
	ffmpegCmd.Stderr = &stderr
	ffmpegCmd.Stdin = bytes.NewReader(vidData)
	if err := ffmpegCmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w, stderr: %s", err, stderr.String())
	}

	gif2webpCmd := exec.Command("gif2webp", "-quiet", "-q", "60", "-m", "6", gifPath, "-o", webpPath)
	var g2wStderr bytes.Buffer
	gif2webpCmd.Stderr = &g2wStderr
	if err := gif2webpCmd.Run(); err != nil {
		return nil, fmt.Errorf("gif2webp failed: %w, stderr: %s", err, g2wStderr.String())
	}

	return os.ReadFile(webpPath)
}
