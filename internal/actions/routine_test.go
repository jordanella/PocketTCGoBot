package actions

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

// MockTemplateRegistry is a simple implementation for testing
type MockTemplateRegistry struct {
	templates map[string]cv.Template
}

func NewMockTemplateRegistry() *MockTemplateRegistry {
	return &MockTemplateRegistry{
		templates: make(map[string]cv.Template),
	}
}

func (m *MockTemplateRegistry) Add(name string, threshold float64) {
	m.templates[name] = cv.Template{
		Name:      name,
		Threshold: threshold,
	}
}

func (m *MockTemplateRegistry) Get(name string) (cv.Template, bool) {
	tmpl, ok := m.templates[name]
	return tmpl, ok
}

func (m *MockTemplateRegistry) MustGet(name string) cv.Template {
	tmpl, ok := m.templates[name]
	if !ok {
		panic("template not found: " + name)
	}
	return tmpl
}

func (m *MockTemplateRegistry) Has(name string) bool {
	_, ok := m.templates[name]
	return ok
}

// TestRoutineUnmarshal tests the basic YAML unmarshaling without validation
func TestRoutineUnmarshal(t *testing.T) {
	yamlContent := `routine_name: "Test Routine"
steps:
  - action: Click
    x: 100
    y: 200
  - action: Click
    x: 300
    y: 400
`

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_routine.yaml")

	err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Use the routine loader without template registry for basic unmarshaling test
	loader := NewRoutineLoader()
	actionBuilder, err := loader.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load routine: %v", err)
	}

	if actionBuilder == nil {
		t.Fatal("ActionBuilder is nil")
	}

	// Should have 2 steps built
	if len(actionBuilder.steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(actionBuilder.steps))
	}

	// Verify steps have the correct names
	if actionBuilder.steps[0].name != "Click" {
		t.Errorf("Expected first step name 'Click', got '%s'", actionBuilder.steps[0].name)
	}

	if actionBuilder.steps[1].name != "Click" {
		t.Errorf("Expected second step name 'Click', got '%s'", actionBuilder.steps[1].name)
	}
}

// TestRoutineLoaderWithExampleFile tests loading the example routine file
func TestRoutineLoaderWithExampleFile(t *testing.T) {
	// Setup mock template registry
	registry := NewMockTemplateRegistry()
	registry.Add("OK", 0.9)
	registry.Add("Main", 0.85)
	registry.Add("Confirm", 0.88)

	// Create routine loader with template registry
	loader := NewRoutineLoader().WithTemplateRegistry(registry)

	// Use the example file from docs
	examplePath := filepath.Join("..", "..", "docs", "example_routine.yaml")

	// Check if file exists, if not skip test
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Skip("Example routine file not found at", examplePath)
	}

	// Load and build the routine
	actionBuilder, err := loader.LoadFromFile(examplePath)
	if err != nil {
		t.Fatalf("Failed to load routine from file: %v", err)
	}

	if actionBuilder == nil {
		t.Fatal("ActionBuilder is nil")
	}

	// Verify that steps were built
	if len(actionBuilder.steps) == 0 {
		t.Error("Expected steps to be built, got 0 steps")
	}

	// The example file should have built multiple steps
	// First step: Click at (100, 200)
	// Second step: WhileImageFound loop (which becomes a single executable step)
	// Third step: Click at (150, 250)
	// Fourth step: Another WhileImageFound loop
	if len(actionBuilder.steps) < 4 {
		t.Errorf("Expected at least 4 built steps from example routine, got %d", len(actionBuilder.steps))
	}

	t.Logf("Successfully built %d steps from example routine", len(actionBuilder.steps))
}

// TestRoutineValidationErrors tests that validation errors are caught
func TestRoutineValidationErrors(t *testing.T) {
	testCases := []struct {
		name        string
		yamlContent string
		expectError string
	}{
		{
			name: "negative coordinates",
			yamlContent: `
routine_name: "Invalid Click"
steps:
  - action: Click
    x: -10
    y: 50
`,
			expectError: "coordinates",
		},
		{
			name: "unknown action type",
			yamlContent: `
routine_name: "Unknown Action"
steps:
  - action: NonExistentAction
    foo: bar
`,
			expectError: "unknown action type",
		},
		{
			name: "missing template in loop",
			yamlContent: `
routine_name: "Missing Template"
steps:
  - action: WhileImageFound
    max_attempts: 5
    actions:
      - action: Click
        x: 100
        y: 100
`,
			expectError: "template is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, "test.yaml")

			err := os.WriteFile(tempFile, []byte(tc.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			loader := NewRoutineLoader()
			_, err = loader.LoadFromFile(tempFile)

			if err == nil {
				t.Errorf("Expected error containing '%s', got no error", tc.expectError)
			} else if !contains(err.Error(), tc.expectError) {
				t.Errorf("Expected error containing '%s', got: %v", tc.expectError, err)
			}
		})
	}
}

// TestNestedActionStructure tests that nested actions are properly parsed
func TestNestedActionStructure(t *testing.T) {
	yamlContent := `
routine_name: "Nested Actions"
steps:
  - action: WhileImageFound
    max_attempts: 3
    template: "TestTemplate"
    actions:
      - action: Click
        x: 100
        y: 100
      - action: Click
        x: 200
        y: 200
`

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "nested.yaml")

	err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Setup template registry
	registry := NewMockTemplateRegistry()
	registry.Add("TestTemplate", 0.9)

	loader := NewRoutineLoader().WithTemplateRegistry(registry)
	actionBuilder, err := loader.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load routine: %v", err)
	}

	// Should have 1 step (the WhileImageFound loop)
	if len(actionBuilder.steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(actionBuilder.steps))
	}

	// Verify the step name indicates it's a WhileImageFound action
	if !contains(actionBuilder.steps[0].name, "WhileImageFound") {
		t.Errorf("Expected step name to contain 'WhileImageFound', got '%s'", actionBuilder.steps[0].name)
	}
}

// TestRoutineWithThresholdOverride tests action-level threshold override
func TestRoutineWithThresholdOverride(t *testing.T) {
	yamlContent := `
routine_name: "Threshold Override"
steps:
  - action: WhileImageFound
    max_attempts: 5
    template: "TestTemplate"
    threshold: 0.95
    actions:
      - action: Click
        x: 100
        y: 100
`

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "threshold.yaml")

	err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Setup template registry with default threshold
	registry := NewMockTemplateRegistry()
	registry.Add("TestTemplate", 0.8) // Default threshold

	loader := NewRoutineLoader().WithTemplateRegistry(registry)
	actionBuilder, err := loader.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load routine: %v", err)
	}

	if len(actionBuilder.steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(actionBuilder.steps))
	}

	// The threshold override is applied at execution time, so we can't test it here
	// but we can verify the routine loads successfully with threshold specified
	t.Log("Successfully loaded routine with threshold override")
}

// TestRoutineWithRegionOverride tests action-level region override
func TestRoutineWithRegionOverride(t *testing.T) {
	yamlContent := `
routine_name: "Region Override"
steps:
  - action: WhileImageFound
    max_attempts: 5
    template: "TestTemplate"
    region:
      x1: 100
      y1: 200
      x2: 500
      y2: 600
    actions:
      - action: Click
        x: 300
        y: 400
`

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "region.yaml")

	err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	registry := NewMockTemplateRegistry()
	registry.Add("TestTemplate", 0.9)

	loader := NewRoutineLoader().WithTemplateRegistry(registry)
	actionBuilder, err := loader.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load routine: %v", err)
	}

	if len(actionBuilder.steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(actionBuilder.steps))
	}

	t.Log("Successfully loaded routine with region override")
}

// TestMultipleActionTypes tests that different action types can be mixed
func TestMultipleActionTypes(t *testing.T) {
	yamlContent := `routine_name: "Mixed Actions"
steps:
  - action: Click
    x: 100
    y: 100
  - action: Delay
    count: 1
  - action: Click
    x: 200
    y: 200
`

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "mixed.yaml")

	err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewRoutineLoader()
	actionBuilder, err := loader.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load routine: %v", err)
	}

	// Should have 3 steps
	if len(actionBuilder.steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(actionBuilder.steps))
	}

	t.Log("Successfully loaded routine with mixed action types")
}

// TestDeeplyNestedActions tests routines with multiple levels of nesting
func TestDeeplyNestedActions(t *testing.T) {
	yamlContent := `
routine_name: "Deeply Nested"
steps:
  - action: WhileImageFound
    max_attempts: 5
    template: "Outer"
    actions:
      - action: Click
        x: 100
        y: 100
      - action: WhileImageFound
        max_attempts: 3
        template: "Inner"
        actions:
          - action: Click
            x: 200
            y: 200
          - action: Click
            x: 300
            y: 300
`

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "nested.yaml")

	err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	registry := NewMockTemplateRegistry()
	registry.Add("Outer", 0.9)
	registry.Add("Inner", 0.88)

	loader := NewRoutineLoader().WithTemplateRegistry(registry)
	actionBuilder, err := loader.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load deeply nested routine: %v", err)
	}

	// Should have 1 top-level step (the outer WhileImageFound)
	if len(actionBuilder.steps) != 1 {
		t.Errorf("Expected 1 top-level step, got %d", len(actionBuilder.steps))
	}

	t.Log("Successfully loaded deeply nested routine")
}

// TestTemplateNotFoundError tests that missing templates are caught during validation
func TestTemplateNotFoundError(t *testing.T) {
	yamlContent := `
routine_name: "Missing Template"
steps:
  - action: WhileImageFound
    max_attempts: 5
    template: "NonExistentTemplate"
    actions:
      - action: Click
        x: 100
        y: 100
`

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "missing_template.yaml")

	err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Registry without the required template
	registry := NewMockTemplateRegistry()
	registry.Add("SomeOtherTemplate", 0.9)

	loader := NewRoutineLoader().WithTemplateRegistry(registry)
	_, err = loader.LoadFromFile(tempFile)

	if err == nil {
		t.Error("Expected error for missing template, got nil")
	} else if !contains(err.Error(), "not found in registry") {
		t.Errorf("Expected error about template not found, got: %v", err)
	}
}


// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MockBotInterface for execution testing (if needed in future)
type MockBot struct {
	ctx       context.Context
	templates TemplateRegistryInterface
}

func NewMockBot(registry TemplateRegistryInterface) *MockBot {
	return &MockBot{
		ctx:       context.Background(),
		templates: registry,
	}
}

func (m *MockBot) Context() context.Context {
	return m.ctx
}

func (m *MockBot) Templates() TemplateRegistryInterface {
	return m.templates
}

func (m *MockBot) IsPaused() bool {
	return false
}

func (m *MockBot) IsStopped() bool {
	return false
}

// Stub implementations for other required interface methods
func (m *MockBot) ADB() interface{} {
	return nil
}

func (m *MockBot) CV() interface{} {
	return nil
}

func (m *MockBot) ErrorMonitor() interface{} {
	return nil
}

func (m *MockBot) Config() interface{} {
	return nil
}
