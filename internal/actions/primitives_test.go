package actions

import (
	"context"
	"testing"
	"time"

	"jordanella.com/pocket-tcg-go/internal/adb"
	"jordanella.com/pocket-tcg-go/internal/cv"
	"jordanella.com/pocket-tcg-go/internal/monitor"
)

// mockBotConfig implements ConfigInterface for testing
type mockBotConfig struct {
	delayMs int
}

func (m mockBotConfig) Actions() ActionsConfig {
	return mockActionsConfig{delayMs: m.delayMs}
}

// mockActionsConfig implements ActionsConfig for testing
type mockActionsConfig struct {
	delayMs int
}

func (m mockActionsConfig) GetDelayBetweenActions() int {
	return m.delayMs
}

func (m mockActionsConfig) GetScreenshotDelay() int {
	return 1000
}

// mockBot implements BotInterface for testing
type mockBot struct {
	config ConfigInterface
}

func (m *mockBot) ADB() *adb.Controller {
	return nil
}

func (m *mockBot) CV() *cv.Service {
	return nil
}

func (m *mockBot) ErrorMonitor() *monitor.ErrorMonitor {
	return nil
}

func (m *mockBot) Config() ConfigInterface {
	return m.config
}

func (m *mockBot) Context() context.Context {
	return context.Background()
}

func (m *mockBot) IsPaused() bool {
	return false
}

func (m *mockBot) IsStopped() bool {
	return false
}

func TestDelay(t *testing.T) {
	tests := []struct {
		name        string
		configDelay int // milliseconds in config
		multiplier  int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name:        "Delay with multiplier 1",
			configDelay: 100,
			multiplier:  1,
			expectedMin: 100 * time.Millisecond,
			expectedMax: 150 * time.Millisecond, // Allow some tolerance
		},
		{
			name:        "Delay with multiplier 3",
			configDelay: 100,
			multiplier:  3,
			expectedMin: 300 * time.Millisecond,
			expectedMax: 350 * time.Millisecond,
		},
		{
			name:        "Delay with multiplier 0",
			configDelay: 100,
			multiplier:  0,
			expectedMin: 0 * time.Millisecond,
			expectedMax: 50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot := &mockBot{
				config: mockBotConfig{delayMs: tt.configDelay},
			}

			ab := &ActionBuilder{
				bot:   bot,
				steps: []Step{},
				ctx:   context.Background(),
			}

			// Build action with Delay
			ab.Delay(tt.multiplier)

			// Measure execution time
			start := time.Now()
			err := ab.Execute()
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Execute() failed: %v", err)
			}

			if duration < tt.expectedMin {
				t.Errorf("Delay was too short: got %v, expected at least %v", duration, tt.expectedMin)
			}

			if duration > tt.expectedMax {
				t.Errorf("Delay was too long: got %v, expected at most %v", duration, tt.expectedMax)
			}
		})
	}
}

func TestSleep(t *testing.T) {
	bot := &mockBot{
		config: mockBotConfig{delayMs: 100},
	}

	ab := &ActionBuilder{
		bot:   bot,
		steps: []Step{},
		ctx:   context.Background(),
	}

	// Build action with Sleep
	sleepDuration := 200 * time.Millisecond
	ab.Sleep(sleepDuration)

	// Measure execution time
	start := time.Now()
	err := ab.Execute()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	expectedMin := sleepDuration
	expectedMax := sleepDuration + 50*time.Millisecond

	if duration < expectedMin {
		t.Errorf("Sleep was too short: got %v, expected at least %v", duration, expectedMin)
	}

	if duration > expectedMax {
		t.Errorf("Sleep was too long: got %v, expected at most %v", duration, expectedMax)
	}
}

func TestTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeoutSeconds  int
		expectedTimeout time.Duration
	}{
		{
			name:            "Timeout 1 second",
			timeoutSeconds:  1,
			expectedTimeout: 1 * time.Second,
		},
		{
			name:            "Timeout 5 seconds",
			timeoutSeconds:  5,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "Timeout 30 seconds",
			timeoutSeconds:  30,
			expectedTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot := &mockBot{
				config: mockBotConfig{delayMs: 100},
			}

			ab := &ActionBuilder{
				bot:   bot,
				steps: []Step{},
				ctx:   context.Background(),
			}

			// Set timeout
			ab.Timeout(tt.timeoutSeconds)

			// Verify timeout was set correctly
			if ab.timeout != tt.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tt.expectedTimeout, ab.timeout)
			}
		})
	}
}

func TestTimeoutChaining(t *testing.T) {
	bot := &mockBot{
		config: mockBotConfig{delayMs: 100},
	}

	ab := &ActionBuilder{
		bot:   bot,
		steps: []Step{},
		ctx:   context.Background(),
	}

	// Test chaining
	ab.Timeout(10).
		Sleep(100 * time.Millisecond).
		Delay(2)

	if ab.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", ab.timeout)
	}

	if len(ab.steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(ab.steps))
	}
}
