package api

import (
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
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("download request: %w", err)
	}

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

	return data, mime, nil
}
