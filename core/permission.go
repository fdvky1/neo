package core

import (
	"context"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type Permission string

const (
	PermissionGroupOnly   Permission = "GROUP_ONLY"
	PermissionPrivateOnly Permission = "PRIVATE_ONLY"
)

// CheckCommandPermissions returns an error message if the command's
// permissions deny the event context, or empty string if allowed.
func CheckCommandPermissions(cmd *Command, evt *events.Message) string {
	for _, p := range cmd.Permissions {
		switch p {
		case PermissionGroupOnly:
			if !evt.Info.IsGroup {
				return "This command can only be used in groups"
			}
		case PermissionPrivateOnly:
			if evt.Info.IsGroup {
				return "This command can only be used in private chat"
			}
		}
	}
	return ""
}

// sendPermissionDenied sends a plain text reply indicating why the
// command was blocked, so the caller gets feedback instead of silence.
func sendPermissionDenied(client *whatsmeow.Client, evt *events.Message, text string) {
	client.SendMessage(context.Background(), evt.Info.Chat, &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
		},
	})
}
