package download

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"neo/api"
	hc "neo/context"
	"neo/core"
	"neo/helper"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
)

func init() {
	core.Default.Register(YouTube)
	core.Default.Use(handleYTReply)
}

func handleYTReply(ctx *hc.Ctx) bool {
	if ctx.Cmd() != "" {
		return false
	}

	quotedID := hc.GetQuotedStanzaID(ctx.Message())
	if quotedID == "" {
		return false
	}

	result, ok := api.GetYtCache(quotedID)
	if !ok {
		return false
	}

	formatID := strings.TrimSpace(hc.ParseMessageText(ctx.Event()))
	if formatID == "" || !api.HasFormatID(result, formatID) {
		return false
	}

	ctx.Send("Downloading...")
	downloadByFormatID(ctx, result, formatID)
	return true
}

var YouTube = &core.Command{
	Name:        "youtube",
	Aliases:     []string{"yt", "ytdl"},
	Category:    "download",
	Description: "Download YouTube video",
	Run: func(ctx *hc.Ctx) {
		url := ctx.Arg(0)
		if url == "" {
			ctx.Reply("Usage: !yt <url>")
			return
		}

		ytRegex := regexp.MustCompile(`(?i)(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/|youtube\.com\/shorts\/)([^"&?\/\s]{11})`)
		if !ytRegex.MatchString(url) {
			ctx.Reply("Invalid YouTube URL")
			return
		}

		ctx.Send("Processing...")
		var res api.YtDownloadResult
		if err := api.Default().Get("/yt/query", map[string]string{"url": url}, &res); err != nil {
			ctx.Reply(fmt.Sprintf("Error: %v", err))
			return
		}

		showFormatPicker(ctx, &res)
	},
}

func showFormatPicker(ctx *hc.Ctx, res *api.YtDownloadResult) {
	thumb, _, err := api.Download(res.Thumbnail)
	if err != nil {
		ctx.Send(formatCaption(res))
		return
	}

	img, err := ctx.UploadImage(thumb, formatCaption(res))
	if err != nil {
		ctx.Reply("Failed to upload thumbnail: " + err.Error())
		return
	}

	resp, err := ctx.Client().SendMessage(context.Background(), ctx.ChatJID(), &waE2E.Message{ImageMessage: img})
	if err != nil {
		ctx.Reply("Failed to send: " + err.Error())
		return
	}

	api.SetYtCache(resp.ID, res)
}

func formatCaption(res *api.YtDownloadResult) string {
	var b strings.Builder
	b.WriteString("Title: ")
	b.WriteString(res.Title)
	b.WriteString("\n\n── Video ──\n")
	for _, f := range res.VideoFormats {
		if f.URL == "" {
			continue
		}
		fmt.Fprintf(&b, "%s · %s · %s · %s\n", f.FormatID, f.Resolution, f.Ext, formatSize(f.Filesize))
	}
	b.WriteString("\n── Audio ──\n")
	for _, f := range res.AudioFormats {
		if f.URL == "" {
			continue
		}
		fmt.Fprintf(&b, "%s · %s · %s\n", f.FormatID, f.Ext, formatSize(f.Filesize))
	}
	b.WriteString("\nReply with formatId")
	return b.String()
}

func formatSize(bytes int) string {
	if bytes <= 0 {
		return "?"
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%dKB", bytes/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}

func downloadByFormatID(ctx *hc.Ctx, res *api.YtDownloadResult, fid string) {
	var targetURL string
	var isAudioOnly bool

	// Look for video
	for _, f := range res.VideoFormats {
		if f.FormatID == fid && f.URL != "" {
			targetURL = f.URL
			break
		}
	}

	// Look for audio if not found in video
	if targetURL == "" {
		for _, f := range res.AudioFormats {
			if f.FormatID == fid && f.URL != "" {
				targetURL = f.URL
				isAudioOnly = true
				break
			}
		}
	}

	if targetURL == "" {
		ctx.Reply("Format " + fid + " not found")
		return
	}

	data, mime, err := api.Download(targetURL)
	if err != nil {
		ctx.Reply(fmt.Sprintf("Download failed: %v", err))
		return
	}

	// Detect if it is audio from googlevideo (which is often fragmented MP4/DASH),
	// pipe it to ffmpeg to get standard MP3 so WhatsApp can play it
	if isAudioOnly && strings.Contains(targetURL, "googlevideo.com") {
		convertedData, err := helper.ConvertToMP3(data)
		if err == nil {
			data = convertedData
			mime = "audio/mpeg"
		} else {
			fmt.Printf("Failed to convert audio using ffmpeg: %v\n", err)
		}
	}

	helper.SendMedia(ctx, data, mime, res.Title)
}
