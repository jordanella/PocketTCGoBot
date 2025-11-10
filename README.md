# PocketTCG Bot - Go Edition

A high-performance Pokemon TCG Pocket automation bot written in Go, designed to build upon the PTCGPB project with improved reliability, efficiency, and maintainability through a complete architectural rewrite.

This is built using https://github.com/kevnITG/PTCGPB as a model. kevinnnn is a brilliant developer and none of this would be possible without the project he maintains.

## Overview

PocketTCG Bot automates Pokemon TCG Pocket gameplay across multiple MuMu Player emulator instances with sophisticated account management, YAML-based routine scripting, and parallel error monitoring via sentries. Built with Go for zero-dependency deployment and native performance.

**Status:** v0.1.0 - Core infrastructure complete, production architecture implemented

### Key Features

- **Bot Group Orchestration** - Coordinate multiple bot instances with unique execution contexts (orchestration IDs)
- **YAML-Based Routines** - Script automation workflows with 41 actions, conditionals, loops, and variables
- **Sentry Supervision** - Parallel error monitoring routines for autonomous recovery
- **Account Pool System** - SQL-based account querying with database checkout mutex for conflict-free multi-orchestration
- **Template Registry** - 236+ pre-defined CV templates with YAML definitions and image caching
- **Computer Vision** - Pure Go template matching for screen detection and navigation
- **Multi-Instance Coordination** - Shared registries with per-instance state isolation
- **GUI Interface** - Cross-platform Fyne-based management console
- **ADB Integration** - Direct Android Debug Bridge control for emulator management

### Why Go?

- **No Runtime Dependencies** - Single executable, no Python environment or package management
- **Native Performance** - Faster and more reliable than scripting languages
- **Windows-Native** - Built specifically for Windows with MuMu Player integration
- **Pure Go CV** - No OpenCV or external CV libraries required
- **Easy Deployment** - Distribute a single .exe file to users

## Current Capabilities

### Core Infrastructure ✅ Complete
- **41 Actions** - Click, swipe, CV, loops, variables, conditionals, account management
- **Routine System** - YAML-based scripting with eager loading registry
- **Sentry Engine** - Parallel error monitoring with recovery actions
- **Template Registry** - Image caching with YAML definitions and 236+ templates
- **Variable System** - Per-instance stores with `${variable}` interpolation
- **Config System** - User-configurable parameters with runtime overrides
- **Bot Group Orchestrator** - Multi-instance coordination with shared registries
- **Routine State Machine** - Idle/Running/Paused/Stopped/Completed lifecycle
- **Account Pool Manager** - SQL queries, manual include/exclude, watched paths
- **Database Checkout System** - Global mutex preventing duplicate account injections
- **Orchestration ID System** - UUID-based execution context isolation
- **MVC Architecture** - Proper separation for templates, routines, and account pools

### Database Integration ✅ Complete
- **SQLite Backend** - Account storage, routine executions, checkout tracking
- **Migration System** - 11 migrations with proper versioning
- **Account Lifecycle** - Track routine executions per account with orchestration context
- **Checkout Mutex** - Database columns for orchestration/instance tracking
- **Stale Detection** - 10-minute timeout for crash recovery

### GUI Features ✅ Complete
- **Multi-tab Interface** - Dashboard, bot launcher, ADB test, config
- **Account Pool Wizard** - Visual query builder for pool definitions
- **Template Manager** - Load and cache templates from YAML
- **Routine Browser** - View, validate, and reload routines
- **Emulator Manager** - MuMu instance detection and management

### In Development 🚧
- Individual bot lifecycle controls (pause/resume/stop per instance)
- Real-time status polling and updates
- Health monitoring implementation
- Auto-restart on failure
- Sentry activity metrics
- Domain-specific routine library (Pokemon TCG Pocket)

### Planned 📋
- Hot reload GUI buttons for templates/routines
- Variable inspector per bot instance
- Config editor GUI for routine parameters
- Enhanced logging and statistics
- Discord webhook notifications
- OCR integration for text recognition

## Quick Start

### Prerequisites

- **Windows 10/11** (required - Linux/macOS not supported)
- **MuMu Player 12** (Android emulator)
- **Pokemon TCG Pocket** APK installed in MuMu
- **Go 1.23+** (for building from source)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd PocketTCGoBot
```

2. Build the application:
```bash
go build -o bin/pocket-bot.exe ./cmd/bot
```

3. Configure `bin/Settings.ini` for your setup (see [Configuration](#configuration))

4. Run the bot:
```bash
cd bin
./pocket-bot.exe
```

### First Run

1. Launch the GUI
2. Go to **ADB Test** tab to verify MuMu instances are detected
3. Configure your settings in the **Config** tab
4. Add accounts to `bin/accounts/` directory
5. Start bot from **Dashboard** tab

## Project Structure

```
PocketTCGoBot/
├── cmd/
│   ├── bot/                  # Main GUI application
│   ├── import_accounts/      # Account XML import tool
│   ├── seed-database/        # Database seeding tool
│   └── test_*/               # Testing utilities
├── internal/
│   ├── bot/                  # Orchestrator, manager, bot lifecycle
│   ├── actions/              # 41 actions, routine engine, sentry system
│   ├── accountpool/          # Pool manager, unified pools, SQL filtering
│   ├── database/             # SQLite migrations, models, checkout API
│   ├── adb/                  # Android Debug Bridge controller
│   ├── cv/                   # Computer vision and image capture
│   ├── accounts/             # Account injection/extraction
│   ├── emulator/             # MuMu instance detection and management
│   ├── gui/                  # Fyne GUI tabs and wizards
│   ├── config/               # Configuration loader
│   ├── monitor/              # Error monitoring
│   ├── coordinator/          # Multi-bot coordination (legacy)
│   ├── ocr/                  # OCR (placeholder)
│   └── discord/              # Discord integration (placeholder)
├── pkg/
│   └── templates/            # Template registry and definitions
├── bin/                      # Runtime directory (build and run from here)
│   ├── templates/            # CV template PNG images (236+) + YAML definitions
│   ├── routines/             # YAML routine definitions (committed)
│   ├── pools/                # Account pool YAML definitions (committed)
│   ├── accounts/             # Account XML files (gitignored)
│   ├── *.db                  # SQLite databases (gitignored)
│   ├── *.exe                 # Built executables (gitignored)
│   └── Settings.ini          # Configuration file (gitignored, copy from example)
├── docs/
│   ├── ARCHITECTURE.md       # Complete system architecture
│   ├── ORCHESTRATION_ID_IMPLEMENTATION.md
│   └── [14+ other docs]      # Actions, sentries, routines, etc.
└── gui_mockups/              # Future interface designs
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for comprehensive architecture documentation.

## Configuration

Configuration is managed via `bin/Settings.ini`. Key sections:

### Emulator Settings
```ini
Columns = 3              # Number of emulator columns
rowGap = 0               # Gap between rows
folderPath = C:\Program Files\Netease\MuMuPlayer-12.0
```

### Pack Preferences
```ini
openMewtwo = true
openCharizard = true
openPikachu = true
minStars = 3             # Minimum stars to keep packs
```

### Account Management
```ini
deleteMethod = Create Bots
injectSortMethod = CreationDate
waitForEligibleAccounts = false
```

See [docs/SETUP.md](docs/SETUP.md) for complete configuration guide.

## Development

### Building

```bash
# Build to bin/ directory for testing
go build -o bin/pocket-bot.exe ./cmd/bot

# Build all internal packages (verification)
go build ./internal/...

# Run tests
go test ./...

# Run from bin/ directory (where assets are located)
cd bin && ./pocket-bot.exe

# Database migrations run automatically on first startup
# See internal/database/migrations.go for migration history
```

### Creating YAML Routines

Routines are YAML files in `bin/routines/`:

```yaml
routine_name: "My Routine"
description: "Does something useful"
tags: ["automation", "farming"]

config:
  - name: max_iterations
    type: int
    default: 10

steps:
  - action: SetVariable
    name: counter
    value: "0"

  - action: FindAndClickCenter
    template: MyButton
    timeout: 5000

  - action: ConditionalLoop
    condition:
      variable: counter
      operator: "<"
      value: "${max_iterations}"
    steps:
      - action: IncrementVariable
        name: counter
```

### Adding New Actions

1. Define action struct in [internal/actions/](internal/actions/)
2. Implement `Execute(ctx context.Context, bot BotInterface) error`
3. Register in `actionRegistry` map
4. Document in [docs/ACTIONS.md](docs/ACTIONS.md)

### Adding New Templates

1. Add PNG to `bin/templates/`
2. Create YAML definition in `bin/templates/definitions/`:
```yaml
- name: MyTemplate
  file: MyTemplate.png
  region:
    x1: 0
    y1: 0
    x2: 540
    y2: 960
  threshold: 0.8
  description: "Description of what this template matches"
```

### Creating Account Pools

Create YAML file in `bin/pools/`:

```yaml
pool_name: "My Pool"
description: "Accounts with specific criteria"

config:
  queries:
    - name: "High Shinedust"
      sql: "SELECT * FROM accounts WHERE total_shinedust > ?"
      parameters:
        - 50000

  sort_by:
    - column: total_shinedust
      order: DESC

  limit: 100
  auto_refresh:
    enabled: true
    interval: 60
```

## Contributing

This project is in early prototype stage. Contributions are welcome! Please:

1. Check existing issues or create one to discuss changes
2. Fork the repository
3. Create a feature branch
4. Make your changes with tests
5. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

### Development Priorities

See [ROADMAP.md](ROADMAP.md) for detailed development plan.

Current focus areas:
- Individual bot lifecycle controls (pause/resume/stop per instance)
- Real-time status polling and updates
- Health monitoring and auto-restart
- Domain-specific routine library (Pokemon TCG Pocket)
- GUI enhancements for pool and orchestration management

## Architecture Highlights

### Two-Tier Account Conflict Resolution

1. **Orchestration ID (UUID)** - Isolates execution contexts per bot group
   - Prevents stale execution records from affecting new runs
   - Tracked in `routine_executions` table with indexes

2. **Database Checkout Mutex** - Global source of truth for account injection
   - Atomic checkout operations prevent duplicate injections
   - 10-minute stale detection for crash recovery
   - Defer & retry logic for account conflicts

### Routine System

- **Build-Execute Pattern** - Routines compiled once, executed many times
- **Shared Registries** - Memory efficient for multi-instance coordination
- **Thread-Safe State** - Atomic operations with mutex protection
- **Sentry Supervision** - Parallel error monitoring with recovery actions
- **Variable Interpolation** - Runtime `${variable}` substitution

### Account Pool Architecture

- **Pool Definitions** - Shared YAML templates for account queries
- **Execution-Specific Pools** - Per-orchestration queue instances
- **SQL Filtering** - Complex queries with parameters and sorting
- **Progress Monitoring** - InitialAccountCount tracking for UI display

See [ARCHITECTURE.md](ARCHITECTURE.md) for complete system design.

## Known Issues

- Individual bot controls not yet in GUI (can launch/stop groups only)
- Real-time status updates require manual refresh
- Health monitoring stubbed but not implemented
- Auto-restart policy not implemented
- Sentry metrics not visible in GUI
- OCR engine placeholder only
- Discord webhooks placeholder only
- Domain-specific routines minimal (infrastructure ready)

## Deprecating AHK Version

This Go implementation aims to fully replace the AutoHotkey version located in `archived/`. Key improvements:

| Feature | AHK | Go |
|---------|-----|-----|
| Dependencies | AHK runtime | None (standalone .exe) |
| Performance | Moderate | High (native code) |
| Multi-instance | Limited | Native support |
| Maintainability | Script-based | Compiled, typed |
| Error Handling | Basic | Advanced patterns |
| Platform | Windows only | Windows only |
| CV Library | External | Pure Go (no dependencies) |

## License

[License to be determined]

## Support

- **Issues:** [GitHub Issues](link-to-issues)
- **Discussions:** [GitHub Discussions](link-to-discussions)
- **Discord:** [Coming soon]

## Acknowledgments

- Original AHK bot developers
- Pokemon TCG Pocket community
- Fyne GUI framework team
