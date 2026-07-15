package api

import (
	"sync"
	"time"
)

type YtFormat struct {
	Ext        string `json:"ext"`
	Filesize   int    `json:"filesize"`
	FormatID   string `json:"formatId"`
	HasAudio   bool   `json:"hasAudio"`
	Resolution string `json:"resolution"`
	URL        string `json:"url"`
}

type YtDownloadResult struct {
	AudioFormats []YtFormat `json:"audioFormats"`
	Thumbnail    string     `json:"thumbnail"`
	Title        string     `json:"title"`
	VideoFormats []YtFormat `json:"videoFormats"`
	VideoID      string     `json:"videoId"`
}

const ytCacheTTL = 10 * time.Minute

type ytCacheEntry struct {
	result    *YtDownloadResult
	expiresAt time.Time
}

var (
	ytMu    sync.Mutex
	ytCache = map[string]ytCacheEntry{}
)

func GetYtCache(messageID string) (*YtDownloadResult, bool) {
	ytMu.Lock()
	defer ytMu.Unlock()

	entry, ok := ytCache[messageID]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(ytCache, messageID)
		return nil, false
	}
	return entry.result, true
}

func SetYtCache(messageID string, result *YtDownloadResult) {
	ytMu.Lock()
	defer ytMu.Unlock()

	now := time.Now()
	for id, entry := range ytCache {
		if now.After(entry.expiresAt) {
			delete(ytCache, id)
		}
	}
	ytCache[messageID] = ytCacheEntry{
		result:    result,
		expiresAt: now.Add(ytCacheTTL),
	}
}

func HasFormatID(res *YtDownloadResult, formatID string) bool {
	for _, f := range res.VideoFormats {
		if f.FormatID == formatID && f.URL != "" {
			return true
		}
	}
	for _, f := range res.AudioFormats {
		if f.FormatID == formatID && f.URL != "" {
			return true
		}
	}
	return false
}

// func DownloadRemuxedMP4(rawURL string) ([]byte, error) {
// 	if _, err := exec.LookPath("ffmpeg"); err != nil {
// 		return nil, fmt.Errorf("ffmpeg not found: %w", err)
// 	}

// 	data, mime, err := Download(rawURL)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if !strings.HasPrefix(mime, "video/mp4") {
// 		return data, nil
// 	}

// 	// TEST: set YT_NO_REMUX=1 to skip ffmpeg and use raw API output
// 	if os.Getenv("YT_NO_REMUX") != "" {
// 		return data, nil
// 	}

// 	out, err := os.CreateTemp("", "yt-out-*.mp4")
// 	if err != nil {
// 		return nil, fmt.Errorf("temp file: %w", err)
// 	}
// 	outPath := out.Name()
// 	out.Close()
// 	defer os.Remove(outPath)

// 	cmdCtx, cancelCmd := context.WithTimeout(context.Background(), 5*time.Minute)
// 	defer cancelCmd()

// 	cmd := exec.CommandContext(cmdCtx, "ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
// 		"-i", "pipe:0", "-c", "copy", "-movflags", "+faststart", "-f", "mp4", outPath)
// 	cmd.Stdin = bytes.NewReader(data)
// 	if output, err := cmd.CombinedOutput(); err != nil {
// 		return nil, fmt.Errorf("ffmpeg remux: %w: %s", err, strings.TrimSpace(string(output)))
// 	}

// 	info, err := os.Stat(outPath)
// 	if err != nil {
// 		return nil, fmt.Errorf("stat remuxed file: %w", err)
// 	}
// 	if info.Size() > MaxDownloadBytes {
// 		return nil, fmt.Errorf("remuxed file too large: %d bytes", info.Size())
// 	}

// 	return os.ReadFile(outPath)
// }
