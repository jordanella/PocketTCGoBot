# Orchestration Group Persistence

## Overview

Orchestration groups are now automatically persisted to disk as YAML files. This allows groups to be saved, edited, and reloaded across application restarts.

## Storage Location

Group definitions are saved in:
```
<FolderPath>/groups/
```

Where `<FolderPath>` is configured in your bot config (typically `bin`).

For example: `bin/groups/Premium_Farmers.yaml`

## File Format

Each group is saved as a YAML file with the following structure:

```yaml
name: Premium Farmers
description: Farming premium packs routine
routine_name: farm_premium_packs.yaml
routine_config:
  max_runs: "100"
  delay_between_runs: "5"
available_instances:
  - 1
  - 2
  - 3
  - 4
requested_bot_count: 2
account_pool_name: Premium
launch_options:
  validate_routine: true
  validate_templates: true
  validate_emulators: false
  stagger_delay: 5s
  emulator_timeout: 30s
  on_conflict: 0
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

## Automatic Behavior

### Creating Groups
When you create a new orchestration group via the GUI:
1. The definition is saved to memory
2. A YAML file is automatically created in `<FolderPath>/groups/`
3. The filename is sanitized from the group name (spaces → underscores)

### Loading Groups
When the application starts:
1. All YAML files in `<FolderPath>/groups/` are loaded
2. Definitions are validated and stored in memory
3. Invalid definitions are logged as warnings and skipped

**Note**: Saved definitions are NOT automatically started. They must be manually started via the Orchestration tab.

### Deleting Groups
When you shutdown a group:
1. The group is stopped (if running)
2. The active group is removed from memory
3. The YAML file is deleted from disk
4. The definition is removed from memory

## Manual Management

### Editing Group Definitions
You can manually edit YAML files in the `groups/` directory:
1. Stop the group if it's running
2. Edit the YAML file
3. Restart the application to reload definitions
4. Start the group from the Orchestration tab

### Backing Up Groups
Simply copy the entire `groups/` directory to back up all definitions.

### Sharing Groups
Share individual YAML files or the entire `groups/` directory with team members.

## API Usage

### Saving a Definition
```go
definition := bot.NewBotGroupDefinition("My Group", "routine.yaml", []int{1,2,3}, 2)
definition.AccountPoolName = "Premium"

// Saves to both memory and disk
err := orchestrator.SaveGroupDefinition(definition)
```

### Loading Definitions
```go
// Load all definitions from disk (called automatically on startup)
err := orchestrator.LoadGroupDefinitionsFromDisk()

// Get a specific definition
def, err := orchestrator.LoadGroupDefinition("My Group")

// List all loaded definitions
definitions := orchestrator.ListGroupDefinitions()
```

### Deleting a Definition
```go
// Deletes from both memory and disk
err := orchestrator.DeleteGroupDefinition("My Group")
```

## File Naming

Group names are sanitized for filenames:
- Spaces → underscores
- Special characters → removed
- Only alphanumeric, hyphens, and underscores allowed
- Empty names → "unnamed"

Examples:
- "Premium Farmers" → `Premium_Farmers.yaml`
- "Test Group #1" → `Test_Group_1.yaml`
- "daily-routine" → `daily-routine.yaml`

## Benefits

1. **Persistence**: Groups survive application restarts
2. **Version Control**: YAML files can be committed to git
3. **Sharing**: Easy to share configurations with team members
4. **Backup**: Simple to backup by copying the directory
5. **Editing**: Can manually edit YAML for advanced configurations
6. **Portability**: Move configurations between environments

## Troubleshooting

### "Failed to load group definitions"
- Check that the `groups/` directory exists and is readable
- Verify YAML syntax is valid
- Check file permissions

### "Failed to save group definition"
- Ensure the application has write permissions
- Check disk space availability
- Verify the group name is valid

### Group doesn't appear after restart
- Check that the YAML file exists in the `groups/` directory
- Verify the YAML syntax is valid (use a YAML validator)
- Check the application logs for validation errors

## Implementation Details

### Code Locations

- **Definition struct**: `internal/bot/orchestrator_definition.go`
- **YAML methods**: `SaveToYAML()`, `LoadFromYAML()`, `LoadAllFromYAML()`, `DeleteYAML()`
- **Orchestrator integration**: `internal/bot/orchestrator.go`
- **GUI integration**: `internal/gui/tabs/orchestration.go`
- **Startup loading**: `internal/gui/controller.go`

### Dependencies

- `gopkg.in/yaml.v3` - YAML marshaling/unmarshaling
