package components

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"jordanella.com/pocket-tcg-go/internal/bot"
)

// OrchestrationCardData holds all bindable data for the orchestration card
type OrchestrationCardData struct {
	// Basic info
	GroupName   binding.String
	Description binding.String
	StartedAt   binding.String

	// Status
	IsActive   binding.Bool
	StatusText binding.String

	// Pool progress
	PoolRemaining binding.Int
	PoolTotal     binding.Int
	PoolProgress  binding.String // Computed: "X/Y"

	// Account pools
	AccountPoolNames binding.String

	// Instances
	ActiveInstancesList binding.String
	OtherInstancesList  binding.String
	AvailableInstances  binding.String

	// Internal state for updates
	lastUpdate time.Time
}

// NewOrchestrationCardData creates new card data with bindings initialized from a BotGroup
func NewOrchestrationCardData(group *bot.BotGroup) *OrchestrationCardData {
	data := &OrchestrationCardData{
		GroupName:           binding.NewString(),
		Description:         binding.NewString(),
		StartedAt:           binding.NewString(),
		IsActive:            binding.NewBool(),
		StatusText:          binding.NewString(),
		PoolRemaining:       binding.NewInt(),
		PoolTotal:           binding.NewInt(),
		PoolProgress:        binding.NewString(),
		AccountPoolNames:    binding.NewString(),
		ActiveInstancesList: binding.NewString(),
		OtherInstancesList:  binding.NewString(),
		AvailableInstances:  binding.NewString(),
	}

	// Initialize from group
	data.UpdateFromGroup(group)

	// Set up computed bindings
	data.setupComputedBindings()

	return data
}

// setupComputedBindings creates derived bindings that update automatically
func (d *OrchestrationCardData) setupComputedBindings() {
	// Pool progress computed binding (updates when either remaining or total changes)
	updatePoolProgress := func() {
		remaining, _ := d.PoolRemaining.Get()
		total, _ := d.PoolTotal.Get()
		d.PoolProgress.Set(fmt.Sprintf("%d/%d", remaining, total))
	}

	d.PoolRemaining.AddListener(binding.NewDataListener(updatePoolProgress))
	d.PoolTotal.AddListener(binding.NewDataListener(updatePoolProgress))

	// Initial calculation
	updatePoolProgress()
}

// UpdateFromGroup refreshes all bindings from the current group state
// This is thread-safe and should be called periodically or on state changes
func (d *OrchestrationCardData) UpdateFromGroup(group *bot.BotGroup) {
	// Update timestamp
	d.lastUpdate = time.Now()

	// Basic info
	d.GroupName.Set(group.Name)

	// For now, we'll use routine name as description (you can enhance this)
	d.Description.Set(fmt.Sprintf("Running routine: %s", group.RoutineName))

	// Status (use exported method for thread safety)
	isRunning := group.IsRunning()

	d.IsActive.Set(isRunning)
	if isRunning {
		d.StatusText.Set("Active")
	} else {
		d.StatusText.Set("Stopped")
	}

	// Pool progress (thread-safe access)
	if group.AccountPool != nil {
		stats := group.AccountPool.GetStats()
		d.PoolRemaining.Set(stats.Available)
		d.PoolTotal.Set(stats.Total)
	} else {
		d.PoolRemaining.Set(0)
		d.PoolTotal.Set(0)
	}

	// Account pool name
	if group.AccountPoolName != "" {
		d.AccountPoolNames.Set(group.AccountPoolName)
	} else {
		d.AccountPoolNames.Set("No pool assigned")
	}

	// Active instances (use exported method for thread safety)
	activeBots := group.GetAllBotInfo()
	activeInstances := make([]int, 0, len(activeBots))
	for id := range activeBots {
		activeInstances = append(activeInstances, id)
	}

	// Format active instances
	if len(activeInstances) > 0 {
		activeStr := formatInstanceList(activeInstances, 5)
		d.ActiveInstancesList.Set(activeStr)
	} else {
		d.ActiveInstancesList.Set("None")
	}

	// Other instances (available but not active)
	otherInstances := make([]int, 0)
	for _, availID := range group.AvailableInstances {
		isActive := false
		for _, activeID := range activeInstances {
			if availID == activeID {
				isActive = true
				break
			}
		}
		if !isActive {
			otherInstances = append(otherInstances, availID)
		}
	}

	if len(otherInstances) > 0 {
		otherStr := formatInstanceList(otherInstances, 5)
		d.OtherInstancesList.Set(otherStr)
	} else {
		d.OtherInstancesList.Set("None")
	}

	// Available instances summary
	d.AvailableInstances.Set(fmt.Sprintf("%d available", len(group.AvailableInstances)))

	// Started at (you may want to track this in BotGroup)
	// For now, showing the orchestration ID as a placeholder
	d.StartedAt.Set(group.OrchestrationID[:8]) // Show first 8 chars of UUID
}

// formatInstanceList formats a list of instance IDs with overflow handling
// maxVisible controls how many instances to show before adding "and X more..."
func formatInstanceList(instances []int, maxVisible int) string {
	if len(instances) == 0 {
		return "None"
	}

	if len(instances) <= maxVisible {
		result := ""
		for i, id := range instances {
			if i > 0 {
				result += ", "
			}
			result += fmt.Sprintf("Instance %d", id)
		}
		return result
	}

	// Show first N and indicate overflow
	result := ""
	for i := 0; i < maxVisible; i++ {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("Instance %d", instances[i])
	}

	remaining := len(instances) - maxVisible
	result += fmt.Sprintf(" and %d more...", remaining)

	return result
}
