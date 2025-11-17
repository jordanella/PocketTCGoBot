package events

import "time"

// EventType represents different types of events in the system
type EventType string

const (
	// Orchestration events
	EventTypeGroupCreated       EventType = "group.created"
	EventTypeGroupUpdated       EventType = "group.updated"
	EventTypeGroupDeleted       EventType = "group.deleted"
	EventTypeGroupLaunched      EventType = "group.launched"
	EventTypeGroupStopped       EventType = "group.stopped"
	EventTypeGroupStatusChanged EventType = "group.status_changed"

	// Bot events
	EventTypeBotStarted   EventType = "bot.started"
	EventTypeBotStopped   EventType = "bot.stopped"
	EventTypeBotFailed    EventType = "bot.failed"
	EventTypeBotCompleted EventType = "bot.completed"
	EventTypeBotProgress  EventType = "bot.progress"

	// Instance events
	EventTypeInstanceHealthChanged EventType = "instance.health_changed"
	EventTypeInstanceAssigned      EventType = "instance.assigned"
	EventTypeInstanceReleased      EventType = "instance.released"

	// Account pool events
	EventTypeAccountCheckedOut EventType = "account.checked_out"
	EventTypeAccountReturned   EventType = "account.returned"
	EventTypeAccountCompleted  EventType = "account.completed"
	EventTypeAccountFailed     EventType = "account.failed"
	EventTypePoolRefreshed     EventType = "pool.refreshed"

	// Error events
	EventTypeError EventType = "error"
)

// Event represents a system event with metadata
type Event struct {
	Type      EventType              // Type of event
	Source    string                 // Component that emitted event (e.g., "orchestrator", "health_monitor")
	Timestamp time.Time              // When the event occurred
	Data      map[string]interface{} // Event-specific data
}

// EventHandler is a function that processes an event
type EventHandler func(Event)

// SubscriptionID uniquely identifies a subscription
type SubscriptionID int64

// EventBus defines the interface for event pub/sub
type EventBus interface {
	// Subscribe registers a handler for a specific event type
	Subscribe(eventType EventType, handler EventHandler) SubscriptionID

	// Unsubscribe removes a subscription by ID
	Unsubscribe(id SubscriptionID)

	// Publish sends an event to all subscribers (blocking)
	Publish(event Event)

	// PublishAsync sends an event asynchronously (non-blocking)
	PublishAsync(event Event)

	// Stop stops the event bus and drains remaining events
	Stop()
}

// Helper functions to create common events

// NewGroupLaunchedEvent creates a group launched event
func NewGroupLaunchedEvent(groupName string, launchedBots, requestedBots int, instances []int) Event {
	return Event{
		Type:      EventTypeGroupLaunched,
		Source:    "orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"group_name":     groupName,
			"launched_bots":  launchedBots,
			"requested_bots": requestedBots,
			"instances":      instances,
		},
	}
}

// NewGroupStoppedEvent creates a group stopped event
func NewGroupStoppedEvent(groupName string) Event {
	return Event{
		Type:      EventTypeGroupStopped,
		Source:    "orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"group_name": groupName,
		},
	}
}

// NewBotStartedEvent creates a bot started event
func NewBotStartedEvent(groupName string, instanceID int) Event {
	return Event{
		Type:      EventTypeBotStarted,
		Source:    "orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"group_name":  groupName,
			"instance_id": instanceID,
		},
	}
}

// NewBotStoppedEvent creates a bot stopped event
func NewBotStoppedEvent(groupName string, instanceID int) Event {
	return Event{
		Type:      EventTypeBotStopped,
		Source:    "orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"group_name":  groupName,
			"instance_id": instanceID,
		},
	}
}

// NewBotFailedEvent creates a bot failed event
func NewBotFailedEvent(groupName string, instanceID int, err error) Event {
	return Event{
		Type:      EventTypeBotFailed,
		Source:    "orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"group_name":  groupName,
			"instance_id": instanceID,
			"error":       err.Error(),
		},
	}
}

// NewBotCompletedEvent creates a bot completed event
func NewBotCompletedEvent(groupName string, instanceID int) Event {
	return Event{
		Type:      EventTypeBotCompleted,
		Source:    "orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"group_name":  groupName,
			"instance_id": instanceID,
		},
	}
}

// NewInstanceHealthChangedEvent creates an instance health changed event
func NewInstanceHealthChangedEvent(instanceID int, isReady, wasReady, windowDetected, adbConnected bool) Event {
	return Event{
		Type:      EventTypeInstanceHealthChanged,
		Source:    "health_monitor",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"instance_id":      instanceID,
			"is_ready":         isReady,
			"was_ready":        wasReady,
			"window_detected":  windowDetected,
			"adb_connected":    adbConnected,
		},
	}
}

// NewAccountCheckedOutEvent creates an account checked out event
func NewAccountCheckedOutEvent(poolName, accountID, deviceAccount string) Event {
	return Event{
		Type:      EventTypeAccountCheckedOut,
		Source:    "account_pool",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"pool_name":      poolName,
			"account_id":     accountID,
			"device_account": deviceAccount,
		},
	}
}

// NewPoolRefreshedEvent creates a pool refreshed event
func NewPoolRefreshedEvent(poolName string, totalAccounts, availableAccounts int) Event {
	return Event{
		Type:      EventTypePoolRefreshed,
		Source:    "account_pool",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"pool_name":          poolName,
			"total_accounts":     totalAccounts,
			"available_accounts": availableAccounts,
		},
	}
}

// NewErrorEvent creates an error event
func NewErrorEvent(source, component string, err error, metadata map[string]interface{}) Event {
	data := map[string]interface{}{
		"source":    source,
		"component": component,
		"error":     err.Error(),
	}

	// Merge metadata
	for k, v := range metadata {
		data[k] = v
	}

	return Event{
		Type:      EventTypeError,
		Source:    source,
		Timestamp: time.Now(),
		Data:      data,
	}
}
