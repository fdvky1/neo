package context

import (
	"strings"
	stdctx "context"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"go.mau.fi/whatsmeow"
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

// GetQuotedStanzaID returns the stanza ID of the message being replied to, if any.
func GetQuotedStanzaID(m *waE2E.Message) string {
	if m == nil {
		return ""
	}
	switch {
	case m.GetExtendedTextMessage().GetContextInfo() != nil:
		return m.GetExtendedTextMessage().GetContextInfo().GetStanzaID()
	case m.GetImageMessage().GetContextInfo() != nil:
		return m.GetImageMessage().GetContextInfo().GetStanzaID()
	case m.GetVideoMessage().GetContextInfo() != nil:
		return m.GetVideoMessage().GetContextInfo().GetStanzaID()
	case m.GetDocumentMessage().GetContextInfo() != nil:
		return m.GetDocumentMessage().GetContextInfo().GetStanzaID()
	case m.GetAudioMessage().GetContextInfo() != nil:
		return m.GetAudioMessage().GetContextInfo().GetStanzaID()
	case m.GetStickerMessage().GetContextInfo() != nil:
		return m.GetStickerMessage().GetContextInfo().GetStanzaID()
	default:
		return ""
	}
}

// GetContextInfo extracts the ContextInfo from any message type.
func GetContextInfo(m *waE2E.Message) *waE2E.ContextInfo {
	if m == nil {
		return nil
	}
	switch {
	case m.GetExtendedTextMessage().GetContextInfo() != nil:
		return m.GetExtendedTextMessage().GetContextInfo()
	case m.GetImageMessage().GetContextInfo() != nil:
		return m.GetImageMessage().GetContextInfo()
	case m.GetVideoMessage().GetContextInfo() != nil:
		return m.GetVideoMessage().GetContextInfo()
	case m.GetDocumentMessage().GetContextInfo() != nil:
		return m.GetDocumentMessage().GetContextInfo()
	case m.GetAudioMessage().GetContextInfo() != nil:
		return m.GetAudioMessage().GetContextInfo()
	case m.GetStickerMessage().GetContextInfo() != nil:
		return m.GetStickerMessage().GetContextInfo()
	default:
		return nil
	}
}

// GetQuotedText extracts text from a quoted message.
func GetQuotedText(m *waE2E.Message) string {
	if m == nil {
		return ""
	}
	switch {
	case m.GetConversation() != "":
		return m.GetConversation()
	case m.GetExtendedTextMessage().GetText() != "":
		return m.GetExtendedTextMessage().GetText()
	case m.GetImageMessage().GetCaption() != "":
		return m.GetImageMessage().GetCaption()
	case m.GetVideoMessage().GetCaption() != "":
		return m.GetVideoMessage().GetCaption()
	default:
		return ""
	}
}
// GetContactName returns a display name for the given JID from the store.
func GetContactName(client *whatsmeow.Client, jid waTypes.JID) string {
	if client == nil || client.Store == nil || client.Store.Contacts == nil {
		return "+" + jid.User
	}
	contact, err := client.Store.Contacts.GetContact(stdctx.Background(), jid)
	if err == nil && contact.Found {
		if contact.PushName != "" {
			return contact.PushName
		}
		if contact.FullName != "" {
			return contact.FullName
		}
		if contact.BusinessName != "" {
			return contact.BusinessName
		}
	}
	return "+" + jid.User
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
