# Emulator Instances Tab

The Emulator Instances tab provides a hierarchical view of orchestration groups and their running instances, matching the `emulator_instances.txt` mockup.

## Overview

This tab replaces the old Dashboard tab with a more structured view that shows:

1. **Active Groups** - Orchestration groups with running instances
2. **Idle Instances** - Instances that are available but not currently running
3. **Inactive Instances** - Instances not in any group

## Architecture

### Components

#### EmulatorInstanceCardV2
[emulator_instance_card.go](../components/emulator_instance_card.go)

Represents a single emulator instance card with three states:

- **Active**: Shows account info, routine status, and buttons: Pause, Stop, Abort, Shutdown
- **Idle**: Shows associated groups chips and buttons: Quick Start, Shutdown
- **Inactive**: Shows associated groups chips and buttons: Quick Start, Launch

Key features:
- Uses `canvas.Text` for compact display (14px)
- Conditional button display based on state
- Dynamic chip lists for associated groups with "and N more..." truncation
- Indented 20px to appear under parent group

#### GroupSectionCardV2
[group_section_card.go](../components/group_section_card.go)

Represents an orchestration group section with:
- Group name and orchestration ID
- Description (routine name)
- Started time and account pool progress
- "+ Instance" button
- Container for child instance cards (indented below)

### Tab Structure

The `EmulatorInstancesTab` ([emulator_instances.go](emulator_instances.go)) manages:

```
Emulator Instances
├── Controls (Refresh All button, status label)
├── Active Groups Section
│   ├── Group 1 Card
│   │   ├── Instance 1 Card (indented)
│   │   └── Instance 2 Card (indented)
│   └── Group 2 Card
│       └── Instance 3 Card (indented)
├── Idle Instances Section
│   └── Idle Parent Card
│       ├── Instance 4 Card (indented)
│       └── Instance 5 Card (indented)
└── Inactive Instances Section
    └── Inactive Parent Card
        ├── Instance 6 Card (indented)
        └── Instance 7 Card (indented)
```

## Integration

### Controller Changes

The tab is initialized in `controller.go` after the orchestrator:

```go
// Initialize emulator instances tab
c.emulatorInstancesTab = tabs.NewEmulatorInstancesTab(c.orchestrator, c.window)
```

It appears as the first tab, replacing the old Dashboard.

## Data Flow

1. **Discover Instances** - Uses `orchestrator.GetEmulatorManager()` to discover running MuMu windows
2. **Load Configured Instances** - Uses shared `MuMuManager` to read all instance configs from data folder
3. **Load Active Groups** - Loads active orchestration groups from orchestrator
4. **Get Instance Assignments** - Queries `orchestrator.GetAllInstanceAssignments()` to find running bots
5. **Get Group Assignments** - Queries `orchestrator.GetAllInstanceIDsFromGroups()` to find which instances belong to groups
6. **Categorize Instances**:
   - **Active**: Instances with running bots (shown under group sections)
   - **Idle**: Detected windows (running) but not running bots (may or may not be in groups)
   - **Inactive**: Configured instances without detected windows (not running)
7. **Create Cards** - For each category, creates appropriate cards with correct state and callbacks
8. **Periodic Refresh** - Every 1 second, calls `UpdateFromState()` on all cards

## Button Actions

### Active Instance Actions (TODO)
- **Pause**: Pause the bot routine (keep instance running)
- **Stop**: Stop the bot routine gracefully
- **Abort**: Force stop the routine immediately
- **Shutdown**: Stop and remove from group

### Idle/Inactive Instance Actions
- **Quick Start**: Start the instance with default settings (TODO)
- **Launch**: ✅ Launches the emulator using `emulatorManager.LaunchInstance()`, refreshes view after 2 seconds
- **Shutdown**: Power off the emulator (TODO)

### Group Actions
- **+ Instance**: Add another instance to the group

## Orchestrator API Used

The tab uses the following orchestrator methods to populate instances:

```go
// Get emulator manager for discovering instances
func (o *Orchestrator) GetEmulatorManager() *emulator.Manager

// Get instance assignments (which instances are running)
func (o *Orchestrator) GetAllInstanceAssignments() map[int]*InstanceAssignment

// Get all instance IDs from groups (which instances belong to groups)
func (o *Orchestrator) GetAllInstanceIDsFromGroups() map[int][]string

// Get active groups
func (o *Orchestrator) ListActiveGroups() []*BotGroup
```

These methods were added to support the instance categorization logic.

## Component Patterns Used

Follows the component library patterns:

- `Heading()` - Page title
- `SectionHeader()` - Section titles
- `Card()` / `CardWithIndent()` - Card containers
- `Subheading()` - Group/instance names
- `Caption()` - Orchestration IDs, small text
- `BoldText()` - Field labels
- `InlineLabels()` - Name + ID pattern
- `LabelButtonsRow()` - Header with right-aligned buttons
- `InlineInfoRow()` - Info fields
- `NavigationChip()` - Associated group chips
- `PrimaryButton()`, `SecondaryButton()`, `DangerButton()` - Action buttons
- `canvas.Text` - Compact text display (14px)

## Visual Layout

Matches mockup pattern:

```
┌─────────────────────────────────────────────────────────┐
│ Emulator Instances                                      │
├─────────────────────────────────────────────────────────┤
│ [ Refresh All ]                    2 group(s), 4 inst   │
├─────────────────────────────────────────────────────────┤
│ Active Groups                                           │
│   ┌─────────────────────────────────────────────────┐   │
│   │ Premium Farmers <abc123>          [ + Instance ]│   │
│   │ Running routine: farm_premium.yaml              │   │
│   │ Started: 2h ago   Account Pool: Premium (5/10)  │   │
│   └─────────────────────────────────────────────────┘   │
│      ┌─────────────────────────────────────────────┐    │
│      │ Instance 1 - Index 1   [Pause][Stop][Abort] │    │
│      │ Account premium_001 since 2h                │    │
│      │ Status: Running                             │    │
│      └─────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────┤
│ Idle Instances                                          │
│   ┌─────────────────────────────────────────────────┐   │
│   │ Idle                                            │   │
│   └─────────────────────────────────────────────────┘   │
│      ┌─────────────────────────────────────────────┐    │
│      │ Instance 5 - Index 5   [Quick Start][Shutd] │    │
│      │ No account injected                         │    │
│      │ Associated: <Premium> <Event> and 2 more... │    │
│      └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

## Migration from Dashboard

The old `DashboardTab` ([dashboard.go](../../dashboard.go)) has been replaced. Key differences:

### Old Dashboard
- Showed MuMu instances grid
- Showed running bots grid
- Manual refresh button
- Mixed instance discovery and bot management

### New Emulator Instances Tab
- Hierarchical orchestration groups → instances
- Organized by state (active/idle/inactive)
- Automatic refresh (1 second)
- Focused on orchestration management
- Uses component library for consistency

## See Also

- [OrchestrationCardV2](../components/orchestration_card_v2.go) - Similar pattern for orchestration groups tab
- [Mockup Implementation Guide](../components/MOCKUP_PATTERNS.md)
- [Component Library](../components/README.md)
