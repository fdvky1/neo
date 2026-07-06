package context

import (
	"strings"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// ParseMessageText extracts text content from a message event.
func ParseMessageText(evt *events.Message) string {
	var msg *waE2E.Message
	if evt.IsViewOnce {
		msg = evt.Message.GetViewOnceMessage().GetMessage()
	} else if evt.IsEphemeral {
		msg = evt.Message.GetEphemeralMessage().GetMessage()
	} else {
		msg = evt.Message
	}

	switch {
	case msg.GetConversation() != "":
		return msg.GetConversation()
	case msg.GetExtendedTextMessage().GetText() != "":
		return msg.GetExtendedTextMessage().GetText()
	case msg.GetImageMessage().GetCaption() != "":
		return msg.GetImageMessage().GetCaption()
	case msg.GetVideoMessage().GetCaption() != "":
		return msg.GetVideoMessage().GetCaption()
	default:
		return ""
	}
}

// GetQuotedMessage extracts the quoted message from context info.
func GetQuotedMessage(m *waE2E.Message) *waE2E.Message {
	if m == nil {
		return nil
	}
	switch {
	case m.GetExtendedTextMessage().GetContextInfo() != nil:
		return m.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	case m.GetImageMessage().GetContextInfo() != nil:
		return m.GetImageMessage().GetContextInfo().GetQuotedMessage()
	case m.GetVideoMessage().GetContextInfo() != nil:
		return m.GetVideoMessage().GetContextInfo().GetQuotedMessage()
	case m.GetDocumentMessage().GetContextInfo() != nil:
		return m.GetDocumentMessage().GetContextInfo().GetQuotedMessage()
	case m.GetAudioMessage().GetContextInfo() != nil:
		return m.GetAudioMessage().GetContextInfo().GetQuotedMessage()
	case m.GetStickerMessage().GetContextInfo() != nil:
		return m.GetStickerMessage().GetContextInfo().GetQuotedMessage()
	default:
		return nil
	}
}

// WithReply creates ContextInfo that quotes the original message.
func WithReply(evt *events.Message) *waE2E.ContextInfo {
	return &waE2E.ContextInfo{
		StanzaID:      &evt.Info.ID,
		Participant:   proto.String(evt.Info.Sender.String()),
		QuotedMessage: evt.Message,
	}
}

// ParseJID parses a JID string, adding @s.whatsapp.net if no server is specified.
func ParseJID(arg string) (waTypes.JID, bool) {
	if arg == "" {
		return waTypes.JID{}, false
	}
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return waTypes.NewJID(arg, waTypes.DefaultUserServer), true
	}
	recipient, err := waTypes.ParseJID(arg)
	if err != nil || recipient.User == "" {
		return recipient, false
	}
	return recipient, true
}
