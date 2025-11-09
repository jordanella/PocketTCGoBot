package actions

import (
	"fmt"
	"strconv"
	"sync"
)

// VariableStore is a thread-safe implementation of VariableStoreInterface
type VariableStore struct {
	mu         sync.RWMutex
	vars       map[string]string
	persistent map[string]bool // Tracks which variables should persist between routine iterations
}

// NewVariableStore creates a new variable store
func NewVariableStore() *VariableStore {
	return &VariableStore{
		vars:       make(map[string]string),
		persistent: make(map[string]bool),
	}
}

func (vs *VariableStore) Set(name string, value string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.vars[name] = value
}

func (vs *VariableStore) Get(name string) (string, bool) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	val, ok := vs.vars[name]
	return val, ok
}

func (vs *VariableStore) Has(name string) bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	_, ok := vs.vars[name]
	return ok
}

func (vs *VariableStore) Delete(name string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	delete(vs.vars, name)
}

func (vs *VariableStore) Clear() {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.vars = make(map[string]string)
	vs.persistent = make(map[string]bool)
}

// ClearNonPersistent clears all variables except those marked as persistent
// This is used when reinitializing routines to preserve persistent variables
func (vs *VariableStore) ClearNonPersistent() {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Keep only persistent variables
	for name := range vs.vars {
		if !vs.persistent[name] {
			delete(vs.vars, name)
		}
	}
}

// MarkPersistent marks a variable as persistent (won't be cleared on reinitialization)
func (vs *VariableStore) MarkPersistent(name string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.persistent[name] = true
}

// IsPersistent checks if a variable is marked as persistent
func (vs *VariableStore) IsPersistent(name string) bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.persistent[name]
}

func (vs *VariableStore) GetAll() map[string]string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	// Return a copy to prevent external modification
	copy := make(map[string]string, len(vs.vars))
	for k, v := range vs.vars {
		copy[k] = v
	}
	return copy
}

// SetVariable sets a variable to a specific value
type SetVariable struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func (a *SetVariable) Validate(ab *ActionBuilder) error {
	if a.Name == "" {
		return fmt.Errorf("SetVariable: name is required")
	}
	// Value can be empty string, so no validation needed
	return nil
}

func (a *SetVariable) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("SetVariable (%s = %s)", a.Name, a.Value),
		execute: func(bot BotInterface) error {
			// Interpolate the value if it contains variables
			value, err := InterpolateString(a.Value, bot)
			if err != nil {
				return fmt.Errorf("SetVariable: %w", err)
			}
			bot.Variables().Set(a.Name, value)
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// GetVariable gets a variable value and optionally sets it to another variable
// This is mainly useful for debugging or storing to a different variable
type GetVariable struct {
	Name   string `yaml:"name"`
	Target string `yaml:"target,omitempty"` // Optional: store result in this variable
}

func (a *GetVariable) Validate(ab *ActionBuilder) error {
	if a.Name == "" {
		return fmt.Errorf("GetVariable: name is required")
	}
	return nil
}

func (a *GetVariable) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("GetVariable (%s)", a.Name),
		execute: func(bot BotInterface) error {
			value, ok := bot.Variables().Get(a.Name)
			if !ok {
				return fmt.Errorf("GetVariable: variable '%s' not found", a.Name)
			}
			// If target is specified, store the value there
			if a.Target != "" {
				bot.Variables().Set(a.Target, value)
			}
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// Increment increments a numeric variable by a specified amount (default 1)
type Increment struct {
	Name   string `yaml:"name"`
	Amount string `yaml:"amount,omitempty"` // Default: "1"
}

func (a *Increment) Validate(ab *ActionBuilder) error {
	if a.Name == "" {
		return fmt.Errorf("Increment: name is required")
	}
	// Validate amount is a valid number if provided
	if a.Amount != "" {
		if _, err := strconv.Atoi(a.Amount); err != nil {
			return fmt.Errorf("Increment: amount must be a valid integer, got '%s'", a.Amount)
		}
	}
	return nil
}

func (a *Increment) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("Increment (%s)", a.Name),
		execute: func(bot BotInterface) error {
			// Get current value
			currentStr, ok := bot.Variables().Get(a.Name)
			if !ok {
				// Variable doesn't exist, initialize to 0
				currentStr = "0"
			}

			current, err := strconv.Atoi(currentStr)
			if err != nil {
				return fmt.Errorf("Increment: variable '%s' is not a valid integer: %s", a.Name, currentStr)
			}

			// Determine increment amount
			amount := 1
			if a.Amount != "" {
				amount, err = strconv.Atoi(a.Amount)
				if err != nil {
					return fmt.Errorf("Increment: amount is not a valid integer: %s", a.Amount)
				}
			}

			// Set new value
			newValue := current + amount
			bot.Variables().Set(a.Name, strconv.Itoa(newValue))
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// Decrement decrements a numeric variable by a specified amount (default 1)
type Decrement struct {
	Name   string `yaml:"name"`
	Amount string `yaml:"amount,omitempty"` // Default: "1"
}

func (a *Decrement) Validate(ab *ActionBuilder) error {
	if a.Name == "" {
		return fmt.Errorf("Decrement: name is required")
	}
	// Validate amount is a valid number if provided
	if a.Amount != "" {
		if _, err := strconv.Atoi(a.Amount); err != nil {
			return fmt.Errorf("Decrement: amount must be a valid integer, got '%s'", a.Amount)
		}
	}
	return nil
}

func (a *Decrement) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("Decrement (%s)", a.Name),
		execute: func(bot BotInterface) error {
			// Get current value
			currentStr, ok := bot.Variables().Get(a.Name)
			if !ok {
				// Variable doesn't exist, initialize to 0
				currentStr = "0"
			}

			current, err := strconv.Atoi(currentStr)
			if err != nil {
				return fmt.Errorf("Decrement: variable '%s' is not a valid integer: %s", a.Name, currentStr)
			}

			// Determine decrement amount
			amount := 1
			if a.Amount != "" {
				amount, err = strconv.Atoi(a.Amount)
				if err != nil {
					return fmt.Errorf("Decrement: amount is not a valid integer: %s", a.Amount)
				}
			}

			// Set new value
			newValue := current - amount
			bot.Variables().Set(a.Name, strconv.Itoa(newValue))
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
