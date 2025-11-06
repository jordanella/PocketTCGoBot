package monitor

import "time"

// ErrorHandler type and predefined handlers
type ErrorHandler struct {
	ErrorType     ErrorType
	Priority      Priority
	Template      interface{}
	SearchRegion  interface{}
	CheckInterval time.Duration
	Recovery      func(interface{}) error
	ShouldStop    bool
}

// Factory functions for common handlers

// CommunicationErrorHandler creates a handler for communication errors
func CommunicationErrorHandler() ErrorHandler {
	return ErrorHandler{
		ErrorType:     ErrorCommunication,
		Priority:      PriorityCritical,
		Template:      nil, // TODO: Add error template path
		SearchRegion:  nil,
		CheckInterval: 1 * time.Second,
		Recovery: func(bot interface{}) error {
			// TODO: Implement recovery logic
			return nil
		},
		ShouldStop: true,
	}
}

// LevelUpHandler creates a handler for level up popups
func LevelUpHandler() ErrorHandler {
	return ErrorHandler{
		ErrorType:     ErrorNoResponse,
		Priority:      PriorityHigh,
		Template:      nil, // TODO: Add level up template path
		SearchRegion:  nil,
		CheckInterval: 2 * time.Second,
		Recovery: func(bot interface{}) error {
			// TODO: Implement click to dismiss level up popup
			return nil
		},
		ShouldStop: false,
	}
}

// PrivacyPopupHandler creates a handler for privacy policy popups
func PrivacyPopupHandler() ErrorHandler {
	return ErrorHandler{
		ErrorType:     ErrorNoResponse,
		Priority:      PriorityHigh,
		Template:      nil, // TODO: Add privacy popup template path
		SearchRegion:  nil,
		CheckInterval: 2 * time.Second,
		Recovery: func(bot interface{}) error {
			// TODO: Implement click to dismiss privacy popup
			return nil
		},
		ShouldStop: false,
	}
}
