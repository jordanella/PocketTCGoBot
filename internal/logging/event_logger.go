package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jordanella.com/pocket-tcg-go/internal/events"
)

// EventLogger subscribes to event bus and logs all events
type EventLogger struct {
	logger         *Logger
	eventBus       events.EventBus
	subscriptionID events.SubscriptionID
	logFile        *os.File
}

// NewEventLogger creates a new event logger
func NewEventLogger(eventBus events.EventBus, logDir string) (*EventLogger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("events_%s.log", timestamp))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	logger := NewLogger("EventLogger")
	logger.AddOutput(logFile)

	el := &EventLogger{
		logger:   logger,
		eventBus: eventBus,
		logFile:  logFile,
	}

	// Subscribe to all event types
	el.subscribeToEvents()

	return el, nil
}

// subscribeToEvents subscribes to all event types
func (el *EventLogger) subscribeToEvents() {
	// Subscribe to all events using a wildcard approach
	// We'll subscribe multiple times for different event types
	eventTypes := []events.EventType{
		events.EventTypeGroupLaunched,
		events.EventTypeGroupStopped,
		events.EventTypeBotStarted,
		events.EventTypeBotStopped,
		events.EventTypeBotFailed,
		events.EventTypeBotCompleted,
		events.EventTypeInstanceHealthChanged,
		events.EventTypePoolRefreshed,
		events.EventTypeAccountCheckedOut,
		events.EventTypeAccountReturned,
		events.EventTypeError,
	}

	for _, eventType := range eventTypes {
		el.eventBus.Subscribe(eventType, el.handleEvent)
	}
}

// handleEvent handles incoming events and logs them
func (el *EventLogger) handleEvent(event events.Event) {
	context := map[string]interface{}{
		"event_type": string(event.Type),
		"source":     event.Source,
	}

	// Add event-specific data to context
	if event.Data != nil {
		for k, v := range event.Data {
			context[k] = v
		}
	}

	// Log the event
	el.logger.InfoWithContext(fmt.Sprintf("Event: %s", event.Type), context)
}

// Close closes the event logger and log file
func (el *EventLogger) Close() error {
	if el.logFile != nil {
		return el.logFile.Close()
	}
	return nil
}
