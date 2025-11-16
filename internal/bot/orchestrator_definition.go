package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// BotGroupDefinition represents a saved orchestration group configuration.
// This is the persistent blueprint that can be saved, loaded, and edited.
// It defines what a group SHOULD do, while BotGroup represents what it IS doing.
type BotGroupDefinition struct {
	// Identity
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description"`

	// Routine configuration
	RoutineName   string            `yaml:"routine_name" json:"routine_name"`
	RoutineConfig map[string]string `yaml:"routine_config,omitempty" json:"routine_config,omitempty"` // Variable overrides

	// Emulator configuration
	AvailableInstances []int `yaml:"available_instances" json:"available_instances"`
	RequestedBotCount  int   `yaml:"requested_bot_count" json:"requested_bot_count"`

	// Account pool configuration
	AccountPoolName  string   `yaml:"account_pool_name,omitempty" json:"account_pool_name,omitempty"`     // Legacy single pool (deprecated)
	AccountPoolNames []string `yaml:"account_pool_names,omitempty" json:"account_pool_names,omitempty"` // Multiple pools

	// Launch options
	LaunchOptions LaunchOptions `yaml:"launch_options" json:"launch_options"`

	// Restart policy
	RestartPolicy RestartPolicy `yaml:"restart_policy" json:"restart_policy"`

	// Metadata
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	Tags      []string  `yaml:"tags,omitempty" json:"tags,omitempty"` // For categorization/filtering
}

// Clone creates a deep copy of the definition
func (d *BotGroupDefinition) Clone() *BotGroupDefinition {
	clone := *d

	// Deep copy slices and maps
	clone.AvailableInstances = append([]int{}, d.AvailableInstances...)
	clone.AccountPoolNames = append([]string{}, d.AccountPoolNames...)
	clone.Tags = append([]string{}, d.Tags...)

	clone.RoutineConfig = make(map[string]string)
	for k, v := range d.RoutineConfig {
		clone.RoutineConfig[k] = v
	}

	return &clone
}

// Validate checks if the definition is valid and returns an error if not
func (d *BotGroupDefinition) Validate() error {
	if d.Name == "" {
		return fmt.Errorf("name is required")
	}

	if d.RoutineName == "" {
		return fmt.Errorf("routine name is required")
	}

	if len(d.AvailableInstances) == 0 {
		return fmt.Errorf("at least one emulator instance is required")
	}

	if d.RequestedBotCount <= 0 {
		return fmt.Errorf("requested bot count must be positive")
	}

	if d.RequestedBotCount > len(d.AvailableInstances) {
		return fmt.Errorf("requested bot count (%d) exceeds available instances (%d)",
			d.RequestedBotCount, len(d.AvailableInstances))
	}

	// Validate that instance IDs are unique
	instanceSet := make(map[int]bool)
	for _, id := range d.AvailableInstances {
		if instanceSet[id] {
			return fmt.Errorf("duplicate instance ID: %d", id)
		}
		instanceSet[id] = true
	}

	return nil
}

// Update updates the definition with new values and sets UpdatedAt timestamp
func (d *BotGroupDefinition) Update(updates *BotGroupDefinition) error {
	if updates.Name != "" && updates.Name != d.Name {
		return fmt.Errorf("cannot change name of existing definition")
	}

	// Update fields
	if updates.Description != "" {
		d.Description = updates.Description
	}
	if updates.RoutineName != "" {
		d.RoutineName = updates.RoutineName
	}
	if len(updates.AvailableInstances) > 0 {
		d.AvailableInstances = updates.AvailableInstances
	}
	if updates.RequestedBotCount > 0 {
		d.RequestedBotCount = updates.RequestedBotCount
	}
	if updates.AccountPoolName != "" {
		d.AccountPoolName = updates.AccountPoolName
	}
	if len(updates.RoutineConfig) > 0 {
		d.RoutineConfig = updates.RoutineConfig
	}
	if len(updates.Tags) > 0 {
		d.Tags = updates.Tags
	}

	// Update launch options and restart policy
	d.LaunchOptions = updates.LaunchOptions
	d.RestartPolicy = updates.RestartPolicy

	// Set updated timestamp
	d.UpdatedAt = time.Now()

	// Validate after update
	return d.Validate()
}

// NewBotGroupDefinition creates a new definition with defaults
func NewBotGroupDefinition(name, routineName string, instances []int, botCount int) *BotGroupDefinition {
	now := time.Now()

	return &BotGroupDefinition{
		Name:               name,
		RoutineName:        routineName,
		AvailableInstances: instances,
		RequestedBotCount:  botCount,
		RoutineConfig:      make(map[string]string),
		Tags:               []string{},
		CreatedAt:          now,
		UpdatedAt:          now,
		LaunchOptions: LaunchOptions{
			ValidateRoutine:   true,
			ValidateTemplates: true,
			ValidateEmulators: false,
			StaggerDelay:      5 * time.Second,
			EmulatorTimeout:   30 * time.Second,
		},
		RestartPolicy: RestartPolicy{
			Enabled:        true,
			MaxRetries:     5,
			InitialDelay:   10 * time.Second,
			MaxDelay:       5 * time.Minute,
			BackoffFactor:  2.0,
			ResetOnSuccess: true,
		},
	}
}

// SaveToYAML saves the definition to a YAML file
func (d *BotGroupDefinition) SaveToYAML(dirPath string) error {
	if err := d.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid definition: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate filename from group name (sanitized)
	filename := sanitizeFilename(d.Name) + ".yaml"
	filePath := filepath.Join(dirPath, filename)

	// Marshal to YAML
	data, err := yaml.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed to marshal definition: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadFromYAML loads a definition from a YAML file
func LoadFromYAML(filePath string) (*BotGroupDefinition, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal YAML
	var def BotGroupDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Validate loaded definition
	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("loaded definition is invalid: %w", err)
	}

	return &def, nil
}

// LoadAllFromYAML loads all group definitions from a directory
func LoadAllFromYAML(dirPath string) ([]*BotGroupDefinition, error) {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Directory doesn't exist, return empty list
		return []*BotGroupDefinition{}, nil
	}

	// Read directory entries
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	definitions := make([]*BotGroupDefinition, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .yaml and .yml files
		name := entry.Name()
		if filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml" {
			continue
		}

		// Load definition
		filePath := filepath.Join(dirPath, name)
		def, err := LoadFromYAML(filePath)
		if err != nil {
			fmt.Printf("Warning: failed to load %s: %v\n", name, err)
			continue
		}

		definitions = append(definitions, def)
	}

	return definitions, nil
}

// DeleteYAML deletes the YAML file for this definition
func (d *BotGroupDefinition) DeleteYAML(dirPath string) error {
	filename := sanitizeFilename(d.Name) + ".yaml"
	filePath := filepath.Join(dirPath, filename)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, not an error
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// sanitizeFilename converts a group name to a safe filename
func sanitizeFilename(name string) string {
	// Replace spaces and special characters with underscores
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			result += string(r)
		} else if r == ' ' {
			result += "_"
		}
	}
	if result == "" {
		result = "unnamed"
	}
	return result
}
