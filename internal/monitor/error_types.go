package monitor

import "time"

// All error-related types
type ErrorType int

const (
	ErrorCommunication ErrorType = iota
	ErrorStuck
	ErrorNoResponse
	// ...
)

type Priority int

const (
	PriorityCritical Priority = iota
	PriorityHigh
	PriorityMedium
	PriorityLow
)

type ErrorEvent struct {
	Type         ErrorType
	Priority     Priority
	Template     interface{} // Avoid circular import
	Context      map[string]interface{}
	Timestamp    time.Time
	RecoveryFunc func() error
}
