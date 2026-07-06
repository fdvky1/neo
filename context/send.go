package context

import (
	stdctx "context"

	waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func (c *Ctx) Reply(text string) {
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      &c.event.Info.ID,
				Participant:   proto.String(c.event.Info.Sender.String()),
				QuotedMessage: c.event.Message,
			},
		},
	}
	c.client.SendMessage(stdctx.Background(), c.event.Info.Chat, msg)
}

func (c *Ctx) Send(text string) {
	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
		},
	}
	c.client.SendMessage(stdctx.Background(), c.event.Info.Chat, msg)
}

func (c *Ctx) SendMessage(msg *waE2E.Message) {
	c.client.SendMessage(stdctx.Background(), c.event.Info.Chat, msg)
}

func (c *Ctx) React(emoji string) {
	msg := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waCommon.MessageKey{
				RemoteJID: proto.String(c.event.Info.Chat.String()),
				FromMe:    proto.Bool(c.event.Info.IsFromMe),
				ID:        proto.String(c.event.Info.ID),
			},
			Text:              proto.String(emoji),
			SenderTimestampMS: proto.Int64(c.event.Info.Timestamp.UnixMilli()),
		},
	}
	c.client.SendMessage(stdctx.Background(), c.event.Info.Chat, msg)
}

func (c *Ctx) Edit(text string) error {
	msgKey := &waCommon.MessageKey{
		RemoteJID: proto.String(c.ChatJID().String()),
		FromMe:    proto.Bool(true),
		ID:        proto.String(c.event.Info.ID),
	}

	if c.message.GetExtendedTextMessage() != nil {
		c.message.ExtendedTextMessage.Text = proto.String(text)
	} else {
		c.message = &waE2E.Message{
			Conversation: proto.String(text),
		}
	}

	editMsg := &waE2E.Message{
		ProtocolMessage: &waE2E.ProtocolMessage{
			Key:           msgKey,
			Type:          waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
			EditedMessage: c.message,
		},
	}

	_, err := c.client.SendMessage(stdctx.Background(), c.event.Info.Chat, editMsg)
	return err
}

func (c *Ctx) Revoke() {
	_, _ = c.client.SendMessage(
		stdctx.Background(),
		c.event.Info.Chat,
		c.client.BuildRevoke(c.event.Info.Chat, c.event.Info.Sender, c.event.Info.ID),
	)
}

func (c *Ctx) MarkRead() {
	_ = c.client.MarkRead(
		stdctx.Background(),
		[]string{c.event.Info.ID},
		c.event.Info.Timestamp,
		c.event.Info.Chat,
		c.event.Info.Sender,
	)
}

func (c *Ctx) SendPresence(composing bool) {
	state := waTypes.ChatPresencePaused
	if composing {
		state = waTypes.ChatPresenceComposing
	}
	_ = c.client.SubscribePresence(stdctx.Background(), c.event.Info.Chat)
	_ = c.client.SendChatPresence(stdctx.Background(), c.event.Info.Chat, state, waTypes.ChatPresenceMediaText)
}
