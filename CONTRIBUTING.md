# Contributing to PocketTCGoBot

Thank you for your interest in contributing! This project is in early prototype stage and welcomes contributions of all kinds.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Architecture](#project-architecture)
- [Coding Standards](#coding-standards)
- [Contribution Workflow](#contribution-workflow)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Priority Areas](#priority-areas)

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Focus on what's best for the project and community
- Be patient with beginners

## Getting Started

### Prerequisites

- Go 1.23 or later
- Git
- MuMu Player emulator (for testing)
- Pokemon TCG Pocket APK
- Basic understanding of Go and Android automation

### Development Setup

1. **Fork and Clone**
```bash
git clone https://github.com/your-username/PocketTCGoBot.git
cd PocketTCGoBot
```

2. **Install Dependencies**
```bash
go mod download
```

3. **Verify Build**
```bash
go build ./cmd/bot-gui
```

4. **Run Tests**
```bash
go test ./...
```

5. **Set Up Test Environment**
- Install MuMu Player
- Configure `bin/Settings.ini`
- Add test account XMLs to `bin/accounts/`

## Project Architecture

### Key Principles

1. **Dependency Injection** - Use interfaces to avoid circular dependencies
2. **Builder Pattern** - Actions use fluent method chaining
3. **Thread Safety** - Use mutexes and proper synchronization
4. **Context-Based Cancellation** - Respect context for graceful shutdown
5. **Pure Go** - No external CV libraries (OpenCV, etc.)

### Package Organization

- **`cmd/`** - Entry points (executables)
- **`internal/`** - Private packages (not importable)
  - `bot/` - Core bot engine
  - `actions/` - Action library and primitives
  - `adb/` - Android Debug Bridge
  - `cv/` - Computer vision
  - `gui/` - GUI implementation
- **`pkg/`** - Public packages (reusable)
- **`bin/`** - Runtime files (config, accounts, templates)

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed documentation.

### Core Interfaces

```go
// BotInterface - Implemented by bot.Bot
type BotInterface interface {
    GetADB() *adb.Controller
    GetCV() *cv.Service
    GetContext() context.Context
    GetEmulator() *emulator.MuMu
    // ...
}
```

### Action Builder Pattern

```go
// Example action implementation
func (l *Library) NavigateToShop() error {
    return l.Action().
        FindAndClickCenter(templates.Home).
        Sleep(500 * time.Millisecond).
        FindAndClickCenter(templates.Shop).
        WaitFor(templates.WonderPick, 30).
        Execute()
}
```

## Coding Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` before committing
- Run `go vet` and address warnings
- Use meaningful variable names (no single letters except loop counters)
- Add comments for exported functions and types

### Naming Conventions

- **Packages:** lowercase, single word (`actions`, `cv`, not `action_library`)
- **Files:** lowercase, underscore-separated (`action_builder.go`, not `ActionBuilder.go`)
- **Interfaces:** noun or adjective + "er" (`BotInterface`, `Capturer`)
- **Constructors:** `New` or `NewWithOptions`

### Error Handling

```go
// Good - provide context
if err != nil {
    return fmt.Errorf("failed to inject account: %w", err)
}

// Bad - swallow errors
if err != nil {
    log.Println(err)
}
```

### Context Usage

Always respect context cancellation:

```go
func (b *Bot) DoWork(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Do work
        }
    }
}
```

### Thread Safety

Use mutexes for shared state:

```go
type Service struct {
    mu    sync.RWMutex
    cache map[string]interface{}
}

func (s *Service) Get(key string) interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.cache[key]
}
```

## Contribution Workflow

### 1. Choose or Create an Issue

- Check existing issues for something you want to work on
- For new features, create an issue first to discuss
- Comment on the issue to claim it

### 2. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-123
```

### 3. Make Your Changes

- Write clean, readable code
- Add comments for complex logic
- Follow existing code patterns
- Keep commits focused and atomic

### 4. Write Tests

- Add unit tests for new functionality
- Update existing tests if behavior changes
- Aim for meaningful test coverage (not just 100%)

### 5. Update Documentation

- Update README.md if user-facing changes
- Add/update code comments
- Update ARCHITECTURE.md for structural changes
- Add examples if introducing new patterns

### 6. Commit Your Changes

Use conventional commit messages:

```
feat: add OCR support for card name recognition
fix: resolve race condition in ADB controller
docs: update setup guide with MuMu 12 instructions
refactor: replace template strings with template definitions
test: add unit tests for account injection
```

### 7. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Create a pull request with:
- Clear description of changes
- Link to related issue(s)
- Screenshots/videos if UI changes
- Testing steps for reviewers

### 8. Address Review Feedback

- Be open to suggestions
- Ask questions if unclear
- Make requested changes promptly
- Keep discussion focused and respectful

## Testing Guidelines

### Unit Tests

```go
func TestAccountInjection(t *testing.T) {
    // Arrange
    injector := accounts.NewInjector(mockADB)
    account := &accounts.Account{ID: "test123"}

    // Act
    err := injector.Inject(account)

    // Assert
    if err != nil {
        t.Errorf("expected no error, got %v", err)
    }
}
```

### Integration Tests

- Test with real MuMu instances when possible
- Use mock interfaces for external dependencies
- Clean up resources in `defer` or `t.Cleanup()`

### Test Organization

- Place tests in same package as code (`_test.go` suffix)
- Use table-driven tests for multiple scenarios
- Use testify/assert for readable assertions

```go
func TestTemplateMatching(t *testing.T) {
    tests := []struct {
        name      string
        template  Template
        threshold float32
        want      bool
    }{
        {"exact match", templates.Button, 0.95, true},
        {"partial match", templates.Card, 0.7, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Match(tt.template, tt.threshold)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Documentation

### Code Comments

```go
// NewBot creates a new bot instance for the given emulator index.
// It initializes the ADB connection, CV service, and configuration.
// The bot must be initialized with Initialize() before running.
func NewBot(index int, cfg Config) (*Bot, error) {
    // ...
}
```

### README Updates

Update README.md when:
- Adding new user-facing features
- Changing installation/setup process
- Modifying configuration options
- Adding new dependencies

### Architecture Documentation

Update ARCHITECTURE.md when:
- Adding new packages
- Introducing new patterns
- Changing core interfaces
- Modifying data flow

## Priority Areas

### High Priority

1. **Template Refactoring**
   - Replace hardcoded template strings with `templates.*` definitions
   - Ensure all templates are defined in `pkg/templates/templates.go`

2. **Action Library Completion**
   - Implement stubbed methods in `internal/actions/library.go`
   - Add missing navigation methods
   - Complete pack opening logic

3. **Error Handling**
   - Implement error recovery handlers
   - Add health checking logic
   - Improve error context and logging

4. **Testing**
   - Add unit tests for core packages
   - Integration tests for action sequences
   - Mock implementations for testing

### Medium Priority

5. **Card Recognition**
   - Implement card border detection
   - Rarity classification
   - Card logging to database

6. **OCR Implementation**
   - Text extraction for shinedust
   - Card name recognition
   - Error message detection

7. **Database Integration**
   - Schema design
   - Migration system
   - Account metadata storage

### Lower Priority

8. **Discord Integration**
   - Webhook notifications
   - Wonder Pick group coordination

9. **Performance Optimization**
   - Template matching speed
   - Frame capture optimization
   - Memory usage reduction

10. **Cross-Platform Testing**
    - Linux testing
    - macOS testing
    - CI/CD setup

## Specific Contribution Ideas

### For Beginners

- Add missing comments to exported functions
- Write unit tests for utility functions
- Update documentation with examples
- Fix typos and improve error messages
- Add validation to configuration loading

### For Intermediate

- Implement stubbed action methods
- Add new template definitions
- Refactor template string references
- Implement card detection algorithms
- Add integration tests

### For Advanced

- Design and implement database schema
- Build OCR engine without external dependencies
- Optimize template matching performance
- Implement advanced error recovery
- Design multi-bot coordination system

## Getting Help

- **Questions:** Open a GitHub Discussion
- **Bugs:** Create an issue with reproduction steps
- **Features:** Open an issue to discuss before implementing
- **Code Review:** Request review from maintainers

## Recognition

Contributors will be recognized in:
- README.md acknowledgments section
- Release notes for significant contributions
- Git commit history

Thank you for contributing to PocketTCGoBot!
