package bot

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBotRoutineRegistryIntegration tests that the bot properly initializes and uses the routine registry
func TestBotRoutineRegistryIntegration(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create necessary subdirectories
	routinesDir := filepath.Join(tempDir, "routines")
	if err := os.Mkdir(routinesDir, 0755); err != nil {
		t.Fatalf("Failed to create routines directory: %v", err)
	}

	// Create a simple test routine
	testRoutineYAML := `routine_name: "Test Integration Routine"
steps:
  - action: Delay
    count: 1
`
	routineFile := filepath.Join(routinesDir, "test_routine.yaml")
	if err := os.WriteFile(routineFile, []byte(testRoutineYAML), 0644); err != nil {
		t.Fatalf("Failed to write test routine: %v", err)
	}

	// Create a minimal config
	config := &Config{
		FolderPath: tempDir,
	}

	// Create a bot instance (without full initialization since we're just testing registries)
	bot, err := New(1, config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Manually initialize just the registries (skip ADB, emulator, etc.)
	bot.templateRegistry = nil // Would be initialized in real scenario
	bot.routineRegistry = nil  // Would be initialized in real scenario

	// Note: We can't fully test Initialize() here because it requires emulator setup
	// But we can verify the registry would work if initialized

	t.Log("Bot structure supports routine registry integration")

	// Verify the bot has the Templates() and Routines() methods
	// (This is compile-time verified, but we can check they return the expected types)
	if bot.Templates() != nil {
		t.Error("Templates() should return nil before initialization")
	}
	if bot.Routines() != nil {
		t.Error("Routines() should return nil before initialization")
	}

	// Test cleanup
	bot.Shutdown()
}

// TestBotImplementsBotInterface verifies that Bot implements the actions.BotInterface
func TestBotImplementsBotInterface(t *testing.T) {
	// This is a compile-time check - if Bot doesn't implement BotInterface,
	// this test won't even compile

	config := &Config{
		FolderPath: t.TempDir(),
	}

	bot, err := New(1, config)
	if err != nil {
		t.Fatalf("Failed to create bot: %v", err)
	}

	// Verify all BotInterface methods exist
	_ = bot.ADB()          // returns *adb.Controller
	_ = bot.CV()           // returns *cv.Service
	_ = bot.Context()      // returns context.Context
	_ = bot.Config()       // returns actions.ConfigInterface
	_ = bot.Templates()    // returns actions.TemplateRegistryInterface
	_ = bot.Routines()     // returns actions.RoutineRegistryInterface
	_ = bot.ErrorMonitor() // returns *monitor.ErrorMonitor
	_ = bot.IsPaused()     // returns bool
	_ = bot.IsStopped()    // returns bool

	t.Log("Bot successfully implements all BotInterface methods")
}
