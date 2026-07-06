package context

import (
	stdctx "context"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func (c *Ctx) UploadImage(bytes []byte, caption string) (*waE2E.ImageMessage, error) {
	mime := mimetype.Detect(bytes)
	resp, err := c.client.Upload(stdctx.Background(), bytes, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}
	return &waE2E.ImageMessage{
		Caption:       proto.String(caption),
		Mimetype:      proto.String(mime.String()),
		URL:           &resp.URL,
		DirectPath:    &resp.DirectPath,
		MediaKey:      resp.MediaKey,
		FileEncSHA256: resp.FileEncSHA256,
		FileSHA256:    resp.FileSHA256,
		FileLength:    &resp.FileLength,
	}, nil
}

func (c *Ctx) UploadVideo(bytes []byte, caption string) (*waE2E.VideoMessage, error) {
	mime := mimetype.Detect(bytes)
	resp, err := c.client.Upload(stdctx.Background(), bytes, whatsmeow.MediaVideo)
	if err != nil {
		return nil, err
	}
	return &waE2E.VideoMessage{
		Caption:       proto.String(caption),
		Mimetype:      proto.String(mime.String()),
		URL:           &resp.URL,
		DirectPath:    &resp.DirectPath,
		MediaKey:      resp.MediaKey,
		FileEncSHA256: resp.FileEncSHA256,
		FileSHA256:    resp.FileSHA256,
		FileLength:    &resp.FileLength,
	}, nil
}

func (c *Ctx) UploadAudio(bytes []byte) (*waE2E.AudioMessage, error) {
	mime := mimetype.Detect(bytes)
	resp, err := c.client.Upload(stdctx.Background(), bytes, whatsmeow.MediaAudio)
	if err != nil {
		return nil, err
	}
	return &waE2E.AudioMessage{
		Mimetype:      proto.String(mime.String()),
		URL:           &resp.URL,
		DirectPath:    &resp.DirectPath,
		MediaKey:      resp.MediaKey,
		FileEncSHA256: resp.FileEncSHA256,
		FileSHA256:    resp.FileSHA256,
		FileLength:    &resp.FileLength,
	}, nil
}

func (c *Ctx) UploadDocument(bytes []byte, title, filename string) (*waE2E.DocumentMessage, error) {
	mime := mimetype.Detect(bytes)
	resp, err := c.client.Upload(stdctx.Background(), bytes, whatsmeow.MediaDocument)
	if err != nil {
		return nil, err
	}
	return &waE2E.DocumentMessage{
		Title:         proto.String(title),
		FileName:      proto.String(filename),
		Mimetype:      proto.String(mime.String()),
		URL:           &resp.URL,
		DirectPath:    &resp.DirectPath,
		MediaKey:      resp.MediaKey,
		FileEncSHA256: resp.FileEncSHA256,
		FileSHA256:    resp.FileSHA256,
		FileLength:    &resp.FileLength,
	}, nil
}

func (c *Ctx) UploadSticker(bytes []byte) (*waE2E.StickerMessage, error) {
	resp, err := c.client.Upload(stdctx.Background(), bytes, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}
	return &waE2E.StickerMessage{
		Mimetype:      proto.String("image/webp"),
		URL:           &resp.URL,
		DirectPath:    &resp.DirectPath,
		MediaKey:      resp.MediaKey,
		FileEncSHA256: resp.FileEncSHA256,
		FileSHA256:    resp.FileSHA256,
		FileLength:    &resp.FileLength,
	}, nil
}

func (c *Ctx) SendImage(bytes []byte, caption string) {
	img, err := c.UploadImage(bytes, caption)
	if err != nil {
		c.Reply("Failed to upload image: " + err.Error())
		return
	}
	c.SendMessage(&waE2E.Message{ImageMessage: img})
}

func (c *Ctx) SendVideo(bytes []byte, caption string) {
	vid, err := c.UploadVideo(bytes, caption)
	if err != nil {
		c.Reply("Failed to upload video: " + err.Error())
		return
	}
	c.SendMessage(&waE2E.Message{VideoMessage: vid})
}

func (c *Ctx) SendAudio(bytes []byte) {
	aud, err := c.UploadAudio(bytes)
	if err != nil {
		c.Reply("Failed to upload audio: " + err.Error())
		return
	}
	c.SendMessage(&waE2E.Message{AudioMessage: aud})
}

func (c *Ctx) SendDocument(bytes []byte, title, filename string) {
	doc, err := c.UploadDocument(bytes, title, filename)
	if err != nil {
		c.Reply("Failed to upload document: " + err.Error())
		return
	}
	c.SendMessage(&waE2E.Message{DocumentMessage: doc})
}

func (c *Ctx) SendSticker(bytes []byte) {
	sticker, err := c.UploadSticker(bytes)
	if err != nil {
		c.Reply("Failed to upload sticker: " + err.Error())
		return
	}
	c.SendMessage(&waE2E.Message{StickerMessage: sticker})
}

func matchExtension(ext string, list []string) bool {
	for _, e := range list {
		if strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}
