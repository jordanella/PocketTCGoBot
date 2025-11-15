# YAML Persistence Implementation Summary

## Overview

Implemented automatic YAML persistence for orchestration group definitions, allowing groups to be saved to disk and loaded on application startup.

## Changes Made

### 1. BotGroupDefinition YAML Methods (`internal/bot/orchestrator_definition.go`)

#### Added Imports
```go
import (
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)
```

#### Added YAML Tags
Updated all struct fields to include `yaml` tags for proper serialization:
```go
type BotGroupDefinition struct {
    Name        string `yaml:"name" json:"name"`
    Description string `yaml:"description,omitempty" json:"description"`
    // ... etc
}
```

#### New Methods

**`SaveToYAML(dirPath string) error`**
- Validates the definition
- Creates directory if it doesn't exist
- Sanitizes group name for filename
- Marshals to YAML and writes to disk
- Returns error on failure

**`LoadFromYAML(filePath string) (*BotGroupDefinition, error)`**
- Reads YAML file from disk
- Unmarshals into BotGroupDefinition
- Validates the loaded definition
- Returns error on failure

**`LoadAllFromYAML(dirPath string) ([]*BotGroupDefinition, error)`**
- Reads all .yaml/.yml files from directory
- Loads each definition
- Logs warnings for invalid files but continues
- Returns slice of valid definitions

**`DeleteYAML(dirPath string) error`**
- Deletes YAML file for this definition
- Ignores "file not found" errors
- Returns error on failure

**`sanitizeFilename(name string) string`**
- Converts group names to safe filenames
- Replaces spaces with underscores
- Removes special characters
- Defaults to "unnamed" if empty

### 2. Orchestrator Integration (`internal/bot/orchestrator.go`)

#### Added Field
```go
type Orchestrator struct {
    // ...
    groupConfigDir string  // Configuration directory for saving group definitions
}
```

#### Updated Constructor
```go
func NewOrchestrator(...) *Orchestrator {
    // Default groups config directory
    groupConfigDir := "data/groups"
    if config != nil && config.FolderPath != "" {
        groupConfigDir = config.FolderPath + "/groups"
    }

    return &Orchestrator{
        // ...
        groupConfigDir: groupConfigDir,
    }
}
```

#### Updated Methods

**`SaveGroupDefinition(def *BotGroupDefinition) error`**
- Now saves to both memory AND disk
- Calls `def.SaveToYAML(o.groupConfigDir)`
- Logs success message

**`DeleteGroupDefinition(name string) error`**
- Now deletes from both memory AND disk
- Calls `def.DeleteYAML(o.groupConfigDir)`
- Logs warning if file deletion fails

**New: `LoadGroupDefinitionsFromDisk() error`**
- Calls `LoadAllFromYAML(o.groupConfigDir)`
- Loads all definitions into memory
- Logs each loaded definition
- Returns error if directory read fails

### 3. Controller Integration (`internal/gui/controller.go`)

#### Added Startup Loading
In `initializeDatabase()` method, after creating the orchestrator:
```go
// Load saved group definitions from disk
if err := c.orchestrator.LoadGroupDefinitionsFromDisk(); err != nil {
    c.logTab.AddLog(LogLevelWarn, 0, fmt.Sprintf("Failed to load group definitions: %v", err))
}
```

This ensures all saved group definitions are loaded when the application starts.

### 4. Additional Struct Tags

#### `LaunchOptions` (`internal/bot/orchestrator.go`)
Added yaml tags to all fields:
```go
type LaunchOptions struct {
    ValidateRoutine   bool `yaml:"validate_routine" json:"validate_routine"`
    ValidateTemplates bool `yaml:"validate_templates" json:"validate_templates"`
    // ... etc
}
```

#### `RestartPolicy` (`internal/bot/config.go`)
Added yaml tags to all fields:
```go
type RestartPolicy struct {
    Enabled        bool          `yaml:"enabled" json:"enabled"`
    MaxRetries     int           `yaml:"max_retries" json:"max_retries"`
    // ... etc
}
```

## Flow Diagram

### Creating a Group (GUI)
```
User fills form in OrchestrationTab
    ↓
createGroup() validates input
    ↓
Creates BotGroupDefinition
    ↓
orchestrator.SaveGroupDefinition(def)
    ↓
def.SaveToYAML(groupConfigDir)
    ↓
YAML file created: bin/groups/Group_Name.yaml
    ↓
orchestrator.CreateGroupFromDefinition(def)
    ↓
Group appears in UI
```

### Loading Groups (Startup)
```
Controller.initializeDatabase()
    ↓
orchestrator = NewOrchestrator(...)
    ↓
orchestrator.LoadGroupDefinitionsFromDisk()
    ↓
LoadAllFromYAML("bin/groups")
    ↓
For each .yaml file:
    ↓
    LoadFromYAML(filePath)
    ↓
    Validate definition
    ↓
    Add to memory
    ↓
Definitions available for use
```

### Deleting a Group (GUI)
```
User clicks "Shutdown" in OrchestrationTab
    ↓
handleShutdown() confirms with user
    ↓
orchestrator.StopGroup(name) if running
    ↓
orchestrator.DeleteGroup(name)
    ↓
orchestrator.DeleteGroupDefinition(name)
    ↓
def.DeleteYAML(groupConfigDir)
    ↓
YAML file deleted from disk
    ↓
Group removed from UI
```

## File Structure

```
bin/
├── groups/                          # Auto-created directory
│   ├── Premium_Farmers.yaml        # Example group definition
│   ├── Event_Runners.yaml          # Example group definition
│   └── Daily_Routine.yaml          # Example group definition
└── bot-gui.exe
```

## YAML Format

See `docs/EXAMPLE_GROUP_CONFIG.yaml` for a complete example.

Basic structure:
```yaml
name: Group Name
description: Optional description
routine_name: routine.yaml
routine_config:
  key: "value"
available_instances: [1, 2, 3, 4]
requested_bot_count: 2
account_pool_name: PoolName
launch_options:
  validate_routine: true
  validate_templates: true
  validate_emulators: false
  on_conflict: 2
  stagger_delay: 5s
  emulator_timeout: 30s
  restart_policy:
    enabled: true
    max_retries: 5
    initial_delay: 10s
    max_delay: 5m0s
    backoff_factor: 2
    reset_on_success: true
restart_policy:
  enabled: true
  max_retries: 5
  initial_delay: 10s
  max_delay: 5m0s
  backoff_factor: 2
  reset_on_success: true
created_at: 2025-01-14T10:30:00Z
updated_at: 2025-01-14T10:30:00Z
tags: []
```

## Testing

### Manual Test Steps

1. **Create a Group**
   - Launch the application
   - Go to Orchestration tab
   - Click "Create New Group"
   - Fill in:
     - Name: "Test Group"
     - Routine: "test.yaml"
     - Instances: "1,2,3"
     - Bot Count: "2"
   - Click Create
   - Verify: `bin/groups/Test_Group.yaml` exists

2. **Verify YAML Content**
   - Open `bin/groups/Test_Group.yaml`
   - Verify all fields are present and correct

3. **Restart Application**
   - Close and restart the application
   - Check logs for: "Loaded group definition 'Test Group' from disk"
   - Group definition should be available (though not running)

4. **Delete Group**
   - Go to Orchestration tab
   - Click "Shutdown" on the test group
   - Confirm deletion
   - Verify: `bin/groups/Test_Group.yaml` is deleted

### Expected Behavior

✅ Creating a group saves YAML to disk
✅ YAML file contains all configuration
✅ Restarting app loads definitions from disk
✅ Definitions are validated on load
✅ Invalid files are skipped with warnings
✅ Deleting a group removes YAML file
✅ Groups directory is auto-created if missing

## Error Handling

- **Directory Creation**: Auto-creates if missing
- **Invalid YAML**: Logs warning, skips file, continues
- **File Permissions**: Returns error with details
- **Validation Errors**: Prevents saving/loading invalid definitions
- **Missing Files**: Ignores on delete, returns empty list on load

## Benefits

1. **Persistence**: Groups survive app restarts
2. **Version Control**: YAML files can be committed
3. **Sharing**: Easy to share configurations
4. **Backup**: Simple directory copy
5. **Manual Editing**: Can edit YAML directly
6. **Portability**: Move configs between environments

## Dependencies

- `gopkg.in/yaml.v3` - Already in go.mod

## Documentation

- `ORCHESTRATION_PERSISTENCE.md` - User guide
- `docs/EXAMPLE_GROUP_CONFIG.yaml` - Example configuration
- This file - Implementation details

## Future Enhancements

Potential improvements:
- UI to manage saved definitions (list, edit, duplicate)
- Import/export functionality
- Template groups (predefined configurations)
- Group versioning/history
- Validation UI for manual edits
