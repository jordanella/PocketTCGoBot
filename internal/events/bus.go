package events

import (
	"fmt"
	"sync"
	"time"
)

// subscription represents a single event subscription
type subscription struct {
	id      SubscriptionID
	handler EventHandler
}

// DefaultEventBus is the default implementation of EventBus
type DefaultEventBus struct {
	// Subscriber management
	subscribers map[EventType][]subscription
	mu          sync.RWMutex

	// Event queue
	eventQueue chan Event
	stopCh     chan struct{}
	wg         sync.WaitGroup

	// Subscription ID generator
	nextSubID SubscriptionID
	subMu     sync.Mutex
}

// NewEventBus creates a new event bus with specified buffer size
func NewEventBus(bufferSize int) *DefaultEventBus {
	bus := &DefaultEventBus{
		subscribers: make(map[EventType][]subscription),
		eventQueue:  make(chan Event, bufferSize),
		stopCh:      make(chan struct{}),
		nextSubID:   1,
	}

	// Start event processor
	bus.wg.Add(1)
	go bus.processEvents()

	return bus
}

// Subscribe registers a handler for a specific event type
func (eb *DefaultEventBus) Subscribe(eventType EventType, handler EventHandler) SubscriptionID {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Generate unique subscription ID
	eb.subMu.Lock()
	subID := eb.nextSubID
	eb.nextSubID++
	eb.subMu.Unlock()

	// Add subscription
	eb.subscribers[eventType] = append(eb.subscribers[eventType], subscription{
		id:      subID,
		handler: handler,
	})

	return subID
}

// Unsubscribe removes a subscription by ID
func (eb *DefaultEventBus) Unsubscribe(id SubscriptionID) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Find and remove subscription
	for eventType, subs := range eb.subscribers {
		for i, sub := range subs {
			if sub.id == id {
				// Remove from slice
				eb.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
				return
			}
		}
	}
}

// Publish sends an event to all subscribers (blocking until queued)
func (eb *DefaultEventBus) Publish(event Event) {
	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	select {
	case eb.eventQueue <- event:
		// Event queued successfully
	case <-eb.stopCh:
		fmt.Printf("[EventBus] Dropped event (bus stopped): %v\n", event.Type)
	}
}

// PublishAsync sends an event asynchronously (non-blocking)
func (eb *DefaultEventBus) PublishAsync(event Event) {
	go eb.Publish(event)
}

// Stop stops the event bus and drains remaining events
func (eb *DefaultEventBus) Stop() {
	close(eb.stopCh)
	eb.wg.Wait()
}

// processEvents runs in a goroutine and dispatches events to handlers
func (eb *DefaultEventBus) processEvents() {
	defer eb.wg.Done()

	for {
		select {
		case event := <-eb.eventQueue:
			eb.dispatch(event)

		case <-eb.stopCh:
			// Drain remaining events before stopping
			for {
				select {
				case event := <-eb.eventQueue:
					eb.dispatch(event)
				default:
					return
				}
			}
		}
	}
}

// dispatch sends an event to all registered handlers
func (eb *DefaultEventBus) dispatch(event Event) {
	// Get handlers with read lock
	eb.mu.RLock()
	subs, exists := eb.subscribers[event.Type]
	if !exists || len(subs) == 0 {
		eb.mu.RUnlock()
		return
	}

	// Make a copy of handlers to avoid holding lock during dispatch
	handlers := make([]EventHandler, len(subs))
	for i, sub := range subs {
		handlers[i] = sub.handler
	}
	eb.mu.RUnlock()

	// Call handlers in goroutines to avoid blocking
	for _, handler := range handlers {
		go eb.safeHandlerCall(handler, event)
	}
}

// safeHandlerCall calls a handler with panic recovery
func (eb *DefaultEventBus) safeHandlerCall(handler EventHandler, event Event) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[EventBus] Handler panic for event %v: %v\n", event.Type, r)
		}
	}()

	handler(event)
}

// GetSubscriberCount returns the number of subscribers for an event type
func (eb *DefaultEventBus) GetSubscriberCount(eventType EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return len(eb.subscribers[eventType])
}

// GetQueueSize returns the current number of events in the queue
func (eb *DefaultEventBus) GetQueueSize() int {
	return len(eb.eventQueue)
}
