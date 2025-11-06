package gui

import (
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

// EventType represents different types of UI events
type EventType int

const (
	EventTypeProgressBarShow EventType = iota
	EventTypeProgressBarHide
	EventTypeLabelUpdate
	EventTypeLogAdd
	EventTypeStatusUpdate
	EventTypeDialogError
	EventTypeDialogInfo
)

// Event represents a UI update event
type Event struct {
	Type      EventType
	Target    string // Widget identifier
	Data      map[string]interface{}
}

// EventBus manages event distribution
type EventBus struct {
	events   chan Event
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
	stopCh   chan struct{}
	app      fyne.App
}

// EventHandler processes events
type EventHandler func(Event)

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		events:   make(chan Event, 100), // Buffered channel
		handlers: make(map[EventType][]EventHandler),
		stopCh:   make(chan struct{}),
	}
}

// Subscribe registers an event handler for a specific event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish sends an event to the bus
func (eb *EventBus) Publish(event Event) {
	select {
	case eb.events <- event:
		log.Printf("[EventBus] Published event: type=%d, target=%s\n", event.Type, event.Target)
	case <-eb.stopCh:
		log.Println("[EventBus] Publish: Bus is stopped, ignoring event")
	default:
		log.Printf("[EventBus] WARNING: Channel full, dropping event: type=%d, target=%s\n", event.Type, event.Target)
	}
}

// Start begins the event processing ticker on main thread
// This MUST be called after the Fyne window is shown
func (eb *EventBus) Start(app fyne.App) {
	eb.app = app

	// Start a ticker that processes events on the main thread
	// This is the key to avoiding threading issues with Fyne
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				eb.processEvents()
			case <-eb.stopCh:
				return
			}
		}
	}()
}

// Stop stops the event bus
func (eb *EventBus) Stop() {
	close(eb.stopCh)
}

// processEvents drains the event queue and dispatches to handlers
func (eb *EventBus) processEvents() {
	// Process all available events in the queue
	processedCount := 0
	for {
		select {
		case event := <-eb.events:
			log.Printf("[EventBus] Processing event: type=%d, target=%s\n", event.Type, event.Target)
			eb.dispatch(event)
			processedCount++
		default:
			// No more events, return
			if processedCount > 0 {
				log.Printf("[EventBus] Processed %d events in this tick\n", processedCount)
			}
			return
		}
	}
}

// dispatch sends events to registered handlers
func (eb *EventBus) dispatch(event Event) {
	eb.mu.RLock()
	handlers, ok := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !ok {
		log.Printf("[EventBus] WARNING: No handlers for event type %d\n", event.Type)
		return
	}

	log.Printf("[EventBus] Dispatching to %d handler(s)\n", len(handlers))
	// Call handlers directly - we're on the ticker goroutine
	for i, handler := range handlers {
		log.Printf("[EventBus] Calling handler %d/%d\n", i+1, len(handlers))
		handler(event)
		log.Printf("[EventBus] Handler %d/%d completed\n", i+1, len(handlers))
	}
}

// Helper functions for common events

// ShowProgressBar creates an event to show a progress bar
func ShowProgressBar(target string) Event {
	return Event{
		Type:   EventTypeProgressBarShow,
		Target: target,
		Data:   make(map[string]interface{}),
	}
}

// HideProgressBar creates an event to hide a progress bar
func HideProgressBar(target string) Event {
	return Event{
		Type:   EventTypeProgressBarHide,
		Target: target,
		Data:   make(map[string]interface{}),
	}
}

// UpdateLabel creates an event to update a label
func UpdateLabel(target string, text string) Event {
	return Event{
		Type:   EventTypeLabelUpdate,
		Target: target,
		Data: map[string]interface{}{
			"text": text,
		},
	}
}

// AddLog creates an event to add a log entry
func AddLog(level LogLevel, instance int, message string) Event {
	return Event{
		Type:   EventTypeLogAdd,
		Target: "log",
		Data: map[string]interface{}{
			"level":    level,
			"instance": instance,
			"message":  message,
		},
	}
}

// UpdateStatus creates an event to update status
func UpdateStatus(target string, status string) Event {
	return Event{
		Type:   EventTypeStatusUpdate,
		Target: target,
		Data: map[string]interface{}{
			"status": status,
		},
	}
}

// ShowErrorDialog creates an event to show an error dialog
func ShowErrorDialog(message string) Event {
	return Event{
		Type:   EventTypeDialogError,
		Target: "dialog",
		Data: map[string]interface{}{
			"message": message,
		},
	}
}

// ShowInfoDialog creates an event to show an info dialog
func ShowInfoDialog(title, message string) Event {
	return Event{
		Type:   EventTypeDialogInfo,
		Target: "dialog",
		Data: map[string]interface{}{
			"title":   title,
			"message": message,
		},
	}
}
