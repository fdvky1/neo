package context

import (
	"strings"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type Ctx struct {
	client    *whatsmeow.Client
	event     *events.Message
	message   *waE2E.Message
	arguments []string
	prefix    string
	cmd       string
}

func NewCtx(client *whatsmeow.Client, evt *events.Message, prefix, cmd string, args []string) *Ctx {
	var msg *waE2E.Message
	if evt.IsViewOnce {
		msg = evt.Message.GetViewOnceMessage().GetMessage()
	} else if evt.IsEphemeral {
		msg = evt.Message.GetEphemeralMessage().GetMessage()
	} else {
		msg = evt.Message
	}
	return &Ctx{
		client:    client,
		event:     evt,
		message:   msg,
		arguments: args,
		prefix:    prefix,
		cmd:       cmd,
	}
}

func (c *Ctx) Client() *whatsmeow.Client     { return c.client }
func (c *Ctx) Event() *events.Message        { return c.event }
func (c *Ctx) Message() *waE2E.Message       { return c.message }
func (c *Ctx) SenderJID() waTypes.JID        { return c.event.Info.Sender.ToNonAD() }
func (c *Ctx) ChatJID() waTypes.JID          { return c.event.Info.Chat }
func (c *Ctx) IsGroup() bool                 { return c.event.Info.IsGroup }
func (c *Ctx) IsFromMe() bool                { return c.event.Info.IsFromMe }
func (c *Ctx) Number() string                { return c.event.Info.Sender.ToNonAD().String() }
func (c *Ctx) Prefix() string                { return c.prefix }
func (c *Ctx) Cmd() string                   { return c.cmd }
func (c *Ctx) Arguments() []string           { return c.arguments }
func (c *Ctx) FullArgs() string              { return strings.Join(c.arguments, " ") }
func (c *Ctx) Arg(index int) string {
	if index < len(c.arguments) {
		return c.arguments[index]
	}
	return ""
}
