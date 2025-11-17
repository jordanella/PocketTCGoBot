package bot

import "jordanella.com/pocket-tcg-go/internal/accountpool"

// BotGroupManagerAdapter adapts a BotGroup to provide Manager-like functionality
// This allows orchestrator-created bots to access the group's account pool
type BotGroupManagerAdapter struct {
	group *BotGroup
}

// NewBotGroupManagerAdapter creates a manager adapter for a bot group
func NewBotGroupManagerAdapter(group *BotGroup) *BotGroupManagerAdapter {
	return &BotGroupManagerAdapter{
		group: group,
	}
}

// AccountPool returns the bot group's account pool
func (a *BotGroupManagerAdapter) AccountPool() accountpool.AccountPool {
	return a.group.AccountPool
}
