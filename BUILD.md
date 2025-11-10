# Build Instructions

## Quick Start

### Building the Application

```bash
# Build to bin directory for testing with assets
go build -o bin/pocket-bot.exe ./cmd/bot

# Run from bin directory (where Settings.ini, templates, routines, pools are located)
cd bin
./pocket-bot.exe
```

### Building for Release

```bash
# Build optimized release binary
go build -ldflags="-s -w" -o release/pocket-bot.exe ./cmd/bot

# Note: Copy bin/ assets (templates, routines, pools, Settings.ini.example) to release directory
```

## Project Structure

### Source Code
- `cmd/bot/` - Main application entry point
- `internal/` - All internal packages (bot, actions, database, etc.)
- `pkg/` - Public packages (templates registry)

### Runtime Assets (in `bin/`)
- `Settings.ini` - Configuration file (gitignored, create from example)
- `templates/` - CV template PNG images (236+)
- `routines/` - YAML routine definitions
- `pools/` - Account pool YAML definitions
- `accounts/` - Account XML files (gitignored for privacy)
- `*.db` - SQLite database files (gitignored)

### Build Artifacts
- `bin/*.exe` - Compiled binaries for testing (gitignored)
- `release/` - Release builds (gitignored)

## Development Workflow

1. **Edit source code** in `cmd/`, `internal/`, or `pkg/`
2. **Build to bin/** for testing: `go build -o bin/pocket-bot.exe ./cmd/bot`
3. **Run from bin/** to access templates/routines/pools: `cd bin && ./pocket-bot.exe`
4. **Test with real assets** in the bin/ directory

**Note**: The bin/ directory serves dual purpose:
- Source assets (templates/, routines/, pools/) are committed to git
- Runtime files (*.exe, *.db, accounts/) are gitignored
- This keeps development workflow simple without needing to copy assets

## Asset Management

### Templates
- Add PNG files to `bin/templates/`
- Define in YAML at `bin/templates/definitions/`
- Templates are committed to git (exception in .gitignore)

### Routines
- Create YAML files in `bin/routines/`
- Routines are committed to git
- Use subdirectories for organization (e.g., `routines/farming/`, `routines/combat/`)

### Account Pools
- Create YAML files in `bin/pools/`
- Pool definitions are committed to git
- Actual account data is gitignored

### Accounts
- **NEVER commit account XML files** (gitignored for security)
- Store in `bin/accounts/`
- Use `cmd/import_accounts` tool to populate from backups

## Build Tags and Flags

### Standard Build
```bash
go build -o bin/pocket-bot.exe ./cmd/bot
```

### Optimized Release Build
```bash
go build -ldflags="-s -w" -o release/pocket-bot.exe ./cmd/bot
```

Flags explanation:
- `-s` - Omit symbol table
- `-w` - Omit DWARF debug information
- Results in ~30-50% smaller binary

### Windows-Only Build

This application is designed specifically for Windows with MuMu Player integration.

```bash
# Build for Windows (64-bit)
go build -o bin/pocket-bot.exe ./cmd/bot

# Optimized release build
go build -ldflags="-s -w" -o release/pocket-bot.exe ./cmd/bot
```

**Note**: Linux/macOS are not supported. The application relies on:
- Windows-specific MuMu Player emulator paths
- ADB integration with MuMu Player
- Windows window management APIs

## Testing

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/bot
go test ./internal/actions

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Database Migrations

Database migrations run automatically on first startup. The database is created at `bin/pocket_bot.db`.

To reset the database:
```bash
rm bin/*.db
# Migrations will run on next startup
```

See [internal/database/migrations.go](internal/database/migrations.go) for migration history.

## Common Issues

### "Templates not found"
- Make sure you're running the executable from the `bin/` directory
- Check that `bin/templates/` exists and contains PNG files

### "Routine not found"
- Ensure `bin/routines/` directory exists
- Check YAML files for syntax errors
- Use the GUI's routine browser to validate

### "Database error on startup"
- Delete `bin/*.db` and restart (migrations will recreate)
- Check file permissions on bin/ directory

### "Import error: internal/bot"
- Previously blocked by .gitignore
- Fixed in commit: chore: fix .gitignore and remove deprecated code
- If still occurring, run `go mod tidy`

## CI/CD Notes

When setting up CI/CD:
1. Build artifacts go to `dist/` or `release/` (gitignored)
2. Include `bin/templates/`, `bin/routines/`, `bin/pools/` in release package
3. Include `bin/Settings.ini.example` (without actual settings)
4. **Never** include `bin/accounts/` or `bin/*.db`
