package media

import (
	"bytes"
	stdctx "context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	hc "neo/context"
	"neo/core"
	"neo/helper"

	"go.mau.fi/whatsmeow"
	waTypes "go.mau.fi/whatsmeow/types"
)

func init() { core.Default.Register(Quote) }

// quoteAPIRequest represents the POST body for the quote-api /generate endpoint.
type quoteAPIRequest struct {
	Type            string         `json:"type"`
	Format          string         `json:"format"`
	BackgroundColor string         `json:"backgroundColor"`
	Width           int            `json:"width"`
	Height          int            `json:"height"`
	Scale           int            `json:"scale"`
	Messages        []quoteMessage `json:"messages"`
}

type quoteMessage struct {
	Entities     []any              `json:"entities"`
	Avatar       bool               `json:"avatar"`
	From         quoteFrom          `json:"from"`
	Text         string             `json:"text,omitempty"`
	Media        *quoteMedia        `json:"media,omitempty"`
	ReplyMessage *quoteReplyMessage `json:"replyMessage,omitempty"`
}

type quoteFrom struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Photo quotePhoto `json:"photo"`
}

type quotePhoto struct {
	URL                string `json:"url,omitempty"`
	CreateImageLatters bool   `json:"createImageLatters,omitempty"`
}

type quoteMedia struct {
	URL string `json:"url,omitempty"`
}

type quoteReplyMessage struct {
	ChatID string `json:"chatId,omitempty"`
	Name   string `json:"name,omitempty"`
	Text   string `json:"text,omitempty"`
}

type quoteAPIResponse struct {
	Result struct {
		Image string `json:"image"`
	} `json:"result"`
	Image string `json:"image"`
}

var Quote = &core.Command{
	Name:        "quote",
	Aliases:     []string{"q", "qc"},
	Category:    "media",
	Description: "Generate a quote sticker from text or replied message",
	Run: func(ctx *hc.Ctx) {
		quoteURL := os.Getenv("QUOTE_URL")
		if quoteURL == "" {
			ctx.Reply("QUOTE_URL is not configured")
			return
		}

		args := ctx.Arguments()
		msg := ctx.Message()
		quoted := hc.GetQuotedMessage(msg)
		quotedCtxInfo := hc.GetContextInfo(msg)

		// Parse --img flag from args (accounting for smart dashes from iOS/macOS)
		sendAsImage := false
		filteredArgs := make([]string, 0, len(args))
		for _, a := range args {
			// Check for normal dashes, en dash, or em dash
			if a == "--img" || a == "-img" || a == "—img" || a == "–img" {
				sendAsImage = true
			} else {
				filteredArgs = append(filteredArgs, a)
			}
		}

		// Text args = everything except --img
		textArgs := strings.Join(filteredArgs, " ")

		// Determine text: from args or from quoted message
		text := textArgs
		if text == "" && quoted != nil {
			text = hc.GetQuotedText(quoted)
		}

		if text == "" {
			ctx.Reply(fmt.Sprintf("Send command %s%s <text> or reply to a message", ctx.Prefix(), ctx.Cmd()))
			return
		}

		// hasTextArgs = user actually typed content text (not just --img)
		hasTextArgs := textArgs != ""

		// Determine the "from" user for the quote bubble
		senderJID := ctx.SenderJID()
		senderName := ctx.Event().Info.PushName

		// Launch concurrent data fetching
		var wg sync.WaitGroup
		var mu sync.Mutex

		var quotedSenderJID waTypes.JID
		var quotedSenderName string
		hasQuotedSender := false

		// 1. Fetch quoted sender name concurrently
		wg.Add(1)
		go func() {
			defer wg.Done()
			if quoted != nil && quotedCtxInfo != nil && quotedCtxInfo.GetParticipant() != "" {
				participant := quotedCtxInfo.GetParticipant()
				jid, ok := hc.ParseJID(participant)
				if ok {
					qJID := jid.ToNonAD()
					qName := hc.GetContactName(ctx.Client(), qJID)

					mu.Lock()
					quotedSenderJID = qJID
					quotedSenderName = qName
					hasQuotedSender = true
					mu.Unlock()
				}
			}
		}()

		// 2. Resolve main sender name based on quoted status early
		if senderName == "" {
			senderName = "+" + senderJID.User
		}

		wg.Wait() // we need hasQuotedSender for the next step

		if !hasTextArgs && hasQuotedSender {
			senderJID = quotedSenderJID
			senderName = quotedSenderName
		}

		// 3. Fetch profile picture concurrently
		photoObj := quotePhoto{CreateImageLatters: true}
		wg.Add(1)
		go func(targetJID waTypes.JID) {
			defer wg.Done()
			picInfo, err := ctx.Client().GetProfilePictureInfo(
				stdctx.Background(),
				targetJID,
				&whatsmeow.GetProfilePictureParams{},
			)
			if err == nil && picInfo != nil && picInfo.URL != "" {
				mu.Lock()
				photoObj.URL = picInfo.URL
				photoObj.CreateImageLatters = false
				mu.Unlock()
			}
		}(senderJID)

		// Wait for profile picture
		wg.Wait()

		// Build replyMessage if user typed text AND is replying to a message
		var replyMsg *quoteReplyMessage
		if hasTextArgs && quoted != nil && hasQuotedSender {
			quotedText := hc.GetQuotedText(quoted)
			if quotedText != "" {
				replyMsg = &quoteReplyMessage{
					ChatID: quotedSenderJID.User,
					Name:   quotedSenderName,
					Text:   quotedText,
				}
			}
		}

		// Determine image dimensions based on text length
		size := 520
		if len(text) >= 170 {
			size = 1024
		}

		// Build request
		payload := quoteAPIRequest{
			Type:            "quote",
			Format:          "webp",
			BackgroundColor: "#1b1429",
			Width:           size,
			Height:          size,
			Scale:           2,
			Messages: []quoteMessage{
				{
					Entities: []any{},
					Avatar:   true,
					From: quoteFrom{
						ID:    senderJID.User,
						Name:  senderName,
						Photo: photoObj,
					},
					Text:         text,
					ReplyMessage: replyMsg,
				},
			},
		}

		body, err := json.Marshal(payload)
		if err != nil {
			ctx.Reply(fmt.Sprintf("Failed to build request: %v", err))
			return
		}

		resp, err := http.Post(quoteURL, "application/json", bytes.NewReader(body))
		if err != nil {
			ctx.Reply(fmt.Sprintf("Failed to call quote API: %v", err))
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			ctx.Reply(fmt.Sprintf("Failed to read response: %v", err))
			return
		}

		if resp.StatusCode != http.StatusOK {
			ctx.Reply(fmt.Sprintf("Quote API error (%d): %s", resp.StatusCode, string(respBody)))
			return
		}

		var apiResp quoteAPIResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			ctx.Reply(fmt.Sprintf("Failed to parse response: %v", err))
			return
		}

		// The API may return image in result.image or directly in image
		imageB64 := apiResp.Result.Image
		if imageB64 == "" {
			imageB64 = apiResp.Image
		}
		if imageB64 == "" {
			ctx.Reply("Quote API returned empty image")
			return
		}

		imageData, err := base64.StdEncoding.DecodeString(imageB64)
		if err != nil {
			ctx.Reply(fmt.Sprintf("Failed to decode image: %v", err))
			return
		}

		if sendAsImage {
			ctx.SendImage(imageData, "")
			return
		}

		// Default: send as sticker with EXIF
		stickerData, err := helper.InjectExif(imageData, "Quote", "Bot")
		if err != nil {
			// Fallback: send raw webp as sticker
			ctx.SendSticker(imageData)
			return
		}
		ctx.SendSticker(stickerData)
	},
}
