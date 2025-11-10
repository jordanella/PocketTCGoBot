package bot

import (
	"sync"
	"sync/atomic"
)

// RoutineExecutionState represents the current state of routine execution
type RoutineExecutionState int32

const (
	StateIdle RoutineExecutionState = iota
	StateRunning
	StatePaused     // Paused by sentry or user
	StateStopped    // Force stopped by sentry or user
	StateCompleted  // Normal completion
)

// RoutineController manages the execution state and control signals for routines
type RoutineController struct {
	state        atomic.Int32       // Current execution state
	pauseChan    chan struct{}      // Signal to pause execution
	resumeChan   chan struct{}      // Signal to resume execution
	stopChan     chan struct{}      // Signal to force stop execution
	mu           sync.RWMutex       // Protects channel recreation
	currentState RoutineExecutionState // Cached state for channel decisions
}

// NewRoutineController creates a new routine controller
func NewRoutineController() *RoutineController {
	rc := &RoutineController{
		pauseChan:  make(chan struct{}, 1),
		resumeChan: make(chan struct{}, 1),
		stopChan:   make(chan struct{}, 1),
	}
	rc.state.Store(int32(StateIdle))
	rc.currentState = StateIdle
	return rc
}

// GetState returns the current execution state
func (rc *RoutineController) GetState() interface{} {
	return RoutineExecutionState(rc.state.Load())
}

// IsRunning returns true if a routine is currently running
func (rc *RoutineController) IsRunning() bool {
	return rc.GetState().(RoutineExecutionState) == StateRunning
}

// IsPaused returns true if execution is paused
func (rc *RoutineController) IsPaused() bool {
	return rc.GetState().(RoutineExecutionState) == StatePaused
}

// IsStopped returns true if execution is stopped
func (rc *RoutineController) IsStopped() bool {
	state := rc.GetState().(RoutineExecutionState)
	return state == StateStopped || state == StateCompleted
}

// SetRunning sets the state to running
func (rc *RoutineController) SetRunning() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.state.Store(int32(StateRunning))
	rc.currentState = StateRunning
}

// SetCompleted sets the state to completed
func (rc *RoutineController) SetCompleted() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.state.Store(int32(StateCompleted))
	rc.currentState = StateCompleted
}

// SetIdle sets the state to idle
func (rc *RoutineController) SetIdle() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.state.Store(int32(StateIdle))
	rc.currentState = StateIdle
}

// Pause pauses the routine execution
// Returns true if pause was initiated, false if already paused/stopped
func (rc *RoutineController) Pause() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	currentState := RoutineExecutionState(rc.state.Load())
	if currentState != StateRunning {
		return false // Can only pause running routines
	}

	rc.state.Store(int32(StatePaused))
	rc.currentState = StatePaused

	// Non-blocking send to pause channel
	select {
	case rc.pauseChan <- struct{}{}:
	default:
		// Channel already has signal
	}

	return true
}

// Resume resumes the routine execution
// Returns true if resume was initiated, false if not paused
func (rc *RoutineController) Resume() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	currentState := RoutineExecutionState(rc.state.Load())
	if currentState != StatePaused {
		return false // Can only resume paused routines
	}

	rc.state.Store(int32(StateRunning))
	rc.currentState = StateRunning

	// Non-blocking send to resume channel
	select {
	case rc.resumeChan <- struct{}{}:
	default:
		// Channel already has signal
	}

	return true
}

// ForceStop force stops the routine execution
// Returns true if stop was initiated
func (rc *RoutineController) ForceStop() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.state.Store(int32(StateStopped))
	rc.currentState = StateStopped

	// Non-blocking send to stop channel
	select {
	case rc.stopChan <- struct{}{}:
	default:
		// Channel already has signal
	}

	return true
}

// Reset resets the controller to idle state
// Should be called before starting a new routine
func (rc *RoutineController) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.state.Store(int32(StateIdle))
	rc.currentState = StateIdle

	// Drain channels
	select {
	case <-rc.pauseChan:
	default:
	}
	select {
	case <-rc.resumeChan:
	default:
	}
	select {
	case <-rc.stopChan:
	default:
	}
}

// PauseChan returns the pause signal channel
func (rc *RoutineController) PauseChan() <-chan struct{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.pauseChan
}

// ResumeChan returns the resume signal channel
func (rc *RoutineController) ResumeChan() <-chan struct{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.resumeChan
}

// StopChan returns the stop signal channel
func (rc *RoutineController) StopChan() <-chan struct{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.stopChan
}

// WaitWhilePaused blocks until the routine is resumed or stopped
// Returns true if resumed, false if stopped
func (rc *RoutineController) WaitWhilePaused() bool {
	if !rc.IsPaused() {
		return true // Not paused, continue
	}

	// Wait for either resume or stop signal
	select {
	case <-rc.ResumeChan():
		return true // Resumed
	case <-rc.StopChan():
		return false // Stopped
	}
}

// CheckPauseOrStop checks if execution should pause or stop
// Returns true if execution should continue, false if stopped
// If paused, blocks until resumed or stopped
func (rc *RoutineController) CheckPauseOrStop() bool {
	state := rc.GetState().(RoutineExecutionState)

	switch state {
	case StateStopped, StateCompleted:
		return false // Stop execution

	case StatePaused:
		// Wait for resume or stop
		return rc.WaitWhilePaused()

	default:
		// Check for new pause/stop signals
		select {
		case <-rc.PauseChan():
			// Received pause signal, wait for resume
			return rc.WaitWhilePaused()
		case <-rc.StopChan():
			return false // Received stop signal
		default:
			return true // Continue execution
		}
	}
}
