package core

import (
	"fmt"
	"strings"
	"sync"

	hc "neo/context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type CommandFunc func(ctx *hc.Ctx)

type Command struct {
	Name        string
	Aliases     []string
	Description string
	Category    string
	Permissions []Permission
	Run         CommandFunc
}

// Default is the global command muxer. Each command registers itself
// via init() — see neo/cmds for the registration pattern.
var Default = New()

type Middleware func(ctx *hc.Ctx) bool

type Handler struct {
	mu          sync.RWMutex
	commands    map[string]*Command
	prefixes    []string
	middlewares []Middleware
}

func (h *Handler) Use(fn Middleware) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.middlewares = append(h.middlewares, fn)
}

func New(prefixes ...string) *Handler {
	if len(prefixes) == 0 {
		prefixes = []string{"!", "/", "."}
	}
	return &Handler{
		commands: make(map[string]*Command),
		prefixes: prefixes,
	}
}

func (h *Handler) Register(cmds ...*Command) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, cmd := range cmds {
		if cmd.Name == "" {
			panic("core: command name cannot be empty")
		}
		if cmd.Run == nil {
			panic(fmt.Sprintf("core: command %q has no Run function", cmd.Name))
		}
		h.commands[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			h.commands[alias] = cmd
		}
	}
}

func (h *Handler) GetCommands() []*Command {
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := make(map[string]bool)
	var result []*Command
	for _, cmd := range h.commands {
		if !seen[cmd.Name] {
			seen[cmd.Name] = true
			result = append(result, cmd)
		}
	}
	return result
}

func (h *Handler) EventHandler(client *whatsmeow.Client) func(any) {
	return func(rawEvt any) {
		evt, ok := rawEvt.(*events.Message)
		if !ok {
			return
		}
		go h.processMessage(client, evt)
	}
}

func (h *Handler) processMessage(client *whatsmeow.Client, evt *events.Message) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[CMD] panic from %s: %v\n", formatSenderInfo(evt), r)
		}
	}()

	text := hc.ParseMessageText(evt)
	prefix, cmd, args, isCmd := parseCommand(text, h.prefixes)
	ctx := hc.NewCtx(client, evt, prefix, cmd, args)

	h.mu.RLock()
	middlewares := append([]Middleware(nil), h.middlewares...)
	h.mu.RUnlock()
	for _, middleware := range middlewares {
		if middleware(ctx) {
			return
		}
	}

	if !isCmd {
		return
	}

	h.mu.RLock()
	command, exists := h.commands[cmd]
	h.mu.RUnlock()
	if !exists {
		return
	}

	if msg := CheckCommandPermissions(command, evt); msg != "" {
		sendPermissionDenied(client, evt, msg)
		return
	}

	fmt.Printf("[CMD] %s > %s%s\n", formatSenderInfo(evt), prefix, cmd)
	command.Run(ctx)
}

func parseCommand(text string, prefixes []string) (prefix, cmd string, args []string, ok bool) {
	if text == "" {
		return
	}
	for _, p := range prefixes {
		if strings.HasPrefix(text, p) {
			rest := text[len(p):]
			parts := strings.Fields(rest)
			if len(parts) == 0 {
				return
			}
			return p, strings.ToLower(parts[0]), parts[1:], true
		}
	}
	return
}

func formatSenderInfo(evt *events.Message) string {
	sender := evt.Info.Sender.ToNonAD().String()
	if evt.Info.IsGroup {
		return fmt.Sprintf("[%s] %s", evt.Info.Chat.String(), sender)
	}
	return sender
}
