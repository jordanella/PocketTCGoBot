# Pokemon TCG Pocket Bot - UI Architecture Plan

## Executive Summary

This document outlines a complete redesign of the bot's user interface using Fyne, focusing on usability, real-time monitoring, and streamlined workflows. The design prioritizes the most common user actions while providing comprehensive access to all system components.

---

## Startup Experience

### Splash Screen / Initialization
**Purpose**: Provide visual feedback during application startup and system initialization

**Components**:
- Application logo/branding
- Progress bar with stage indicators
- Status messages for each initialization phase:
  - Database connection & migrations
  - Loading account pools from disk
  - Loading routine definitions
  - Loading bot group definitions
  - Loading template registry
  - Initializing emulator manager
  - Event bus initialization
  - Health monitoring startup
  - Checking for new card sets (future: fetch from API if available)

**Design Notes**:
- Non-blocking UI - show partial interface if some components fail
- Error reporting with actionable messages
- Estimated time remaining for long operations
- Skip button for development/testing (loads with minimal setup)

---

## Primary Navigation Structure

### Tab-Based Layout
Main application uses a tabbed interface with the following top-level tabs:

1. **Quick Launch** (Home/Dashboard)
2. **Live Monitor**
3. **Bot Groups**
4. **Account Pools**
5. **Routines**
6. **Accounts**
7. **Card Collection**
8. **History & Analytics**
9. **Settings**

Optional tabs (context-dependent):
- **Logs Viewer** (toggleable via settings)

---

## Tab 1: Quick Launch (Home)

**Purpose**: Instant access to frequently used operations and system overview

### Layout Sections

#### Pinned Groups (Top Priority)
- Grid of cards showing pinned orchestration setups
- Each card displays:
  - Group name & description
  - Routine name
  - Bot count / Instance count
  - Account pool name & available accounts
  - Last run timestamp
  - Quick stats (success rate, avg runtime)
  - **Large "LAUNCH" button** (primary action)
  - Pin/unpin toggle
  - Quick edit icon (opens launch overrides dialog)

#### System Status Overview
- Active Groups: X running, Y bots active
- Account Pools: Available/In-Use/Total accounts
- Emulator Health: X/Y instances ready
- Recent Errors/Warnings (if any)
- Database status indicator

#### Quick Actions Panel
- **Create New Group** button
- **Import Accounts** button
- **Test Routine** button
- **Emergency Stop All** button (red, requires confirmation)

#### Recent Activity Feed
- Last 10 launches with status (success/failed/stopped)
- Click to expand details
- Filter by group/status
- Clear history button

### Interaction Flows

**Launching a Pinned Group**:
1. User clicks "LAUNCH" on pinned card
2. Confirmation dialog appears with override options:
   - Bot count slider/input
   - Instance selector (multi-select)
   - Account pool dropdown
   - Max accounts limit
   - Advanced: routine config overrides
   - "Launch with Defaults" vs "Launch with Overrides"
3. Launch begins, card updates to show "RUNNING" state
4. User auto-navigated to Live Monitor tab

**Pinning a Group**:
- Groups page has star/pin icon
- Pinned groups appear on Quick Launch
- Drag-to-reorder pinned groups
- Maximum 12 pinned groups (configurable)

---

## Tab 2: Live Monitor

**Purpose**: Real-time monitoring of all active bot groups and instances

### Layout Sections

#### Active Groups Panel (Left Sidebar)
- List of running groups (expandable tree)
- Each group shows:
  - Name & orchestration ID
  - Start time & runtime
  - Bot count (active/total)
  - Progress indicator (if applicable)
  - Stop/Pause buttons
- Click group to select and show details in main panel

#### Selected Group Detail View (Main Area)

**Header Bar**:
- Group name & routine name
- Status badge (Running/Stopping/Paused)
- Control buttons: Stop Group, Restart Group, View Logs
- Account pool stats: Available/In-Use accounts

**Bot Instances Grid**:
- Cards for each bot instance showing:
  - Instance ID & emulator window indicator
  - Current status (Starting/Running/Stopping/Failed/Completed)
  - Current account (device account name)
  - Current routine step/action (real-time updates)
  - Runtime for current iteration
  - Total iterations completed
  - Health status indicators (window/ADB)
  - Mini screenshot/screen preview (optional)
  - Individual controls: Stop Bot, Restart Bot, View Details
  - **Instance-Specific Event Feed**: Shows recent events for this specific bot/instance

**System Event Stream** (Bottom Panel, Collapsible):
- Scrolling feed of system-level events from EventBus
- Shows orchestrator, health monitor, pool refresh events
- Instance/bot-specific events are shown on respective cards
- Filterable by: Event type, Severity
- Color-coded by event type
- Auto-scroll toggle
- Toggleable via developer tools setting

#### Emulator Instances Panel (Right Sidebar, Collapsible)

**Running Instances (In Groups)**:
- Shows instances currently assigned to bot groups
- Grouped by bot group
- Each shows: Instance ID, Group name, Health status

**Idle Instances (Available)**:
- Running emulator windows not part of any group
- Each shows: Instance ID, Window title, Health status (window/ADB), "Assign to Group" button

**Configured But Not Running**:
- Instances in MuMu configuration but not currently launched
- Each shows: Instance ID, "Launch Instance" button
- Health status: Offline

**Actions**:
- Refresh/Discover Instances button
- Launch Selected Instance
- Connect/Disconnect ADB
- View Instance Details

#### No Active Groups View
- Empty state illustration
- "No bot groups currently running"
- Quick action: "Go to Bot Groups" or "Go to Quick Launch"

### Real-Time Updates
- WebSocket/event bus integration for live updates
- No manual refresh required
- Visual indicators for state changes (animations)
- Toast notifications for critical events (bot failures, account exhaustion)

---

## Tab 3: Bot Groups

**Purpose**: Manage bot group definitions (templates/profiles)

### Layout

#### Groups List (Left Panel)
- Searchable/filterable list of all group definitions
- Sort by: Name, Last Modified, Created Date, Last Run
- Filter by: Tags, Routine, Account Pool
- Each list item shows:
  - Name & description
  - Tags (colored badges)
  - Routine name
  - Bot count & instances preview
  - Pin status
  - Last run timestamp
- Multi-select for batch operations

#### Selected Group Detail (Main Panel)

**View Mode**:
- Full configuration display (read-only)
- Validation status indicator
- Statistics from previous runs:
  - Total runs, success rate
  - Avg runtime, accounts processed
  - Last 5 run results
- Action buttons:
  - Launch (with overrides dialog)
  - Edit
  - Duplicate
  - Delete (with confirmation)
  - Show in Folder

**Edit Mode** (Form View):
- **Basic Info Section**:
  - Name (required)
  - Description
  - Tags (multi-select with create new)

- **Routine Configuration Section**:
  - Routine selector (dropdown with preview)
  - Routine config variables (dynamic form based on routine's config params)
  - Test routine button (validates without launching)

- **Emulator & Bot Section**:
  - Available instances (multi-select with visual instance picker)
  - Requested bot count (slider with validation)
  - Instance allocation preview (shows which instances will be used)

- **Account Pool Section**:
  - Pool selector (multi-select - can aggregate from multiple pools)
  - Pool stats preview (available accounts across selected pools)
  - Account limiting options (future)

- **Launch Options Section** (Collapsible):
  - Validate routine checkbox
  - Validate templates checkbox
  - Validate emulators checkbox
  - Conflict resolution dropdown
  - Stagger delay slider
  - Emulator timeout input

- **Restart Policy Section** (Collapsible):
  - Enable restart toggle
  - Max retries input
  - Initial delay input
  - Max delay input
  - Backoff factor slider
  - Reset on success checkbox

- **Action Buttons**:
  - Save, Save & Launch, Cancel
  - Validate button (shows validation results)

#### Toolbar Actions
- Create New Group button
- Import Groups (drag-and-drop YAML files or browse)
- Delete Selected (batch)
- Refresh from Disk
- Open Groups Folder

---

## Tab 4: Account Pools

**Purpose**: Manage account pool definitions and view pool statistics

### Layout

#### Pools List (Left Panel)
- All saved pool definitions
- Each shows:
  - Pool name
  - Total accounts (from last refresh)
  - Last refresh timestamp
- Active pools indicator (in use by running groups)

#### Selected Pool Detail (Main Panel)

**View Mode**:
- **Pool Information**:
  - Name & description
  - Creation/modification timestamps
  - Current statistics:
    - Total accounts: X
    - Available: Y
    - In Use: Z (by which groups)
  - Refresh button (manual pool refresh - also auto-refreshes on group launch and pool exhaustion)
  - Test Pool button (validates queries)

- **Configuration Display**:
  - Query sources (read-only view of filters/sorts)
  - Include list (specific accounts)
  - Exclude list (specific accounts)
  - Watched paths (file monitoring)

- **Sample Accounts Preview** (Table):
  - First 10 accounts with ID, pack count, status
  - "View All Accounts" button (opens Accounts tab filtered to this pool)

- **Action Buttons**:
  - Edit
  - Duplicate
  - Delete
  - Show in Folder
  - View Full Account List

**Edit Mode**:
- **Basic Info**:
  - Pool name (required)
  - Description

- **Query Sources** (Dynamic List):
  - Add Query button
  - Each query shows:
    - Query name
    - Filter builder (column, comparator, value)
      - Visual query builder with dropdowns
      - Autocomplete for column names
      - Appropriate comparators based on column type
    - Sort configuration
    - Move up/down, delete query

- **Include/Exclude Lists**:
  - Text area for device account IDs (one per line)
  - Or: Multi-select from all accounts
  - Conflict detection (warn if same account in both lists)

- **Watched Paths**:
  - Add directory picker
  - List of watched paths with remove button
  - Auto-import toggle

- **Action Buttons**:
  - Save, Test & Save (validates queries first), Cancel
  - Test Queries button (shows preview results)

#### Toolbar Actions
- Create New Pool
- Import Pools (drag-and-drop YAML files or browse)
- Refresh All Pools
- Delete Selected
- Open Pools Folder

---

## Tab 5: Routines

**Purpose**: Browse, test, and manage routine definitions and template images

### Sub-Tabs

#### Sub-Tab: Routines (Default)

##### Routines List (Left Panel)
- Tree view organized by directory structure
- Each routine shows:
  - Name (from metadata)
  - File name
  - Category/folder
  - Last modified date
  - Validation status icon
- Search/filter by name, category, tags
- Sort by name, date, category

#### Selected Routine Detail (Main Panel)

**Header**:
- Routine name
- File path
- Last modified timestamp
- Validation status badge (Valid/Invalid/Not Validated)
- Action buttons:
  - Validate
  - Test Run (single bot test)
  - Edit in External Editor
  - Delete
  - Duplicate
  - Show in Folder

**Metadata Section**:
- Description
- Author
- Version
- Tags
- Config parameters (if any)

**Routine Preview** (Tabs):

**Tab: Flowchart View** (Visual representation):
- Node-based diagram of routine flow
- Color-coded by action type:
  - Input: Blue
  - Image detection: Green
  - Control flow: Yellow
  - Account actions: Purple
  - App control: Orange
- Expand nested actions (if/while/repeat)
- Click node to see action details

**Tab: YAML Source**:
- Syntax-highlighted YAML
- Read-only view
- Line numbers
- Copy to clipboard button

**Tab: Action List** (Linear view):
- Numbered list of all actions
- Hierarchical indentation for nested actions
- Action type icons
- Expand/collapse nested structures

**Sentries Section** (if applicable):
- List of configured sentries
- Each shows:
  - Template name
  - Trigger count
  - Actions to execute
  - Visual preview of template

**Statistics Section**:
- Total executions across all bots
- Success rate
- Average runtime
- Most common failure points (if tracked)
- Last 10 execution results

#### Test Run Dialog
- Instance selector (which emulator)
- Account selector (specific account or "next from pool")
- Config variable overrides
- Debug mode toggle
- "Start Test" button
- Real-time execution view with step-by-step progress

##### Toolbar Actions
- Create New Routine (opens template/wizard)
- Refresh from Disk
- Import Routines (drag-and-drop YAML files or browse)
- Validate All
- Open Routines Folder

#### Sub-Tab: Templates

**Purpose**: Manage template images used for image detection in routines

##### Template Library (Grid View)
- Grid of template images organized by category/routine
- Each template card shows:
  - Template image thumbnail
  - Template name
  - File name
  - Resolution
  - Used by (which routines reference this template)
  - Last modified date
- Search/filter by name, category, routine
- Sort by name, date, usage count

##### Selected Template Detail
- Large image preview
- Template metadata:
  - Name, file path, resolution
  - Default matching threshold (if configured)
  - Referenced by (list of routines)
- Actions:
  - Test Template (match against screenshot)
  - Edit Metadata
  - Replace Image
  - Delete
  - Show in Folder
- Template matching test panel:
  - Upload/select screenshot
  - Adjust threshold slider
  - See matching results with confidence score
  - Highlight matching region

##### Toolbar Actions
- Import Templates (drag-and-drop images or browse)
- Organize by Category
- Bulk Rename
- Open Templates Folder
- Refresh from Disk

---

## Tab 6: Accounts

**Purpose**: View and manage all accounts in the database

### Layout

#### Filters & Search Panel (Top)
- Search by device account ID
- Filter by:
  - Status (Available/In-Use)
  - Pack count range (slider)
  - Account fields (custom filters)
  - Pool membership (multi-select pools)
- Sort by:
  - Device account ID
  - Pack count
  - Last used date
  - Creation date
  - Custom fields

#### Accounts Table (Main Area)
- Paginated table (50-100 per page)
- Columns (configurable visibility):
  - Device Account ID
  - Status (badge)
  - Pack Count
  - Poke Gold, Shine Dust, Shop Tickets (custom fields)
  - Last Used Date
  - Currently Assigned To (group/bot instance)
  - Times Used
  - Success Count / Fail Count
  - Created Date
  - Actions (View, Edit, Delete)
- Multi-select rows for batch operations
- Row expansion for detailed view

#### Expanded Row Detail
- All account metadata fields
- Associated XML file path (with open button)
- Execution history:
  - List of routine executions
  - Date, routine name, success/fail, duration
  - Click to view full execution details
- Checkout history (which orchestrations used this account)
- Card collection preview (link to card collection filtered by this account)

#### Bulk Actions Toolbar
- Import Accounts (from folder - drag-and-drop or browse)
- Export Selected (to folder with XML files)
- Delete Selected (with confirmation)
- Update Field (bulk update custom fields)
- Open XML Storage Folder

#### Account Detail Dialog (Double-click row)
- All metadata in editable form
- Account status management
- Manual XML file association
- Execution history table
- Card collection link
- Save/Cancel buttons

#### Statistics Panel (Right Sidebar)
- Total accounts: X
- By status breakdown (Available/In-Use pie chart)
- Average pack count
- Top accounts by packs/gold
- Recently added accounts
- Recently used accounts

---

## Tab 7: Card Collection

**Purpose**: Browse aggregate card data across all accounts

### Layout

#### Set Selector (Top Bar)
- Dropdown to select card set
  - Genetic Apex (A1)
  - Mythical Island (A1a)
  - Future sets...
- Set statistics:
  - Total unique cards: X/Y
  - Total cards collected (all accounts): Z
  - Completion percentage
  - Most collected card
  - Rarest card

#### Card Grid View (Main Area)

**Display Modes**:
- Grid (default): Card thumbnails with counts
- Table: Detailed list view
- Gallery: Large card images with slideshow

**Grid View**:
- Card image thumbnails (or placeholder if image unavailable)
- Each card shows:
  - Card name
  - Card number (e.g., A1-001)
  - Rarity indicator (★ for rare, ◆ for special)
  - Aggregate count across all accounts
  - Collection indicator (badge showing how many accounts have it)
- Click card for detailed view

#### Filters Panel (Left Sidebar)
- **Set** (dropdown)
- **Rarity** (checkboxes):
  - ◆ (1 diamond)
  - ◆◆ (2 diamond)
  - ◆◆◆ (3 diamond)
  - ◆◆◆◆ (4 diamond)
  - ★ (1 star)
  - ★★ (2 star)
  - ★★★ (3 star)
  - Crown (special)
- **Type** (checkboxes):
  - Pokémon
  - Trainer
  - Energy
  - Item
- **Pack** (if applicable):
  - Mewtwo
  - Charizard
  - Pikachu
- **Collection Status**:
  - All cards
  - Collected (at least 1)
  - Not collected (0)
  - Complete playset (2+ copies)
- **Sort By**:
  - Card number
  - Name
  - Total count (most/least collected)
  - Rarity
  - Accounts that have it

#### Card Detail Dialog
- Large card image
- Card metadata:
  - Name, number, rarity
  - Set, pack source
  - Card type
- Aggregate statistics:
  - Total copies across all accounts: X
  - Number of accounts with this card: Y
  - Accounts with multiple copies: Z
  - Average copies per account (among those who have it)
- Account breakdown table:
  - Device Account | Count
  - Sortable and filterable
  - Click to view account details
- Card history:
  - First seen date
  - Most recent acquisition
  - Acquisition trend chart (if historical data available)

#### Statistics Dashboard (Right Sidebar)
- **Collection Progress**:
  - Unique cards collected: X/Y (percentage)
  - Progress bar by rarity
  - Progress bar by type
- **Top Cards**:
  - Most collected card (name + count)
  - Rarest card in collection
- **Recent Additions**:
  - Last 10 cards added (any account)
  - Timestamp and source account
- **Missing Cards**:
  - Number of uncollected cards
  - "View Missing" button (filters grid)

#### Export/Import
- Export collection data to CSV/JSON
- Import card database (card definitions, images)
- Sync with external card database/API

---

## Tab 8: History & Analytics

**Purpose**: Historical view and analytics of all routine executions and system performance

### Layout

#### Time Range Selector (Top Bar)
- Preset ranges: Last 24 Hours, Last 7 Days, Last 30 Days, Last 90 Days, All Time, Custom Range
- Date range picker for custom selection
- Auto-refresh toggle (update charts in real-time)

#### Summary Cards (Top Section)
- **Total Executions**: Count for selected time range
- **Success Rate**: Percentage with trend indicator
- **Total Runtime**: Aggregate runtime with average per execution
- **Accounts Processed**: Unique accounts used
- **Total Iterations**: Sum of all iterations across executions

#### Analytics Tabs

##### Tab: Execution Timeline
- **Chart**: Line/area chart showing executions over time
  - X-axis: Time (hour/day/week based on range)
  - Y-axis: Number of executions
  - Color-coded by success/failure
  - Hover for details
- **Filters**: Group name, Routine name, Status
- **Export**: Download chart as image or data as CSV

##### Tab: Performance Metrics
- **Charts**:
  - Average runtime trend (line chart)
  - Success rate trend (line chart)
  - Execution distribution by hour of day (bar chart)
  - Execution distribution by day of week (bar chart)
- **Top Performers**:
  - Fastest routines (avg runtime)
  - Most reliable routines (success rate)
  - Most used routines (execution count)
- **Bottlenecks**:
  - Slowest routines
  - Highest failure rates
  - Most common failure points

##### Tab: Routine Breakdown
- **Table**: List of all routines with aggregated statistics
  - Routine name
  - Total executions
  - Success rate
  - Avg runtime, Min runtime, Max runtime
  - Last executed
  - Click to filter execution history
- **Charts per Routine**:
  - Success/failure distribution
  - Runtime distribution (histogram)
  - Execution trend over time

##### Tab: Account Performance
- **Table**: Accounts with execution statistics
  - Device account ID
  - Times used
  - Success rate when used
  - Avg packs opened (if tracked)
  - Total runtime
  - Last used date
- **Charts**:
  - Most used accounts (bar chart)
  - Account success rates (comparison)

##### Tab: Group Analysis
- **Table**: Bot groups with historical data
  - Group name
  - Total launches
  - Avg bots per launch
  - Total runtime
  - Success rate
  - Accounts consumed
- **Charts**:
  - Group usage over time
  - Bot count distribution
  - Group performance comparison

#### Execution History Table (Bottom Section)
- Detailed table of all routine executions
- **Columns** (sortable, filterable):
  - Execution ID
  - Date/Time
  - Routine Name
  - Bot Group
  - Account (device account ID)
  - Bot Instance ID
  - Duration
  - Status (Success/Failed/Stopped)
  - Error Message (if failed)
  - Iterations Completed
  - Packs Opened (if tracked)
- **Filters**:
  - Date range
  - Routine name
  - Group name
  - Account
  - Status
  - Instance ID
- **Actions**:
  - View details (opens detailed execution view)
  - View logs (if available)
  - Re-run with same config
- **Pagination**: 50-100 per page
- **Export**: Export filtered results to CSV/JSON

#### Detailed Execution View (Modal/Dialog)
- Full execution details:
  - All metadata (execution ID, timestamps, duration, etc.)
  - Configuration used (routine config variables)
  - Account details
  - Step-by-step log (if available)
  - Error stack trace (if failed)
  - Metrics collected during execution
  - Screenshots captured (if any)
- Actions: Re-run, Export, Close

### Toolbar Actions
- Refresh Data
- Export Current View (CSV/JSON)
- Clear History (with retention period selector)
- Open Logs Folder

---

## Tab 9: Settings

**Purpose**: Application configuration and preferences

### Layout: Accordion/Category Sections

#### General Settings
- **Startup**:
  - Show splash screen (toggle)
  - Default tab on startup (dropdown)
  - Auto-load account pools (toggle)
  - Auto-load routines (toggle)
- **UI Preferences**:
  - Theme (Light/Dark/System)
  - Accent color picker
  - Font size (slider)
  - Compact mode (toggle for smaller UI elements)
  - Show developer tools (toggle for Event Stream tab)

#### Database Settings
- **Connection**:
  - Database file path (read-only, with browse button)
  - Connection pool size
  - Test connection button
- **Maintenance**:
  - Optimize database button
  - Backup database button
  - Restore from backup
  - Clear execution history (with retention period)
  - Database statistics (size, table counts)

#### Emulator Settings
- **MuMu Player**:
  - MuMu installation path (browse)
  - MuMu player path (browse)
  - Auto-discover instances on startup
  - Health check interval (seconds)
  - ADB port range
- **Instance Management**:
  - Default emulator timeout
  - Window detection method
  - Auto-launch instances (toggle)
  - Auto-connect ADB (toggle)

#### Account Pool Settings
- **Default Behavior**:
  - Auto-refresh pools on startup
  - Auto-refresh interval (minutes, 0=disabled)
  - XML storage directory (browse)
  - Pool definitions directory (browse)
- **Import Defaults**:
  - Default watched folder
  - Auto-import on folder change

#### Routine Settings
- **Directories**:
  - Routines directory (browse)
  - Template image directory (browse)
  - Examples directory (browse)
- **Execution**:
  - Default template matching threshold
  - Default action timeout
  - Screenshot on failure (toggle)
  - Screenshot storage directory

#### Bot Group Settings
- **Defaults**:
  - Default bot count
  - Default stagger delay
  - Default conflict resolution
  - Group definitions directory
- **Orchestration**:
  - Max concurrent groups
  - Default restart policy
  - Account wait timeout (minutes)

#### Logging & Events
- **Log Settings**:
  - Log level (Debug/Info/Warn/Error)
  - Log file directory (browse)
  - Log rotation (daily/size-based)
  - Max log file size
  - Retention period (days)
- **Event Bus**:
  - Event buffer size
  - Enable event logging to file
  - Event history retention
- **Event Stream Tab**:
  - Max events displayed
  - Auto-scroll (toggle)
  - Event filters (which event types to show)

#### Notifications
- **System Notifications**:
  - Enable desktop notifications (toggle)
  - Notify on group completion
  - Notify on group failure
  - Notify on account exhaustion
  - Notify on critical errors
- **In-App Alerts**:
  - Toast notification duration
  - Alert sound (toggle + sound file picker)

#### Advanced
- **Performance**:
  - UI update frequency (ms)
  - Enable hardware acceleration
  - Max concurrent bot instances (system-wide limit)
- **Debugging**:
  - Enable verbose logging
  - Export debug bundle (logs + config + db schema)
  - Reset all settings to defaults
- **Data Management**:
  - Clear all caches
  - Reset window positions
  - Export all configurations
  - Import configurations

#### About
- Application version
- Build date/commit hash
- License information
- Links: Documentation, GitHub, Report Issue
- Check for updates button

---

## Additional UI Components

### Global Elements

#### Top Menu Bar (Optional, platform-dependent)
- **File**:
  - New Bot Group
  - New Account Pool
  - Import...
  - Export...
  - Settings
  - Exit
- **View**:
  - Toggle Developer Tools
  - Toggle Event Stream
  - Refresh All Data
  - Full Screen
- **Tools**:
  - Test Routine
  - Validate All Routines
  - Import Accounts
  - Database Console
  - Emergency Stop All
- **Help**:
  - Documentation
  - Keyboard Shortcuts
  - Report Issue
  - About

#### Status Bar (Bottom)
- **Left Side**:
  - Active groups count: "X groups running, Y bots active"
  - Account pools status: "Z accounts available"
- **Right Side**:
  - Emulator health: "W/X instances ready"
  - Database indicator (connected/disconnected)
  - Event bus status (events/sec)
  - Settings icon (quick access)

#### Toast Notifications
- Non-blocking notifications for:
  - Group launched successfully
  - Group completed/failed
  - Account pool refreshed
  - Validation errors
  - Critical system events
- Auto-dismiss after 5 seconds (configurable)
- Click to view details
- Action buttons for relevant actions

#### Context Menus (Right-Click)
- **Bot Groups List**: Launch, Edit, Duplicate, Delete, Export, Pin/Unpin
- **Account Pools List**: Edit, Test, Duplicate, Delete, Refresh, Export
- **Routines List**: Validate, Test, Edit, Duplicate, Delete, Export
- **Accounts Table**: View, Edit, Delete, Export, Mark Failed/Available
- **Bot Instances (Live Monitor)**: Stop, Restart, View Logs, Screenshot

---

## Missing/Supplementary Features

### Features Not Covered by Main Tabs

#### 1. Templates Management (New Tab or Settings Subsection)
**Purpose**: Manage template images used by routines for image detection

**Functionality**:
- Browse all template images
- Organized by category/routine
- View template metadata (resolution, matching threshold)
- Test template matching against screenshots
- Import new templates
- Bulk rename/reorganize

**Could be**:
- Sub-section in Routines tab ("Templates" sub-tab)
- Settings subsection
- Or dedicated tab if templates become numerous

---

#### 2. Execution History / Analytics (New Tab)
**Purpose**: Historical view of all routine executions across time

**Functionality**:
- Filterable table of all executions:
  - Date/time, routine, account, group, duration, status, error
- Aggregate statistics:
  - Executions per day/week/month
  - Success rate trends
  - Average runtime trends
  - Most used routines
  - Most successful accounts
  - Failure analysis (which steps fail most often)
- Charts and graphs:
  - Execution timeline
  - Success rate over time
  - Routine performance comparison
- Export to CSV for external analysis

**Placement Options**:
- Dedicated "Analytics" or "History" tab
- Sub-tab under Live Monitor
- Accessible via context menu from Bot Groups

---

#### 3. Emulator Instances Manager (New Tab or Settings Subsection)
**Purpose**: Manage emulator instances and health monitoring

**Functionality**:
- List all discovered emulator instances
- Health status for each (window detected, ADB connected)
- Manual controls:
  - Launch instance
  - Connect/Disconnect ADB
  - Close instance
  - Restart instance
- Instance configuration:
  - Port assignments
  - Window titles
  - Priority/ordering
- Health monitoring settings per instance
- Test connection button

**Placement Options**:
- Dedicated "Emulators" tab (if users have many instances)
- Sub-section of Settings
- Collapsible panel in Live Monitor

---

#### 4. Backup & Restore (Settings Subsection or Separate Dialog)
**Purpose**: Backup and restore system state

**Functionality**:
- Backup:
  - Database (accounts, executions, metadata)
  - Bot group definitions
  - Account pool definitions
  - Routine files
  - Templates
  - Configuration files
  - Create backup bundle (.zip)
- Restore:
  - Select backup file
  - Choose what to restore (selective)
  - Preview backup contents
  - Restore with confirmation
- Scheduled backups:
  - Auto-backup on interval
  - Backup before major operations

**Placement**: Settings → Data Management → Backup/Restore subsection

---

#### 5. Card Database Management (Card Collection Subsection)
**Purpose**: Manage the card definition database

**Functionality**:
- Import card data (JSON/CSV with card definitions)
- Import card images (bulk import from folder)
- Edit card metadata (if definitions incorrect)
- Map card IDs to images
- Sync with online card database (future)
- Export card database

**Placement**: Card Collection → "Manage Database" button or Settings subsection

---

#### 6. Keyboard Shortcuts (Help Dialog or Settings)
**Purpose**: Display and configure keyboard shortcuts

**Functionality**:
- List all keyboard shortcuts
- Customizable key bindings
- Search shortcuts by action
- Reset to defaults

**Common Shortcuts**:
- `Ctrl+N`: New Bot Group
- `Ctrl+L`: Launch Selected Group
- `Ctrl+S`: Stop All Groups
- `Ctrl+R`: Refresh Current View
- `Ctrl+,`: Open Settings
- `Ctrl+F`: Focus Search/Filter
- `F5`: Refresh Data
- `Esc`: Close Dialog/Cancel

**Placement**: Help menu or Settings → General → Keyboard Shortcuts

---

#### 7. Import/Export Wizards (Dialogs)
**Purpose**: Guided workflows for complex import/export operations

**Import Account Wizard**:
1. Select source (folder, XML files, CSV)
2. Preview accounts to import
3. Conflict resolution (skip, overwrite, rename)
4. Target pool selection (optional)
5. Import progress
6. Summary report

**Export Configuration Wizard**:
1. Select what to export (groups, pools, routines, settings)
2. Choose export format (YAML, JSON, ZIP bundle)
3. Select destination
4. Export progress
5. Success confirmation

**Placement**: Accessible via toolbar buttons or File menu

---

#### 8. Plugin/Extension System (Future Consideration)
**Purpose**: Allow users to extend bot functionality

**Potential Features**:
- Custom routine actions (Go plugins)
- Custom UI panels
- External integrations (Discord webhooks, etc.)
- Custom analytics/reports

**Placement**: Settings → Plugins/Extensions

---

## Design Patterns & Best Practices

### Responsive Layouts
- Minimum window size: 1280x720
- Flexible layouts that adapt to window resizing
- Collapsible panels to maximize space
- Responsive tables with horizontal scroll if needed

### Consistent Interactions
- **Primary Actions**: Large, prominent buttons (e.g., "Launch")
- **Secondary Actions**: Smaller buttons or icon buttons
- **Destructive Actions**: Red buttons with confirmation dialogs
- **Hover States**: Visual feedback on all interactive elements
- **Loading States**: Spinners/progress bars for async operations
- **Empty States**: Helpful illustrations and calls-to-action

### Error Handling
- Inline validation with helpful error messages
- Toast notifications for non-critical errors
- Modal dialogs for critical errors requiring user action
- Detailed error logs accessible via "View Details" button
- Actionable error messages ("Check database connection" vs "Error 500")

### Accessibility
- Keyboard navigation for all UI elements
- Screen reader support where applicable
- High contrast mode support
- Resizable text/UI elements
- Focus indicators for keyboard navigation

### Performance
- Lazy loading for large data sets (pagination, virtual scrolling)
- Debounced search/filter inputs
- Background loading with progress indicators
- Efficient re-rendering (only update changed components)
- Caching frequently accessed data

---

## Data Flow & Real-Time Updates

### Event Bus Integration
- UI subscribes to relevant event types per tab
- Real-time updates without manual refresh:
  - Live Monitor: Bot state changes, execution steps
  - Quick Launch: Group status updates
  - Accounts: Account status changes, checkouts
  - Account Pools: Pool refresh events
- Event throttling to prevent UI overload
- Configurable update frequency

### WebSocket/Polling Alternatives
- If event bus not exposed to UI layer, consider:
  - Polling at configurable intervals
  - WebSocket server for real-time push
  - Hybrid: Critical events pushed, non-critical polled

---

## Visual Design Language

### Color Palette (Suggestions)
- **Primary**: Blue (#007AFF) - Actionable elements, primary buttons
- **Success**: Green (#34C759) - Successful operations, completed status
- **Warning**: Orange (#FF9500) - Warnings, in-progress states
- **Danger**: Red (#FF3B30) - Errors, destructive actions, failed status
- **Info**: Cyan (#00C7BE) - Informational badges, neutral states
- **Neutral**: Grays for backgrounds, borders, disabled states

### Typography
- **Headers**: Bold, larger font for tab titles, section headers
- **Body**: Regular weight for general content
- **Monospace**: For code, IDs, file paths, YAML snippets
- **Icons**: Consistent icon set (Material Icons, Feather, or similar)

### Spacing & Layout
- **Padding**: Consistent padding (8px, 16px, 24px units)
- **Cards**: Elevated cards with subtle shadows for grouping
- **Dividers**: Subtle lines to separate sections
- **Grid System**: Responsive grid for card layouts

---

## Development Phases (Recommended)

### Phase 1: Foundation (MVP)
- Splash screen with initialization
- Quick Launch tab (basic version with pinning)
- Live Monitor tab (real-time bot status)
- Bot Groups tab (CRUD operations)
- Settings tab (essential settings only)
- Basic navigation and layout structure

### Phase 2: Core Functionality
- Account Pools tab (full management)
- Routines tab (browse, validate, test)
- Accounts tab (view, filter, basic editing)
- Complete Settings implementation
- Event bus integration for real-time updates

### Phase 3: Advanced Features
- Card Collection tab (full implementation)
- Launch overrides dialog (full featured)
- Execution History/Analytics
- Templates management
- Emulator instances manager
- Advanced filtering and search across all tabs

### Phase 4: Polish & Optimization
- Performance optimizations (virtual scrolling, caching)
- Enhanced visualizations (charts, graphs, flowcharts)
- Import/Export wizards
- Keyboard shortcuts customization
- Comprehensive error handling and validation
- User onboarding/tutorial

### Phase 5: Future Enhancements
- Plugin system
- Cloud sync (optional)
- Mobile companion app (monitoring only)
- Advanced analytics and reporting
- External integrations (Discord, webhooks)
- Theme customization

---

## Design Decisions (Finalized)

1. **Emulator Instances Panel**: ✅ Part of Live Monitor (right sidebar with idle/running/offline sections)

2. **Templates Management**: ✅ Sub-tab within Routines tab (keeps related features together)

3. **Execution History & Analytics**: ✅ Dedicated Tab 8 with comprehensive analytics and charts

4. **Developer Tools**: ✅ Toggle in Settings → General → "Show developer tools"

5. **Event Stream**: ✅ Part of Live Monitor:
   - System events in bottom panel (collapsible, requires developer tools enabled)
   - Instance/bot-specific events on respective bot cards

6. **Multi-Window Support**: Future feature - anchored mini monitors/controllers to instance windows

7. **Drag-and-Drop**: ✅ Enabled for:
   - Reordering pinned groups
   - Importing YAML files (routines, pools, groups)
   - Importing accounts/templates

8. **Themes**: ✅ Light/Dark/System with optional color profile customization (low priority)

9. **Card Images**: ✅ Store locally, fetch new sets from API during splash screen (future enhancement)

10. **Account Pool Refresh**: ✅ Multiple triggers:
    - Manual refresh in Account Pools tab
    - Auto-refresh on group launch
    - Auto-refresh when running group exhausts pool

11. **Account Statuses**: ✅ Only Available/In-Use (Completed/Failed are routine-execution specific, not account-level)

12. **Account Pool Type**: ✅ No type indicator needed - all pools use unified hybrid approach (queries + includes/excludes + watched paths)

13. **Export Functions**: ✅ "Show in Folder" instead of "Export to YAML" (files always stored as YAML on disk)

14. **Pool Selection**: ✅ Multi-select to aggregate from multiple pools

---

## Dependency Injection & Data Binding Architecture

### Overview

The GUI needs access to the backend systems for executing operations and receiving real-time updates. This section describes how dependencies are injected and how data flows between the backend and UI.

### Dependency Injection Strategy

#### GUI Application Context

Create a central `AppContext` struct that holds references to all backend components:

```go
type AppContext struct {
    // Core components
    Orchestrator     *bot.Orchestrator
    Database         *sql.DB
    PoolManager      *accountpool.PoolManager
    EmulatorManager  *emulator.Manager
    TemplateRegistry *templates.TemplateRegistry
    RoutineRegistry  *actions.RoutineRegistry
    EventBus         events.EventBus

    // Configuration
    Config *bot.Config

    // UI state (if needed)
    Settings *GUISettings
}
```

#### Initialization Flow

1. **Main Function** initializes all backend components (same as current implementation)
2. **Creates AppContext** with references to all initialized components
3. **Passes AppContext** to GUI initialization
4. **GUI stores AppContext** and uses it for all backend operations

```go
func main() {
    // Initialize backend
    db := initializeDatabase()
    poolManager := accountpool.NewPoolManager(...)
    orchestrator := bot.NewOrchestrator(...)
    // ... initialize all components

    // Create application context
    appCtx := &AppContext{
        Orchestrator:     orchestrator,
        Database:         db,
        PoolManager:      poolManager,
        EmulatorManager:  orchestrator.GetEmulatorManager(),
        TemplateRegistry: orchestrator.GetTemplateRegistry(),
        RoutineRegistry:  orchestrator.GetRoutineRegistry(),
        EventBus:         orchestrator.GetEventBus(),
        Config:           config,
        Settings:         loadGUISettings(),
    }

    // Launch GUI with context
    RunGUI(appCtx)
}
```

### Data Binding Strategies

The UI uses a **hybrid approach** combining Fyne data binding, direct event bus subscriptions, and manual updates:

#### 1. Fyne Data Binding (For Simple State)

Use Fyne's built-in `binding` package for simple, UI-only state:

**Use Cases**:
- Form inputs (routine config variables, pool filters)
- UI settings (theme, font size, checkboxes)
- Simple toggles and selections

**Example**:
```go
// In a form
botCountBinding := binding.NewInt()
botCountBinding.Set(groupDef.RequestedBotCount)

// Widget
botCountSlider := widget.NewSliderWithData(1, 10, botCountBinding)

// Get value when saving
botCount, _ := botCountBinding.Get()
```

**Limitations**: Fyne bindings are UI-only and don't connect to backend state directly.

#### 2. Event Bus Subscriptions (For Real-Time Updates)

Use the Event Bus for real-time updates from backend to UI:

**Use Cases**:
- Bot status changes
- Group launched/stopped events
- Account pool refresh events
- Health monitor updates
- Execution progress updates

**Pattern**:
```go
// In tab initialization
func (t *LiveMonitorTab) Initialize(appCtx *AppContext) {
    // Subscribe to relevant events
    appCtx.EventBus.Subscribe(events.EventTypeBotStarted, t.onBotStarted)
    appCtx.EventBus.Subscribe(events.EventTypeBotStopped, t.onBotStopped)
    appCtx.EventBus.Subscribe(events.EventTypeBotFailed, t.onBotFailed)
    appCtx.EventBus.Subscribe(events.EventTypeGroupLaunched, t.onGroupLaunched)
    appCtx.EventBus.Subscribe(events.EventTypeGroupStopped, t.onGroupStopped)
    // ... more subscriptions
}

// Event handler updates UI
func (t *LiveMonitorTab) onBotStarted(event events.Event) {
    // Extract event data
    groupName := event.Data["group_name"].(string)
    instanceID := event.Data["instance_id"].(int)

    // Update UI (must be on UI thread)
    t.updateBotCard(groupName, instanceID, BotStatusRunning)
    t.window.Canvas().Refresh(t.botCardsContainer)
}
```

**Important**: Fyne requires UI updates to happen on the main/UI thread. Use `fyne.CurrentApp().Driver().DoEventually()` or similar for thread-safe updates from event handlers.

**Thread-Safe UI Updates**:
```go
func (t *LiveMonitorTab) onBotStarted(event events.Event) {
    // Capture data outside of goroutine
    groupName := event.Data["group_name"].(string)
    instanceID := event.Data["instance_id"].(int)

    // Schedule UI update on main thread
    t.app.Driver().DoEventually(func() {
        t.updateBotCard(groupName, instanceID, BotStatusRunning)
    })
}
```

#### 3. Polling (For Non-Event-Driven Data)

For data that doesn't emit events, use periodic polling with timers:

**Use Cases**:
- Account pool statistics (if not evented)
- Emulator health status (if not using health change callbacks)
- Database statistics

**Pattern**:
```go
func (t *QuickLaunchTab) startPolling(appCtx *AppContext) {
    ticker := time.NewTicker(5 * time.Second)
    go func() {
        for range ticker.C {
            // Fetch data
            stats := appCtx.PoolManager.GetPoolStats("main_pool")

            // Update UI on main thread
            t.app.Driver().DoEventually(func() {
                t.updatePoolStats(stats)
            })
        }
    }()
}
```

**Note**: Prefer event bus over polling when possible for better performance and responsiveness.

#### 4. Direct Backend Calls (For User Actions)

UI components call backend methods directly for user-initiated actions:

**Use Cases**:
- Launching a bot group
- Creating/editing definitions
- Importing accounts
- Validating routines

**Pattern**:
```go
func (t *BotGroupsTab) onLaunchButtonClicked(groupName string, overrides *bot.LaunchOverrides) {
    // Show loading indicator
    t.showLoadingOverlay()

    // Call backend in goroutine (don't block UI)
    go func() {
        result, err := t.appCtx.Orchestrator.LaunchGroupWithOverrides(groupName, overrides)

        // Update UI on main thread
        t.app.Driver().DoEventually(func() {
            t.hideLoadingOverlay()

            if err != nil {
                t.showError(fmt.Sprintf("Failed to launch group: %v", err))
                return
            }

            // Show success
            t.showSuccess(fmt.Sprintf("Launched %d bots successfully", result.LaunchedBots))

            // Navigate to Live Monitor
            t.navigateToTab("Live Monitor")
        })
    }()
}
```

### Event Bus Integration Details

#### Subscription Management

**Tab-Level Subscriptions**:
- Each tab subscribes to relevant events in its `Initialize()` method
- Unsubscribe when tab is destroyed/hidden (if needed)
- Use filters to only process relevant events

**Example Tab Structure**:
```go
type LiveMonitorTab struct {
    appCtx            *AppContext
    app               fyne.App
    window            fyne.Window
    subscriptions     []events.Subscription
    botCards          map[int]*BotCard
    groupList         *widget.List
    // ... more fields
}

func (t *LiveMonitorTab) Initialize(appCtx *AppContext) {
    t.appCtx = appCtx

    // Subscribe to events
    sub1 := appCtx.EventBus.Subscribe(events.EventTypeBotStarted, t.onBotStarted)
    sub2 := appCtx.EventBus.Subscribe(events.EventTypeGroupLaunched, t.onGroupLaunched)
    // ... more subscriptions

    // Store subscriptions for cleanup
    t.subscriptions = []events.Subscription{sub1, sub2}
}

func (t *LiveMonitorTab) Cleanup() {
    // Unsubscribe from all events
    for _, sub := range t.subscriptions {
        t.appCtx.EventBus.Unsubscribe(sub)
    }
}
```

#### Event Filtering

Filter events at the handler level to avoid unnecessary UI updates:

```go
func (t *LiveMonitorTab) onBotStarted(event events.Event) {
    // Only process if this is the selected group
    groupName := event.Data["group_name"].(string)
    if groupName != t.selectedGroupName {
        return // Ignore events for non-selected groups
    }

    // Process event...
}
```

### Data Refresh Patterns

#### On-Demand Refresh

User clicks refresh button → Direct backend call → Update UI:

```go
func (t *AccountPoolsTab) onRefreshClicked() {
    poolName := t.selectedPool

    go func() {
        // Refresh pool in backend
        err := t.appCtx.PoolManager.RefreshPool(poolName)

        t.app.Driver().DoEventually(func() {
            if err != nil {
                t.showError(fmt.Sprintf("Refresh failed: %v", err))
                return
            }

            // Reload pool data
            pool, _ := t.appCtx.PoolManager.GetPool(poolName)
            stats := pool.GetStats()
            t.updatePoolStats(stats)
        })
    }()
}
```

#### Auto-Refresh on Events

Pool refreshes automatically (e.g., on group launch) → Event published → UI updates:

```go
// Backend publishes event after refresh
eventBus.PublishAsync(events.NewPoolRefreshedEvent(poolName, stats))

// UI subscribes and updates
func (t *AccountPoolsTab) onPoolRefreshed(event events.Event) {
    poolName := event.Data["pool_name"].(string)
    stats := event.Data["stats"].(accountpool.PoolStats)

    t.app.Driver().DoEventually(func() {
        t.updatePoolStats(poolName, stats)
    })
}
```

### Complex UI State Management

For complex state (like bot instance cards with multiple fields updating), consider a **ViewModel pattern**:

```go
type BotInstanceViewModel struct {
    InstanceID       int
    Status           bot.BotStatus
    CurrentAccount   string
    CurrentStep      string
    Runtime          time.Duration
    Iterations       int
    HealthWindow     bool
    HealthADB        bool
    RecentEvents     []events.Event

    // UI widgets
    statusLabel      *widget.Label
    accountLabel     *widget.Label
    stepLabel        *widget.Label
    // ... more widgets
}

func (vm *BotInstanceViewModel) UpdateStatus(status bot.BotStatus) {
    vm.Status = status
    vm.statusLabel.SetText(string(status))
    vm.statusLabel.Refresh()
}

func (vm *BotInstanceViewModel) UpdateCurrentStep(step string) {
    vm.CurrentStep = step
    vm.stepLabel.SetText(step)
    vm.stepLabel.Refresh()
}
```

Then event handlers simply call ViewModel update methods:

```go
func (t *LiveMonitorTab) onBotStarted(event events.Event) {
    instanceID := event.Data["instance_id"].(int)

    t.app.Driver().DoEventually(func() {
        if vm, exists := t.botViewModels[instanceID]; exists {
            vm.UpdateStatus(bot.BotStatusRunning)
        }
    })
}
```

### Summary: Data Flow Patterns

1. **Backend → UI (Real-Time)**:
   - Event Bus subscription
   - Event handler extracts data
   - Updates UI on main thread

2. **UI → Backend (User Actions)**:
   - User interaction triggers handler
   - Handler calls AppContext backend method (in goroutine)
   - Result updates UI on main thread

3. **UI ↔ UI (Form State)**:
   - Fyne data bindings for simple forms
   - No backend involvement

4. **Periodic Updates**:
   - Timer-based polling (when events not available)
   - Fetch data from backend
   - Update UI on main thread

### Best Practices

1. **Always update UI on main thread** using `Driver().DoEventually()`
2. **Avoid blocking UI thread** - use goroutines for backend calls
3. **Unsubscribe from events** when tabs/components are destroyed
4. **Filter events** early to avoid unnecessary processing
5. **Use ViewModels** for complex component state
6. **Batch updates** when possible to reduce refresh calls
7. **Show loading indicators** for long-running operations
8. **Handle errors gracefully** with user-friendly messages

---

## Conclusion

This UI architecture provides a comprehensive, user-friendly interface for managing all aspects of the Pokemon TCG Pocket bot. The design prioritizes:

- **Quick access** to common operations (Quick Launch)
- **Real-time visibility** into active operations (Live Monitor)
- **Complete management** of all system components (dedicated tabs)
- **Flexibility** through overrides and customization
- **Scalability** for future features and growth

The tabbed structure keeps the interface organized while remaining intuitive for both novice and power users. Real-time updates via the event bus ensure users have immediate feedback on system state without manual refreshes.

By following the recommended development phases, you can deliver a functional MVP quickly while building toward a feature-rich, polished application.
