package actions

import (
	"fmt"
	"sync"
)

// SentryManager manages the global sentry lifecycle for a bot
// It handles registration, deduplication, and reference counting for sentries
// across nested routine executions
type SentryManager struct {
	bot    BotInterface
	mu     sync.RWMutex
	active map[string]*ManagedSentry // Key: sentry routine name
}

// ManagedSentry represents a sentry with reference counting
type ManagedSentry struct {
	Sentry      Sentry
	RefCount    int           // Number of active routines using this sentry
	Engine      *SentryEngine // The actual running sentry engine
	MinFrequency int          // Lowest frequency (highest poll rate) requested
}

// NewSentryManager creates a new sentry manager for a bot
func NewSentryManager(bot BotInterface) *SentryManager {
	return &SentryManager{
		bot:    bot,
		active: make(map[string]*ManagedSentry),
	}
}

// Register registers sentries for a routine execution
// If a sentry is already running, increments reference count
// If the new frequency is lower (faster polling), updates the sentry
func (sm *SentryManager) Register(sentries []Sentry) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i := range sentries {
		sentry := &sentries[i]
		key := sentry.Routine

		existing, exists := sm.active[key]
		if exists {
			// Sentry already running - increment reference count
			existing.RefCount++

			// If new frequency is lower (faster polling), restart sentry with new frequency
			if sentry.Frequency < existing.MinFrequency {
				fmt.Printf("Bot %d: Sentry '%s' frequency updated from %ds to %ds (faster polling)\n",
					sm.bot.Instance(), key, existing.MinFrequency, sentry.Frequency)

				// Stop existing engine
				if existing.Engine != nil {
					existing.Engine.Stop()
				}

				// Update frequency and restart
				existing.MinFrequency = sentry.Frequency
				existing.Sentry.Frequency = sentry.Frequency

				// Create new engine with updated frequency
				engine := NewSentryEngine(sm.bot, []Sentry{existing.Sentry})
				if err := engine.Start(); err != nil {
					return fmt.Errorf("failed to restart sentry '%s': %w", key, err)
				}
				existing.Engine = engine
			}

			fmt.Printf("Bot %d: Sentry '%s' already active (refcount: %d)\n",
				sm.bot.Instance(), key, existing.RefCount)
		} else {
			// New sentry - load and start
			fmt.Printf("Bot %d: Starting new sentry '%s' (frequency: %ds)\n",
				sm.bot.Instance(), key, sentry.Frequency)

			// Load the sentry routine
			routineRegistry := sm.bot.Routines()
			if routineRegistry == nil {
				return fmt.Errorf("routine registry not available")
			}

			builder, err := routineRegistry.Get(sentry.Routine)
			if err != nil {
				return fmt.Errorf("sentry routine '%s' not found: %w", sentry.Routine, err)
			}
			sentry.SetRoutineBuilder(builder)

			// Create and start sentry engine
			engine := NewSentryEngine(sm.bot, []Sentry{*sentry})
			if err := engine.Start(); err != nil {
				return fmt.Errorf("failed to start sentry '%s': %w", key, err)
			}

			// Store managed sentry
			sm.active[key] = &ManagedSentry{
				Sentry:       *sentry,
				RefCount:     1,
				Engine:       engine,
				MinFrequency: sentry.Frequency,
			}
		}
	}

	return nil
}

// Unregister unregisters sentries when a routine completes
// Decrements reference count and stops sentry if count reaches zero
func (sm *SentryManager) Unregister(sentries []Sentry) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i := range sentries {
		sentry := &sentries[i]
		key := sentry.Routine

		existing, exists := sm.active[key]
		if !exists {
			// Sentry not found - this shouldn't happen but handle gracefully
			fmt.Printf("Bot %d: Warning - attempted to unregister non-existent sentry '%s'\n",
				sm.bot.Instance(), key)
			continue
		}

		// Decrement reference count
		existing.RefCount--
		fmt.Printf("Bot %d: Sentry '%s' unregistered (refcount: %d)\n",
			sm.bot.Instance(), key, existing.RefCount)

		if existing.RefCount <= 0 {
			// No more routines using this sentry - stop it
			fmt.Printf("Bot %d: Stopping sentry '%s' (no more active routines)\n",
				sm.bot.Instance(), key)

			if existing.Engine != nil {
				existing.Engine.Stop()
			}

			// Remove from active map
			delete(sm.active, key)
		}
	}
}

// StopAll stops all active sentries
// Used during bot shutdown
func (sm *SentryManager) StopAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	fmt.Printf("Bot %d: Stopping all sentries (%d active)\n", sm.bot.Instance(), len(sm.active))

	for key, managed := range sm.active {
		if managed.Engine != nil {
			managed.Engine.Stop()
		}
		delete(sm.active, key)
	}
}

// GetActiveCount returns the number of active sentries
func (sm *SentryManager) GetActiveCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.active)
}

// GetSentryInfo returns information about active sentries (for debugging)
func (sm *SentryManager) GetSentryInfo() map[string]SentryInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	info := make(map[string]SentryInfo)
	for key, managed := range sm.active {
		info[key] = SentryInfo{
			Routine:      key,
			RefCount:     managed.RefCount,
			Frequency:    managed.MinFrequency,
			Severity:     string(managed.Sentry.Severity),
			OnSuccess:    string(managed.Sentry.OnSuccess),
			OnFailure:    string(managed.Sentry.OnFailure),
		}
	}
	return info
}

// SentryInfo holds display information about an active sentry
type SentryInfo struct {
	Routine   string
	RefCount  int
	Frequency int
	Severity  string
	OnSuccess string
	OnFailure string
}
