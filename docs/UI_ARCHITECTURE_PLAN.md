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
8. **Settings**

Optional tabs (context-dependent):
- **Logs Viewer** (toggleable via settings)
- **Event Stream** (developer mode)

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
- Account pool stats: Available/In-Use/Completed/Failed

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

**Real-Time Event Stream** (Bottom Panel, Collapsible):
- Scrolling feed of events from EventBus
- Filterable by: Bot instance, Event type, Severity
- Color-coded by event type
- Auto-scroll toggle
- Export/save logs

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
  - Export to YAML

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
  - Pool selector (dropdown)
  - Pool stats preview (available accounts)
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
- Import from YAML
- Export Selected
- Delete Selected (batch)
- Refresh from Disk

---

## Tab 4: Account Pools

**Purpose**: Manage account pool definitions and view pool statistics

### Layout

#### Pools List (Left Panel)
- All saved pool definitions
- Each shows:
  - Pool name
  - Total accounts (from last refresh)
  - Type indicator (Query-based, File-based, Hybrid)
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
    - Completed: W
    - Failed: V
  - Refresh button (manual pool refresh)
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
  - Export to YAML
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
- Import from YAML
- Refresh All Pools
- Delete Selected

---

## Tab 5: Routines

**Purpose**: Browse, test, and manage routine definitions

### Layout

#### Routines List (Left Panel)
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
  - Export/Share

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

#### Toolbar Actions
- Create New Routine (opens template/wizard)
- Refresh from Disk
- Import Routine
- Validate All
- Open Routines Folder

---

## Tab 6: Accounts

**Purpose**: View and manage all accounts in the database

### Layout

#### Filters & Search Panel (Top)
- Search by device account ID
- Filter by:
  - Status (Available/In-Use/Completed/Failed)
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
- Import Accounts (from folder)
- Export Selected (to folder with XML files)
- Delete Selected (with confirmation)
- Update Field (bulk update custom fields)
- Mark as Failed/Available (bulk status change)
- Assign to Pool (temporary assignment)

#### Account Detail Dialog (Double-click row)
- All metadata in editable form
- Account status management
- Manual XML file association
- Execution history table
- Card collection link
- Save/Cancel buttons

#### Statistics Panel (Right Sidebar)
- Total accounts: X
- By status breakdown (pie chart)
- Average pack count
- Top accounts by packs/gold
- Recently added accounts
- Accounts needing attention (failed, stale)

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

## Tab 8: Settings

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

## Open Questions & Decisions

1. **Emulators Tab**: Dedicated tab vs Settings subsection vs Live Monitor panel?
   - **Recommendation**: Settings subsection for now, promote to tab if complex

2. **Templates Management**: Separate tab vs Routines sub-tab vs Settings?
   - **Recommendation**: Routines sub-tab (Templates), keeps related features together

3. **Execution History**: Separate tab vs accessible via context menus?
   - **Recommendation**: Separate "Analytics" tab for historical/statistical views

4. **Developer Tools**: Always visible vs toggle in Settings?
   - **Recommendation**: Toggle in Settings → General → "Show developer tools"

5. **Event Stream**: Dedicated tab vs Live Monitor panel vs modal window?
   - **Recommendation**: Optional tab (enabled via developer tools), collapsible panel in Live Monitor

6. **Multi-Window Support**: Single window vs multiple windows (e.g., pop-out Live Monitor)?
   - **Recommendation**: Single window for MVP, consider multi-window in Phase 4

7. **Drag-and-Drop**: Enable for reordering, file imports, etc.?
   - **Recommendation**: Yes, for pinned groups, instance selection, file imports

8. **Themes**: Just Light/Dark or full theme customization?
   - **Recommendation**: Light/Dark/System for Phase 1, custom themes in Phase 4

9. **Card Images**: Store locally, fetch from API, or both?
   - **Recommendation**: Store locally with optional API sync (future)

10. **Account Pool Refresh**: Manual only or auto-refresh with interval?
    - **Recommendation**: Both, configurable in Settings

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
