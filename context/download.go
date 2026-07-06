package context

import (
	stdctx "context"
	"fmt"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
)

func (c *Ctx) Download() ([]byte, error) {
	msg := getDownloadable(c.message)
	if msg == nil {
		return nil, fmt.Errorf("message has no downloadable media")
	}
	return c.client.Download(stdctx.Background(), msg)
}

func (c *Ctx) DownloadQuoted() ([]byte, error) {
	quoted := GetQuotedMessage(c.event.Message)
	if quoted == nil {
		return nil, fmt.Errorf("no quoted message")
	}
	msg := getDownloadable(quoted)
	if msg == nil {
		return nil, fmt.Errorf("quoted message has no downloadable media")
	}
	return c.client.Download(stdctx.Background(), msg)
}

func (c *Ctx) DownloadMessage(msg whatsmeow.DownloadableMessage) ([]byte, error) {
	return c.client.Download(stdctx.Background(), msg)
}

func getDownloadable(m *waE2E.Message) whatsmeow.DownloadableMessage {
	switch {
	case m.GetImageMessage() != nil:
		return m.GetImageMessage()
	case m.GetVideoMessage() != nil:
		return m.GetVideoMessage()
	case m.GetAudioMessage() != nil:
		return m.GetAudioMessage()
	case m.GetDocumentMessage() != nil:
		return m.GetDocumentMessage()
	case m.GetStickerMessage() != nil:
		return m.GetStickerMessage()
	default:
		return nil
	}
}
