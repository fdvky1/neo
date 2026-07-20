package group

import (
	"strings"
	"sync"
	
	"neo/core"
	hc "neo/context"
	"neo/database"
)

var (
	// filterCache stores group filters. Map of ChatJID string -> map of Key string -> Content string
	filterCache = make(map[string]map[string]string)
	cacheMu     sync.RWMutex
)

func init() {
	core.Default.Register(&core.Command{
		Name:        "filter",
		Aliases:     []string{"f"},
		Description: "Manage group filters",
		Category:    "Group",
		Permissions: []core.Permission{core.PermissionGroupOnly},
		Run:         HandleFilterCommand,
	})
	
	// Register the middleware to catch incoming messages
	core.Default.Use(FilterMiddleware)
}

// LoadCache loads all filters from DB into memory
func LoadCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	
	var filters []database.Filter
	database.DB.Find(&filters)
	
	for _, f := range filters {
		if filterCache[f.GroupID] == nil {
			filterCache[f.GroupID] = make(map[string]string)
		}
		filterCache[f.GroupID][f.Key] = f.Content
	}
}

func HandleFilterCommand(ctx *hc.Ctx) {
	args := ctx.Arguments()
	if len(args) < 1 {
		ctx.Reply("Usage:\n.filter add key|content\n.filter del key")
		return
	}
	
	action := args[0]
	chatID := ctx.ChatJID().String()
	
	switch action {
	case "add":
		fullArgs := ctx.FullArgs()
		// Remove the 'add ' prefix from full arguments string
		contentStart := strings.Index(fullArgs, "add") + 3
		if contentStart >= len(fullArgs) {
			ctx.Reply("Invalid format. Use: .filter add key|content")
			return
		}
		
		payload := strings.TrimSpace(fullArgs[contentStart:])
		
		parts := strings.SplitN(payload, "|", 2)
		if len(parts) < 2 {
			ctx.Reply("Invalid format. Use: .filter add key|content")
			return
		}
		
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		content := strings.TrimSpace(parts[1])
		
		if key == "" || content == "" {
			ctx.Reply("Key or content cannot be empty.")
			return
		}
		
		// Update DB
		var filter database.Filter
		result := database.DB.Where(database.Filter{Key: key, GroupID: chatID}).FirstOrCreate(&filter, database.Filter{
			Key:     key,
			GroupID: chatID,
			Content: content,
		})
		
		if result.RowsAffected == 0 {
			// Update if exists
			filter.Content = content
			database.DB.Save(&filter)
		}
		
		// Update Cache
		cacheMu.Lock()
		if filterCache[chatID] == nil {
			filterCache[chatID] = make(map[string]string)
		}
		filterCache[chatID][key] = content
		cacheMu.Unlock()
		
		ctx.Reply("Filter '" + key + "' added successfully.")
		
	case "del":
		if len(args) < 2 {
			ctx.Reply("Usage: .filter del key")
			return
		}
		key := strings.ToLower(strings.Join(args[1:], " "))
		
		// Delete from DB
		database.DB.Where(database.Filter{Key: key, GroupID: chatID}).Delete(&database.Filter{})
		
		// Delete from cache
		cacheMu.Lock()
		if groupFilters, ok := filterCache[chatID]; ok {
			delete(groupFilters, key)
		}
		cacheMu.Unlock()
		
		ctx.Reply("Filter '" + key + "' deleted successfully.")
		
	default:
		ctx.Reply("Unknown action. Use add or del.")
	}
}

func FilterMiddleware(ctx *hc.Ctx) bool {
	if !ctx.IsGroup() {
		return false
	}
	
	// Get message text safely handling different message types similarly to core.go
	text := hc.ParseMessageText(ctx.Event())
	
	if text == "" {
		return false
	}
	
	// Don't intercept normal commands
	prefixes := []string{"!", "/", "."} 
	for _, p := range prefixes {
		if strings.HasPrefix(text, p) {
			return false
		}
	}
	
	lowerText := strings.ToLower(strings.TrimSpace(text))
	chatID := ctx.ChatJID().String()
	
	cacheMu.RLock()
	groupFilters, exists := filterCache[chatID]
	if !exists {
		cacheMu.RUnlock()
		return false
	}
	
	content, hasFilter := groupFilters[lowerText]
	cacheMu.RUnlock()
	
	if hasFilter {
		ctx.Reply(content)
		return true // Stop processing, we handled it
	}
	
	return false
}
