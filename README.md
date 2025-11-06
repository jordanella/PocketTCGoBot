# PocketTCG Bot - Go Edition

A high-performance Pokemon TCG Pocket automation bot written in Go, designed to build upon the PTCGPB project written in AutoHotkey with improved reliability, efficiency, and maintainability.
This is built using https://github.com/kevnITG/PTCGPB as a model. kevinnnn is a brilliant developer and none of this would be possible without the project he maintains.

## Overview

PocketTCG Bot automates Wonder Pick farming and account management for Pokemon TCG Pocket running on MuMu Player emulator instances. Built with Go to eliminate dependency management issues while maintaining cross-platform compatibility and native performance.

**Status:** Early prototype with core functionality implemented

### Key Features

- **Multi-Instance Support** - Manage multiple MuMu Player emulator instances simultaneously
- **Account Management** - Inject, extract, and manage game accounts via XML files
- **Computer Vision** - Template-based image recognition with 236+ pre-defined templates
- **Wonder Pick Automation** - Automated Wonder Pick farming with configurable preferences
- **GUI Interface** - Cross-platform GUI built with Fyne framework
- **ADB Integration** - Direct Android Debug Bridge control for emulator management

### Why Go?

- **No Runtime Dependencies** - Single executable, no Python environment or package management
- **Native Performance** - Faster and more reliable than scripting languages
- **Cross-Platform** - Works on Windows, Linux, and macOS without modification
- **Pure Go CV** - No OpenCV or external CV libraries required
- **Easy Deployment** - Distribute a single binary to users

## Current Capabilities

### Implemented
- ✅ MuMu Player instance detection and management
- ✅ Window positioning and resizing with multi-monitor support
- ✅ Launch and kill Pokemon TCG Pocket APK
- ✅ Account injection/extraction via ADB
- ✅ XML-based account storage and loading
- ✅ Template-based image recognition
- ✅ Wonder Pick farming routine
- ✅ Configuration management via INI file
- ✅ Multi-tab GUI with real-time status
- ✅ Screen history tracking
- ✅ Action builder pattern for fluent scripting

### In Development
- 🚧 Bot execution flow routines
- 🚧 Delegation to multiple instances
- 🚧 Asynchronous error handling
- 🚧 Card recognition and rarity detection
- 🚧 Pack opening automation
- 🚧 Mission completion
- 🚧 OCR for text recognition (shinedust, card names)
- 🚧 Database integration for detailed logging
- 🚧 Error monitoring and recovery
- 🚧 Discord webhook notifications

### Planned
- 📋 Complete parity with AHK bot features
- 📋 Enhanced card detection and logging
- 📋 Discord integration for Wonder Pick groups
- 📋 Save-for-Trade (S4T) functionality
- 📋 Advanced statistics and reporting

## Quick Start

### Prerequisites

- **MuMu Player** (Android emulator)
- **Pokemon TCG Pocket** APK installed
- **Go 1.23+** (for building from source)
- **Windows** (primary platform, others untested)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd PocketTCGoBot
```

2. Build the GUI application:
```bash
go build -o pocket-bot-gui.exe ./cmd/bot-gui
```

3. Configure `bin/Settings.ini` for your setup (see [Configuration](#configuration))

4. Run the bot:
```bash
./pocket-bot-gui.exe
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
│   ├── bot/           # CLI version (deprecated)
│   └── bot-gui/       # GUI application (primary)
├── internal/
│   ├── bot/           # Core bot engine
│   ├── actions/       # Action library and primitives
│   ├── adb/           # Android Debug Bridge controller
│   ├── cv/            # Computer vision and image capture
│   ├── accounts/      # Account management
│   ├── emulator/      # MuMu emulator integration
│   ├── gui/           # Fyne GUI implementation
│   ├── config/        # Configuration loader
│   ├── monitor/       # Error monitoring
│   ├── coordinator/   # Multi-bot coordination
│   ├── ocr/           # OCR (placeholder)
│   └── discord/       # Discord integration (placeholder)
├── pkg/
│   └── templates/     # Template definitions
├── bin/
│   ├── accounts/      # Account XML files
│   ├── templates/     # Template PNG images
│   └── Settings.ini   # Configuration file
└── docs/              # Documentation
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed architecture documentation.

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
# Build GUI version
go build -o pocket-bot-gui.exe ./cmd/bot-gui

# Build CLI version (deprecated)
go build -o pocket-bot.exe ./cmd/bot

# Run tests
go test ./...
```

### Adding New Actions

Actions use a fluent builder pattern:

```go
// In internal/actions/library.go
func (l *Library) DoSomething() error {
    return l.Action().
        Click(100, 200).
        Sleep(1 * time.Second).
        FindAndClickCenter(templates.Button).
        Execute()
}
```

### Adding New Templates

1. Add PNG to `bin/templates/`
2. Define in `pkg/templates/templates.go`:
```go
var MyNewTemplate = Template{
    Name: "MyNewTemplate",
    Region: &Region{X1: 0, Y1: 0, X2: 540, Y2: 960},
    Threshold: 0.8,
}
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

Current focus areas:
- Completing action library implementations
- Refactoring template string references to use template definitions
- Card recognition and logging
- Database integration
- Error handling and recovery
- Test coverage

## Roadmap

### Phase 1: Core Parity (Current)
- Complete all AHK bot functionality in Go
- Stabilize multi-instance management
- Refine template matching
- Implement error recovery

### Phase 2: Enhanced Features
- Card recognition and rarity detection
- OCR for text extraction
- Database logging and statistics
- Advanced pack opening logic

### Phase 3: Community Features
- Discord webhook integration
- Wonder Pick group coordination
- Save-for-Trade automation
- Web-based dashboard

### Phase 4: Polish
- Comprehensive testing
- Performance optimization
- Documentation completion
- User-friendly installer

## Known Issues

- Some action library methods are stubs
- Error monitoring needs completion
- OCR not yet implemented
- Discord webhooks not implemented
- Template string references need refactoring to use definitions
- Windows-only testing (Linux/macOS untested)

## Deprecating AHK Version

This Go implementation aims to fully replace the AutoHotkey version located in `archived/`. Key improvements:

| Feature | AHK | Go |
|---------|-----|-----|
| Dependencies | AHK runtime | None (standalone) |
| Performance | Moderate | High |
| Multi-instance | Limited | Native support |
| Maintainability | Script-based | Compiled, typed |
| Error Handling | Basic | Advanced patterns |
| Cross-platform | Windows only | Windows/Linux/macOS |
| CV Library | External | Pure Go |

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
