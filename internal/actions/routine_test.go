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


// TestRunRoutineAction tests the RunRoutine action that invokes another routine
func TestRunRoutineAction(t *testing.T) {
	// Create a simple sub-routine to be called
	subRoutineYAML := `routine_name: "Sub Routine"
steps:
  - action: Click
    x: 500
    y: 500
  - action: Delay
    count: 1
`

	// Create a main routine that calls the sub-routine
	mainRoutineYAML := `routine_name: "Main Routine"
steps:
  - action: Click
    x: 100
    y: 100
  - action: RunRoutine
    routine: sub_routine
  - action: Click
    x: 200
    y: 200
`

	tempDir := t.TempDir()

	// Create routines subdirectory
	routinesDir := filepath.Join(tempDir, "routines")
	err := os.Mkdir(routinesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create routines directory: %v", err)
	}

	// Write the sub-routine file
	subRoutineFile := filepath.Join(routinesDir, "sub_routine.yaml")
	err = os.WriteFile(subRoutineFile, []byte(subRoutineYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write sub-routine file: %v", err)
	}

	// Write the main routine file
	mainRoutineFile := filepath.Join(tempDir, "main_routine.yaml")
	err = os.WriteFile(mainRoutineFile, []byte(mainRoutineYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write main routine file: %v", err)
	}

	// Load the main routine
	loader := NewRoutineLoader()
	actionBuilder, err := loader.LoadFromFile(mainRoutineFile)
	if err != nil {
		t.Fatalf("Failed to load main routine: %v", err)
	}

	// Should have 3 steps: Click, RunRoutine, Click
	if len(actionBuilder.steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(actionBuilder.steps))
	}

	// Verify the middle step is a RunRoutine action
	if !contains(actionBuilder.steps[1].name, "RunRoutine") {
		t.Errorf("Expected second step to be RunRoutine, got '%s'", actionBuilder.steps[1].name)
	}

	t.Log("Successfully loaded routine with RunRoutine action")
}

// TestRunRoutineWithMissingFile tests error handling when routine file doesn't exist
func TestRunRoutineWithMissingFile(t *testing.T) {
	mainRoutineYAML := `routine_name: "Main Routine"
steps:
  - action: RunRoutine
    routine: non_existent_routine
`

	tempDir := t.TempDir()
	mainRoutineFile := filepath.Join(tempDir, "main_routine.yaml")
	err := os.WriteFile(mainRoutineFile, []byte(mainRoutineYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write main routine file: %v", err)
	}

	// Load the main routine (this should succeed - validation happens at runtime)
	loader := NewRoutineLoader()
	actionBuilder, err := loader.LoadFromFile(mainRoutineFile)
	if err != nil {
		t.Fatalf("Failed to load main routine: %v", err)
	}

	// The routine should load successfully - the error will occur at execution time
	if len(actionBuilder.steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(actionBuilder.steps))
	}

	t.Log("Successfully validated that RunRoutine loads without requiring file existence at build time")
}

// TestRoutineRegistry tests the routine registry functionality
func TestRoutineRegistry(t *testing.T) {
	tempDir := t.TempDir()
	routinesDir := filepath.Join(tempDir, "routines")
	err := os.Mkdir(routinesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create routines directory: %v", err)
	}

	// Create a test routine
	testRoutineYAML := `routine_name: "Test Routine"
steps:
  - action: Click
    x: 100
    y: 100
`
	testRoutineFile := filepath.Join(routinesDir, "test_routine.yaml")
	err = os.WriteFile(testRoutineFile, []byte(testRoutineYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test routine: %v", err)
	}

	// Create registry (will load routines when WithTemplateRegistry is called)
	registry := NewRoutineRegistry(routinesDir).WithTemplateRegistry(NewMockTemplateRegistry())

	t.Run("Has() checks if routine exists", func(t *testing.T) {
		if !registry.Has("test_routine") {
			t.Error("Expected Has() to return true for existing routine")
		}

		if registry.Has("non_existent") {
			t.Error("Expected Has() to return false for non-existent routine")
		}
	})

	t.Run("Get() returns pre-loaded routine", func(t *testing.T) {
		// Routine should already be loaded
		builder, err := registry.Get("test_routine")
		if err != nil {
			t.Fatalf("Failed to get routine: %v", err)
		}
		if builder == nil {
			t.Fatal("Expected non-nil builder")
		}

		// Multiple Get() calls should return same instance
		builder2, err := registry.Get("test_routine")
		if err != nil {
			t.Fatalf("Failed to get routine again: %v", err)
		}

		if builder != builder2 {
			t.Error("Expected Get() to return same instance")
		}
	})

	t.Run("GetValidationError() returns nil for valid routine", func(t *testing.T) {
		err := registry.GetValidationError("test_routine")
		if err != nil {
			t.Errorf("Expected no validation error, got: %v", err)
		}
	})

	t.Run("ListValid() returns valid routines", func(t *testing.T) {
		valid := registry.ListValid()
		if len(valid) != 1 {
			t.Errorf("Expected 1 valid routine, got %d", len(valid))
		}

		if len(valid) > 0 && valid[0] != "test_routine" {
			t.Errorf("Expected 'test_routine', got '%s'", valid[0])
		}
	})

	t.Run("ListInvalid() returns empty for all valid routines", func(t *testing.T) {
		invalid := registry.ListInvalid()
		if len(invalid) != 0 {
			t.Errorf("Expected 0 invalid routines, got %d: %v", len(invalid), invalid)
		}
	})

	t.Run("ListAvailable() returns discovered routines", func(t *testing.T) {
		// Create additional routine files
		routine2YAML := `routine_name: "Another Routine"
steps:
  - action: Click
    x: 200
    y: 200
`
		routine2File := filepath.Join(routinesDir, "another_routine.yaml")
		err := os.WriteFile(routine2File, []byte(routine2YAML), 0644)
		if err != nil {
			t.Fatalf("Failed to write second routine: %v", err)
		}

		// Create a .yml file too
		routine3YAML := `routine_name: "YML Routine"
steps:
  - action: Click
    x: 300
    y: 300
`
		routine3File := filepath.Join(routinesDir, "yml_routine.yml")
		err = os.WriteFile(routine3File, []byte(routine3YAML), 0644)
		if err != nil {
			t.Fatalf("Failed to write yml routine: %v", err)
		}

		// Create a new registry to pick up the new files
		registry2 := NewRoutineRegistry(routinesDir).WithTemplateRegistry(NewMockTemplateRegistry())
		available := registry2.ListAvailable()

		// Should have discovered all 3 routines
		if len(available) != 3 {
			t.Errorf("Expected 3 routines, got %d: %v", len(available), available)
		}

		// Check that all expected names are present
		expectedNames := map[string]bool{
			"test_routine":    false,
			"another_routine": false,
			"yml_routine":     false,
		}

		for _, name := range available {
			if _, ok := expectedNames[name]; ok {
				expectedNames[name] = true
			}
		}

		for name, found := range expectedNames {
			if !found {
				t.Errorf("Expected to find routine '%s' in ListAvailable()", name)
			}
		}
	})

	t.Run("Invalid routine is tracked with error", func(t *testing.T) {
		// Create an invalid routine
		invalidYAML := `routine_name: "Invalid Routine"
steps:
  - action: NonExistentAction
    foo: bar
`
		invalidFile := filepath.Join(routinesDir, "invalid.yaml")
		err := os.WriteFile(invalidFile, []byte(invalidYAML), 0644)
		if err != nil {
			t.Fatalf("Failed to write invalid routine: %v", err)
		}

		// Create a new registry to pick up all files
		registry3 := NewRoutineRegistry(routinesDir).WithTemplateRegistry(NewMockTemplateRegistry())

		// Should have the invalid routine tracked
		if !registry3.Has("invalid") {
			t.Error("Expected Has() to return true for invalid routine")
		}

		// Should have validation error
		validationErr := registry3.GetValidationError("invalid")
		if validationErr == nil {
			t.Error("Expected validation error for invalid routine")
		}

		// Get() should return the validation error
		_, err = registry3.Get("invalid")
		if err == nil {
			t.Error("Expected Get() to return error for invalid routine")
		}

		// Should appear in ListInvalid()
		invalid := registry3.ListInvalid()
		found := false
		for _, name := range invalid {
			if name == "invalid" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'invalid' in ListInvalid()")
		}

		// Should still appear in ListAvailable()
		available := registry3.ListAvailable()
		found = false
		for _, name := range available {
			if name == "invalid" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'invalid' in ListAvailable()")
		}
	})
}

// TestRoutineRegistryWithTemplates tests registry integration with template registry
func TestRoutineRegistryWithTemplates(t *testing.T) {
	tempDir := t.TempDir()
	routinesDir := filepath.Join(tempDir, "routines")
	err := os.Mkdir(routinesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create routines directory: %v", err)
	}

	// Create a routine that uses templates
	routineYAML := `routine_name: "Template Routine"
steps:
  - action: WhileImageFound
    template: "TestTemplate"
    max_attempts: 5
    actions:
      - action: Click
        x: 100
        y: 100
`
	routineFile := filepath.Join(routinesDir, "template_routine.yaml")
	err = os.WriteFile(routineFile, []byte(routineYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write routine: %v", err)
	}

	// Create template registry
	templateRegistry := NewMockTemplateRegistry()
	templateRegistry.Add("TestTemplate", 0.9)

	// Create routine registry with template registry
	registry := NewRoutineRegistry(routinesDir).WithTemplateRegistry(templateRegistry)

	// Should load successfully with valid template
	_, err = registry.Get("template_routine")
	if err != nil {
		t.Errorf("Failed to load routine with valid template: %v", err)
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
	routines  RoutineRegistryInterface
}

func NewMockBot(registry TemplateRegistryInterface) *MockBot {
	return &MockBot{
		ctx:       context.Background(),
		templates: registry,
	}
}

func NewMockBotWithRoutines(templates TemplateRegistryInterface, routines RoutineRegistryInterface) *MockBot {
	return &MockBot{
		ctx:       context.Background(),
		templates: templates,
		routines:  routines,
	}
}

func (m *MockBot) Context() context.Context {
	return m.ctx
}

func (m *MockBot) Templates() TemplateRegistryInterface {
	return m.templates
}

func (m *MockBot) Routines() RoutineRegistryInterface {
	return m.routines
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
