package actions

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConditionUnmarshaling(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantType string
		wantErr  bool
	}{
		{
			name: "ImageExists condition",
			yaml: `
type: ImageExists
template: "OK"
`,
			wantType: "*actions.ImageExists",
			wantErr:  false,
		},
		{
			name: "ImageNotExists condition",
			yaml: `
type: ImageNotExists
template: "Error"
`,
			wantType: "*actions.ImageNotExists",
			wantErr:  false,
		},
		{
			name: "Not condition",
			yaml: `
type: Not
condition:
  type: ImageExists
  template: "Test"
`,
			wantType: "*actions.Not",
			wantErr:  false,
		},
		{
			name: "All condition",
			yaml: `
type: All
conditions:
  - type: ImageExists
    template: "A"
  - type: ImageExists
    template: "B"
`,
			wantType: "*actions.All",
			wantErr:  false,
		},
		{
			name: "Any condition",
			yaml: `
type: Any
conditions:
  - type: ImageExists
    template: "A"
  - type: ImageNotExists
    template: "B"
`,
			wantType: "*actions.Any",
			wantErr:  false,
		},
		{
			name: "None condition",
			yaml: `
type: None
conditions:
  - type: ImageExists
    template: "A"
  - type: ImageExists
    template: "B"
`,
			wantType: "*actions.None",
			wantErr:  false,
		},
		{
			name: "Nested conditions",
			yaml: `
type: Any
conditions:
  - type: All
    conditions:
      - type: ImageExists
        template: "A"
      - type: Not
        condition:
          type: ImageExists
          template: "B"
  - type: ImageExists
    template: "C"
`,
			wantType: "*actions.Any",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw map[string]interface{}
			if err := yaml.Unmarshal([]byte(tt.yaml), &raw); err != nil {
				t.Fatalf("Failed to unmarshal raw YAML: %v", err)
			}

			condition, err := unmarshalCondition(raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshalCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotType := typeString(condition)
				if gotType != tt.wantType {
					t.Errorf("unmarshalCondition() got type = %v, want %v", gotType, tt.wantType)
				}
			}
		})
	}
}

func TestIfActionUnmarshaling(t *testing.T) {
	yamlStr := `
action: If
condition:
  type: ImageExists
  template: "OK"
then:
  - action: Click
    x: 100
    y: 200
else:
  - action: Click
    x: 300
    y: 400
`

	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &raw); err != nil {
		t.Fatalf("Failed to unmarshal raw YAML: %v", err)
	}

	// Convert to If action through registry
	ifAction := &If{}
	yamlBytes, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if err := yaml.Unmarshal(yamlBytes, ifAction); err != nil {
		t.Fatalf("Failed to unmarshal If action: %v", err)
	}

	if ifAction.Condition == nil {
		t.Error("Condition should not be nil")
	}

	if len(ifAction.ThenActions) != 1 {
		t.Errorf("Expected 1 then action, got %d", len(ifAction.ThenActions))
	}

	if len(ifAction.ElseActions) != 1 {
		t.Errorf("Expected 1 else action, got %d", len(ifAction.ElseActions))
	}
}

func TestWhileActionUnmarshaling(t *testing.T) {
	yamlStr := `
action: While
max_attempts: 10
condition:
  type: All
  conditions:
    - type: ImageExists
      template: "A"
    - type: ImageExists
      template: "B"
actions:
  - action: Click
    x: 100
    y: 200
  - action: Delay
    count: 1
`

	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &raw); err != nil {
		t.Fatalf("Failed to unmarshal raw YAML: %v", err)
	}

	whileAction := &While{}
	yamlBytes, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if err := yaml.Unmarshal(yamlBytes, whileAction); err != nil {
		t.Fatalf("Failed to unmarshal While action: %v", err)
	}

	if whileAction.Condition == nil {
		t.Error("Condition should not be nil")
	}

	if whileAction.MaxAttempts != 10 {
		t.Errorf("Expected max_attempts = 10, got %d", whileAction.MaxAttempts)
	}

	if len(whileAction.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(whileAction.Actions))
	}
}

func TestRoutineWithConditions(t *testing.T) {
	yamlStr := `
routine_name: "Test Routine"
steps:
  - action: If
    condition:
      type: ImageExists
      template: "OK"
    then:
      - action: Click
        x: 100
        y: 200
  - action: While
    max_attempts: 5
    condition:
      type: Any
      conditions:
        - type: ImageExists
          template: "A"
        - type: ImageExists
          template: "B"
    actions:
      - action: Click
        x: 300
        y: 400
`

	var routine Routine
	if err := yaml.Unmarshal([]byte(yamlStr), &routine); err != nil {
		t.Fatalf("Failed to unmarshal routine: %v", err)
	}

	if routine.RoutineName != "Test Routine" {
		t.Errorf("Expected routine name 'Test Routine', got '%s'", routine.RoutineName)
	}

	if len(routine.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(routine.Steps))
	}

	// Check first step is If
	if _, ok := routine.Steps[0].(*If); !ok {
		t.Errorf("Expected first step to be *If, got %T", routine.Steps[0])
	}

	// Check second step is While
	if _, ok := routine.Steps[1].(*While); !ok {
		t.Errorf("Expected second step to be *While, got %T", routine.Steps[1])
	}
}

func TestIfWithElseIf(t *testing.T) {
	yamlStr := `
action: If
condition:
  type: ImageExists
  template: "A"
then:
  - action: Click
    x: 100
    y: 100
elseif:
  - condition:
      type: ImageExists
      template: "B"
    then:
      - action: Click
        x: 200
        y: 200
  - condition:
      type: ImageExists
      template: "C"
    then:
      - action: Click
        x: 300
        y: 300
else:
  - action: Click
    x: 400
    y: 400
`

	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &raw); err != nil {
		t.Fatalf("Failed to unmarshal raw YAML: %v", err)
	}

	ifAction := &If{}
	yamlBytes, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if err := yaml.Unmarshal(yamlBytes, ifAction); err != nil {
		t.Fatalf("Failed to unmarshal If action: %v", err)
	}

	if ifAction.Condition == nil {
		t.Error("Condition should not be nil")
	}

	if len(ifAction.ThenActions) != 1 {
		t.Errorf("Expected 1 then action, got %d", len(ifAction.ThenActions))
	}

	if len(ifAction.ElseIfs) != 2 {
		t.Errorf("Expected 2 else-if branches, got %d", len(ifAction.ElseIfs))
	} else {
		for i, elseIf := range ifAction.ElseIfs {
			if elseIf.Condition == nil {
				t.Errorf("ElseIf %d condition should not be nil", i)
			}
			if len(elseIf.Then) != 1 {
				t.Errorf("ElseIf %d: expected 1 then action, got %d", i, len(elseIf.Then))
			}
		}
	}

	if len(ifAction.ElseActions) != 1 {
		t.Errorf("Expected 1 else action, got %d", len(ifAction.ElseActions))
	}
}

func TestBreakAction(t *testing.T) {
	yamlStr := `
routine_name: "Test Break"
steps:
  - action: While
    max_attempts: 5
    condition:
      type: ImageExists
      template: "Test"
    actions:
      - action: Click
        x: 100
        y: 100
      - action: If
        condition:
          type: ImageExists
          template: "Stop"
        then:
          - action: Break
`

	var routine Routine
	if err := yaml.Unmarshal([]byte(yamlStr), &routine); err != nil {
		t.Fatalf("Failed to unmarshal routine: %v", err)
	}

	if len(routine.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(routine.Steps))
	}

	whileAction, ok := routine.Steps[0].(*While)
	if !ok {
		t.Fatalf("Expected first step to be *While, got %T", routine.Steps[0])
	}

	if len(whileAction.Actions) != 2 {
		t.Fatalf("Expected 2 actions in while loop, got %d", len(whileAction.Actions))
	}

	// Second action should be an If statement containing a Break
	ifAction, ok := whileAction.Actions[1].(*If)
	if !ok {
		t.Fatalf("Expected second action to be *If, got %T", whileAction.Actions[1])
	}

	if len(ifAction.ThenActions) != 1 {
		t.Fatalf("Expected 1 then action, got %d", len(ifAction.ThenActions))
	}

	_, ok = ifAction.ThenActions[0].(*Break)
	if !ok {
		t.Errorf("Expected then action to be *Break, got %T", ifAction.ThenActions[0])
	}
}

// Helper to get type string
func typeString(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", v)
}
