package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

// Client is the shared HTTP client for the Mono REST API.
type Client struct {
	BaseURL string
	APIKey  string
}

const MaxDownloadBytes int64 = 256 * 1024 * 1024

var (
	defaultClient *Client
	defaultOnce   sync.Once
	httpClient    = &http.Client{Timeout: 2 * time.Minute}
)

func initDefault() {
	baseURL := os.Getenv("MONO_BASEURL")
	if baseURL == "" {
		baseURL = "https://mono.fdvky.me/api/v1"
	}
	defaultClient = &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  os.Getenv("MONO_API_KEY"),
	}
}

// Default returns the shared API client, initialized lazily so env vars
// loaded via godotenv in main.init() are already available.
func Default() *Client {
	defaultOnce.Do(initDefault)
	return defaultClient
}

// Get calls the API and decodes the JSON response into target.
func (c *Client) Get(path string, params map[string]string, target any) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("api url: %w", err)
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return fmt.Errorf("api request: %w", err)
	}
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// Download fetches a URL and returns the raw bytes and MIME type.
func Download(rawURL string) ([]byte, string, error) {
	// fast-path for googlevideo.com to bypass throttling via chunked requests
	if strings.Contains(rawURL, "googlevideo.com") {
		return downloadChunked(rawURL)
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("download request: %w", err)
	}

	// Add standard browser headers to avoid getting throttled/blocked
	// by CDNs like googlevideo.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("DNT", "1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, "", fmt.Errorf("download %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.ContentLength > MaxDownloadBytes {
		return nil, "", fmt.Errorf("download too large: %d bytes", resp.ContentLength)
	}

	mime := resp.Header.Get("Content-Type")

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxDownloadBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read: %w", err)
	}
	if int64(len(data)) > MaxDownloadBytes {
		return nil, "", fmt.Errorf("download too large: over %d bytes", MaxDownloadBytes)
	}

	if mime == "" || mime == "application/octet-stream" {
		mime = mimetype.Detect(data).String()
	}

	// fmt.Printf("Downloaded %d bytes from %s, MIME: %s\n", len(data), rawURL, mime)
	return data, mime, nil
}

func downloadChunked(rawURL string) ([]byte, string, error) {
	chunkSize := 1024 * 1024 // 1MB
	offset := 0

	var buf bytes.Buffer
	var mime string

	for {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return nil, "", err
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("DNT", "1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+chunkSize-1))

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, "", err
		}

		if resp.StatusCode != 200 && resp.StatusCode != 206 {
			if resp.StatusCode == 416 && offset > 0 {
				resp.Body.Close()
				break
			}
			resp.Body.Close()
			return nil, "", fmt.Errorf("bad status: %d", resp.StatusCode)
		}

		if offset == 0 {
			mime = resp.Header.Get("Content-Type")
		}

		data, err := io.ReadAll(io.LimitReader(resp.Body, int64(chunkSize+1)))
		resp.Body.Close()
		if err != nil {
			return nil, "", err
		}

		if int64(buf.Len()+len(data)) > MaxDownloadBytes {
			return nil, "", fmt.Errorf("download too large: over %d bytes", MaxDownloadBytes)
		}

		buf.Write(data)

		if len(data) < chunkSize || resp.StatusCode == 200 {
			break
		}
		offset += len(data)
	}

	detectedMime := mime
	if detectedMime == "" || detectedMime == "application/octet-stream" {
		detectedMime = mimetype.Detect(buf.Bytes()).String()
	}

	// fmt.Printf("Downloaded %d chunks (%d bytes) from %s, MIME: %s\n", (offset/chunkSize)+1, buf.Len(), rawURL, detectedMime)
	return buf.Bytes(), detectedMime, nil
}
