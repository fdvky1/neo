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
	webpPath := filepath.Join("temp", fmt.Sprintf("%s.webp", uuid.New().String()))
	defer os.Remove(webpPath)

	ffmpegCmd := exec.Command("ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", "pipe:0", "-f", "mp4",
		"-ss", "00:00:00", "-t", "00:00:15",
		"-vf", "fps=10,scale=720:-1:flags=lanczos:force_original_aspect_ratio=increase,crop=512:512,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=#00000000,setsar=1",
		"-compression_level", "6",
		"-q:v", "60", "-loop", "0",
		"-preset", "picture", "-an",
		"-f", "webp", webpPath,
	)
	var stderr bytes.Buffer
	ffmpegCmd.Stderr = &stderr
	ffmpegCmd.Stdin = bytes.NewReader(vidData)
	if err := ffmpegCmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w, stderr: %s", err, stderr.String())
	}

	return os.ReadFile(webpPath)
}
