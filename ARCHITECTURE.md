# PocketTCGoBot - Architecture Documentation

This document provides a comprehensive overview of the PocketTCGoBot architecture, design patterns, and implementation details.

## Table of Contents

- [System Overview](#system-overview)
- [Core Principles](#core-principles)
- [Package Architecture](#package-architecture)
- [Data Flow](#data-flow)
- [Design Patterns](#design-patterns)
- [Threading Model](#threading-model)
- [Computer Vision Pipeline](#computer-vision-pipeline)
- [Action System](#action-system)
- [Future Architecture](#future-architecture)

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         GUI Layer                            │
│  (Fyne Framework - Multi-tab Interface)                     │
└───────────────────┬─────────────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────────────┐
│                    Bot Controller                            │
│  • Lifecycle Management  • State Machine  • Coordination    │
└───┬──────────┬──────────┬──────────┬──────────┬─────────────┘
    │          │          │          │          │
┌───▼────┐ ┌──▼─────┐ ┌──▼────┐ ┌──▼──────┐ ┌─▼────────┐
│ Action │ │   CV   │ │  ADB  │ │Emulator │ │ Accounts │
│ Engine │ │Service │ │Control│ │ Manager │ │ Manager  │
└────────┘ └────────┘ └───────┘ └─────────┘ └──────────┘
    │          │          │          │            │
    └──────────┴──────────┴──────────┴────────────┘
                         │
            ┌────────────▼────────────┐
            │  MuMu Player Instances  │
            │  (Android Emulators)    │
            └─────────────────────────┘
```

### Technology Stack

- **Language:** Go 1.23+
- **GUI Framework:** Fyne v2.7.0
- **Configuration:** INI file format (gopkg.in/ini.v1)
- **Image Processing:** Pure Go (no OpenCV)
- **Android Control:** ADB (Android Debug Bridge)
- **Emulator:** MuMu Player (primary target)

## Core Principles

### 1. Zero External Dependencies for CV
- No OpenCV, no C bindings, no system libraries
- Pure Go image processing using `image` and `image/png` packages
- Template matching implemented from scratch
- Maintains portability and easy deployment

### 2. Dependency Injection via Interfaces
- Avoids circular dependencies between packages
- Enables testing with mock implementations
- Clear separation of concerns
- Example: `BotInterface` allows actions to access bot services

### 3. Builder Pattern for Actions
- Fluent, chainable API for action sequences
- Deferred execution with `.Execute()`
- Easy to read and maintain action scripts
- Type-safe action composition

### 4. Thread-Safe by Design
- All shared state protected by mutexes
- Context-based cancellation throughout
- EventBus for cross-thread GUI updates
- No data races (verified with `go test -race`)

### 5. Multi-Instance First
- Designed from the ground up for multiple emulator instances
- Per-instance ADB connections
- Isolated CV services with shared templates
- Coordinated or independent operation

## Package Architecture

### Entry Points (`cmd/`)

#### `cmd/bot-gui/main.go` (Primary)
```go
func main() {
    app := app.New()
    window := app.NewWindow("PocketTCG Bot")

    cfg, _ := config.LoadFromINI("bin/Settings.ini", 1)
    controller := gui.NewController(cfg, app, window)

    window.SetContent(controller.GetContent())
    window.ShowAndRun()
}
```

**Responsibilities:**
- Initialize Fyne application
- Load configuration
- Create GUI controller
- Handle window lifecycle

#### `cmd/bot/main.go` (Deprecated)
- CLI version, kept for reference
- May be revived for headless server mode

### Core Packages (`internal/`)

#### `internal/bot/` - Bot Engine

**Key Files:**
- `bot.go` - Main Bot struct and lifecycle
- `config.go` - Configuration structure (280+ lines)
- `state.go` - State machine
- `screens.go` - Screen detection
- `cycle.go` - Main bot loop

**Bot Structure:**
```go
type Bot struct {
    mu            sync.RWMutex
    cfg           Config
    index         int

    // Services
    adb           *adb.Controller
    cv            *cv.Service
    emulator      *emulator.MuMu
    monitor       *monitor.ErrorMonitor

    // State
    state         State
    isPaused      bool
    screenHistory *ScreenHistory

    // Lifecycle
    ctx           context.Context
    cancel        context.CancelFunc

    // Actions (injected)
    actions       ActionLibrary
}
```

**Lifecycle:**
```go
bot.New()         // Create instance
bot.Initialize()  // Connect ADB, start CV
bot.SetActions()  // Inject action library
bot.Run()         // Start main loop
bot.Shutdown()    // Cleanup resources
```

**Design Notes:**
- Bot implements `BotInterface` for action library
- Screen history tracked in circular buffer (50 frames)
- Context-based shutdown with cleanup
- Mutex protects all state changes

#### `internal/actions/` - Action System

**Key Files:**
- `library.go` - High-level composed actions
- `builder.go` - ActionBuilder with fluent API
- `primitives.go` - Basic actions (Click, Sleep, Key)
- `cv_actions.go` - Vision-based actions
- `wonderpick.go` - Wonder Pick farming
- `loops.go` - Loop constructs

**Action Builder Pattern:**
```go
type ActionBuilder struct {
    bot    BotInterface
    steps  []ActionStep
}

type ActionStep struct {
    name    string
    execute func() error
}

// Fluent API
func (ab *ActionBuilder) Click(x, y int) *ActionBuilder {
    ab.steps = append(ab.steps, ActionStep{
        name: fmt.Sprintf("Click(%d, %d)", x, y),
        execute: func() error {
            return ab.bot.GetADB().Shell(fmt.Sprintf(
                "input tap %d %d", x, y,
            ))
        },
    })
    return ab
}

func (ab *ActionBuilder) Execute() error {
    for _, step := range ab.steps {
        if err := step.execute(); err != nil {
            return fmt.Errorf("%s failed: %w", step.name, err)
        }
    }
    return nil
}
```

**Usage Example:**
```go
func (l *Library) NavigateToShop() error {
    return l.Action().
        FindAndClickCenter(templates.Home).
        Sleep(500 * time.Millisecond).
        FindAndClickCenter(templates.Shop).
        WaitFor(templates.WonderPick, 30).
        Execute()
}
```

**Loop Constructs:**
```go
// Retry until condition met
l.Action().
    Until(templates.CardSelected, 45, func() error {
        return l.Action().
            Click(270, 480).
            Sleep(100 * time.Millisecond).
            Execute()
    }).
    Execute()
```

#### `internal/cv/` - Computer Vision

**Key Files:**
- `service.go` - CV service with caching
- `capture.go` - Frame capture interface
- `capture_windows.go` - Windows implementation
- `templates.go` - Template metadata and loading
- `matching.go` - Template matching algorithm
- `card_detection.go` - Card rarity detection

**CV Service:**
```go
type Service struct {
    mu             sync.RWMutex
    capturer       Capturer
    templates      map[string]image.Image
    cache          *FrameCache
    titleBarHeight int
}

type FrameCache struct {
    frame      image.Image
    timestamp  time.Time
    maxAge     time.Duration  // Default: 100ms
}
```

**Template Matching Algorithm:**
1. Load template PNG from `bin/templates/`
2. If template has region, crop search area
3. Slide template across image
4. Calculate normalized cross-correlation (NCC) at each position
5. Return best match if above threshold

**Matching Function:**
```go
func (s *Service) FindTemplate(
    frame image.Image,
    template Template,
) (found bool, x, y int, confidence float32) {
    // Load template image
    templateImg := s.templates[template.Name]

    // Apply region if specified
    if template.Region != nil {
        frame = cropRegion(frame, template.Region)
    }

    // Slide and match
    bestMatch, bestX, bestY := slideMatch(frame, templateImg)

    if bestMatch >= template.Threshold {
        return true, bestX, bestY, bestMatch
    }
    return false, 0, 0, 0
}
```

**Frame Caching:**
- Captures are expensive (BitBlt, conversion)
- Cache valid for 100ms by default
- Configurable cache duration
- Thread-safe with mutex

**Card Detection:**
```go
// Border types for rarity detection
const (
    BorderNormal    = "normal"
    Border1Star     = "1star"
    Border3Diamond  = "3diamond"
    Border4Diamond  = "4diamond"
    // ... etc
)

// Special card detection
func (s *Service) DetectSpecialCard(frame image.Image) CardType {
    // Check for Rainbow, FullArt, Crown, Immersive, Shiny
    // Uses border color analysis and template matching
}
```

#### `internal/adb/` - Android Debug Bridge

**Key Files:**
- `controller.go` - ADB connection and commands
- `finder.go` - Find ADB executable
- `commands.go` - Common operations

**ADB Controller:**
```go
type Controller struct {
    mu          sync.Mutex
    port        int
    adbPath     string
    isConnected bool

    // Persistent shell for performance
    shellConn   *exec.Cmd
}

// Connection
func (c *Controller) Connect() error {
    cmd := exec.Command(c.adbPath, "connect",
                        fmt.Sprintf("127.0.0.1:%d", c.port))
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("connect failed: %w", err)
    }
    c.isConnected = true
    return nil
}

// Execute shell command
func (c *Controller) Shell(command string) error {
    cmd := exec.Command(c.adbPath, "-s",
                        fmt.Sprintf("127.0.0.1:%d", c.port),
                        "shell", command)
    return cmd.Run()
}

// File operations
func (c *Controller) Push(local, remote string) error {
    cmd := exec.Command(c.adbPath, "-s",
                        fmt.Sprintf("127.0.0.1:%d", c.port),
                        "push", local, remote)
    return cmd.Run()
}
```

**Design Notes:**
- Persistent shell connection for speed
- Per-instance ADB via port forwarding
- Automatic reconnection on failure
- Thread-safe command execution

#### `internal/emulator/` - MuMu Emulator Management

**Key Files:**
- `manager.go` - Multi-instance manager
- `mumu.go` - MuMu-specific operations

**Emulator Manager:**
```go
type Manager struct {
    mu        sync.RWMutex
    instances map[int]*Instance
    mumuPath  string
}

type Instance struct {
    Index       int
    MuMu        *MuMu
    ADB         *adb.Controller
    IsConnected bool
}

// Discover instances
func (m *Manager) DiscoverInstances() ([]*Instance, error) {
    // Read MuMu config to find instances
    // Create Instance for each
    // Connect ADB to each port
}
```

**MuMu Controller:**
```go
type MuMu struct {
    index      int
    hwnd       uintptr  // Windows HWND
    folderPath string

    // Window positioning
    columns    int
    rowGap     int
    scale      Scale
    monitor    int
}

// Window operations
func (m *MuMu) PositionWindow() error {
    // Calculate grid position
    // Move and resize window via Win32 API
}

func (m *MuMu) LaunchAPK(packageName string) error {
    // Use ADB to launch app
}

func (m *MuMu) KillAPK(packageName string) error {
    // Force stop app via ADB
}
```

**Window Positioning Algorithm:**
```
Column: index % columns
Row:    index / columns

X = Column * WindowWidth
Y = Row * (WindowHeight + rowGap) + MonitorOffsetY
```

#### `internal/gui/` - GUI Implementation

**Key Files:**
- `controller.go` - Main GUI controller
- `dashboard.go` - Dashboard tab
- `config.go` - Configuration tab
- `accounts.go` - Account management tab
- `controls.go` - Bot controls
- `logs.go` - Logging display
- `eventbus.go` - Thread-safe events

**GUI Controller:**
```go
type Controller struct {
    mu       sync.RWMutex
    app      fyne.App
    window   fyne.Window
    cfg      *bot.Config
    bot      *bot.Bot
    eventBus *EventBus

    // Tabs
    tabs     *container.AppTabs
}

// Tab creation
func (c *Controller) GetContent() fyne.CanvasObject {
    c.tabs = container.NewAppTabs(
        container.NewTabItem("Dashboard", c.createDashboard()),
        container.NewTabItem("Config", c.createConfig()),
        container.NewTabItem("Accounts", c.createAccounts()),
        container.NewTabItem("Controls", c.createControls()),
        container.NewTabItem("Logs", c.createLogs()),
        container.NewTabItem("ADB Test", c.createADBTest()),
    )
    return c.tabs
}
```

**EventBus Pattern:**
```go
type EventBus struct {
    mu        sync.RWMutex
    listeners map[string][]EventListener
    mainQueue chan Event
}

type Event struct {
    Type string
    Data interface{}
}

// Publish from any thread
func (eb *EventBus) Publish(eventType string, data interface{}) {
    eb.mainQueue <- Event{Type: eventType, Data: data}
}

// Process on main thread
func (eb *EventBus) ProcessEvents() {
    for event := range eb.mainQueue {
        eb.dispatch(event)
    }
}
```

**Why EventBus?**
- Fyne requires UI updates on main thread
- Bot runs on background goroutine
- EventBus bridges the gap safely

#### `internal/accounts/` - Account Management

**Account XML Structure:**
```xml
<deviceAccount>
    <uid>12345678</uid>
    <token>abcdef...</token>
    <createdAt>2024-01-01T00:00:00Z</createdAt>
    <!-- Additional metadata -->
</deviceAccount>
```

**Injection Flow:**
```
1. Push XML to /sdcard/deviceAccount.xml (ADB push)
2. Copy to /data/data/jp.pokemon.pokemontcgp/shared_prefs/
   deviceAccount:.xml (ADB shell su)
3. Verify file exists
4. Restart app to load account
```

**Extraction Flow:**
```
1. Copy from /data/data/.../deviceAccount:.xml to /sdcard/
2. Pull to local filesystem (ADB pull)
3. Parse XML and save to bin/accounts/
```

### Public Packages (`pkg/`)

#### `pkg/templates/`

**Template Definition:**
```go
type Template struct {
    Name      string
    Region    *Region
    Threshold float32
    Scale     float32  // For multi-resolution support
}

type Region struct {
    X1, Y1, X2, Y2 int
}

// Example definitions
var (
    Button = Template{
        Name:      "Button",
        Threshold: 0.85,
    }

    Card = Template{
        Name:   "Card",
        Region: &Region{X1: 160, Y1: 330, X2: 200, Y2: 370},
        Threshold: 0.9,
    }
)
```

**Usage:**
```go
import "jordanella.com/pocket-tcg-go/pkg/templates"

found, x, y := cvService.FindTemplate(frame, templates.Button)
if found {
    // Click button
}
```

## Data Flow

### Bot Execution Flow

```
User Clicks "Start Bot"
  │
  ├─> GUI Controller.StartBot()
  │     │
  │     ├─> bot.Initialize()
  │     │     ├─> Connect ADB
  │     │     ├─> Start CV Service
  │     │     ├─> Load Templates
  │     │     └─> Create Monitor
  │     │
  │     └─> bot.Run() (goroutine)
  │           │
  │           └─> Main Loop:
  │                 ├─> Capture Screen
  │                 ├─> Detect Current Screen
  │                 ├─> Execute Action for Screen
  │                 ├─> Update State
  │                 ├─> Publish Status (EventBus)
  │                 └─> Check Context (repeat or exit)
  │
  └─> EventBus Updates GUI
        └─> Dashboard shows status
```

### Action Execution Flow

```
Action Library Method Called
  │
  ├─> Create ActionBuilder
  │
  ├─> Chain Action Steps
  │     ├─> Click(x, y)
  │     ├─> Sleep(duration)
  │     ├─> FindAndClickCenter(template)
  │     └─> WaitFor(template, retries)
  │
  ├─> Execute() called
  │     │
  │     └─> For each step:
  │           ├─> Execute step function
  │           ├─> Check for errors
  │           ├─> Log step completion
  │           └─> Continue or abort
  │
  └─> Return result to bot
```

### Computer Vision Flow

```
CV Service.FindTemplate(template)
  │
  ├─> Check cache (< 100ms old?)
  │     ├─> Yes: Use cached frame
  │     └─> No:  Capture new frame
  │               ├─> Windows BitBlt
  │               ├─> Convert to image.Image
  │               └─> Cache frame
  │
  ├─> Load template PNG (or use cached)
  │
  ├─> Apply region crop if specified
  │
  ├─> Template matching algorithm
  │     ├─> Slide template across image
  │     ├─> Calculate NCC at each position
  │     └─> Track best match
  │
  └─> Return: found, x, y, confidence
```

## Design Patterns

### 1. Builder Pattern (Actions)

**Problem:** Complex action sequences are hard to read and compose

**Solution:** Fluent builder API with method chaining

```go
// Before (hypothetical)
action := NewClickAction(100, 200)
action.Execute()
sleep := NewSleepAction(1 * time.Second)
sleep.Execute()
find := NewFindAction(template)
find.Execute()

// After (actual)
l.Action().
    Click(100, 200).
    Sleep(1 * time.Second).
    FindAndClickCenter(template).
    Execute()
```

### 2. Dependency Injection (Bot Interface)

**Problem:** Circular dependency between `bot` and `actions`

**Solution:** Interface-based dependency injection

```go
// BotInterface in actions package
type BotInterface interface {
    GetADB() *adb.Controller
    GetCV() *cv.Service
    GetContext() context.Context
}

// Bot implements interface
type Bot struct { ... }
func (b *Bot) GetADB() *adb.Controller { return b.adb }

// Inject after creation
bot := bot.New(...)
actions := actions.NewLibrary(bot)  // Pass as BotInterface
bot.SetActions(actions)
```

### 3. Service Locator (Bot Services)

**Problem:** Actions need access to multiple services

**Solution:** Bot acts as service locator via getter methods

```go
// In action
adb := ab.bot.GetADB()
cv := ab.bot.GetCV()
ctx := ab.bot.GetContext()
```

### 4. Observer Pattern (EventBus)

**Problem:** GUI needs updates from background bot thread

**Solution:** Event bus with publish/subscribe

```go
// Bot publishes
eventBus.Publish("bot.status", StatusUpdate{State: "Running"})

// GUI subscribes
eventBus.Subscribe("bot.status", func(data interface{}) {
    status := data.(StatusUpdate)
    updateDashboard(status)
})
```

### 5. Template Method (Bot Lifecycle)

**Problem:** Standard lifecycle with customizable behavior

**Solution:** Template method pattern with hooks

```go
func (b *Bot) Run() error {
    b.onStart()      // Hook

    for !b.shouldStop() {
        b.cycle()    // Template method
    }

    b.onStop()       // Hook
}
```

### 6. Strategy Pattern (Screen Actions)

**Problem:** Different actions for different screens

**Solution:** Screen-to-action mapping

```go
var screenActions = map[ScreenType]func(*Bot) error{
    ScreenShop:       (*Bot).handleShopScreen,
    ScreenWonderPick: (*Bot).handleWonderPickScreen,
    ScreenPacks:      (*Bot).handlePacksScreen,
}

func (b *Bot) executeScreenAction(screen ScreenType) error {
    if action, ok := screenActions[screen]; ok {
        return action(b)
    }
    return ErrUnknownScreen
}
```

## Threading Model

### Goroutine Structure

```
Main Thread (GUI)
  │
  ├─> Fyne Event Loop
  ├─> EventBus Processor
  └─> User Input Handlers

Background Goroutine (Bot)
  │
  ├─> Bot.Run() main loop
  │     ├─> Screen capture
  │     ├─> Action execution
  │     └─> State updates
  │
  └─> Monitor goroutines (future)
        ├─> Health checker
        └─> Error monitor
```

### Synchronization Primitives

**Mutexes:**
```go
type Bot struct {
    mu sync.RWMutex  // Protects all state
    // ...
}

// Read access
func (b *Bot) GetState() State {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return b.state
}

// Write access
func (b *Bot) SetState(s State) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.state = s
}
```

**Context-based Cancellation:**
```go
func (b *Bot) Run() error {
    for {
        select {
        case <-b.ctx.Done():
            return b.ctx.Err()
        default:
            // Do work
        }
    }
}

func (b *Bot) Shutdown() {
    b.cancel()  // Triggers context cancellation
    // Cleanup...
}
```

**EventBus Channel:**
```go
type EventBus struct {
    mainQueue chan Event  // Buffered channel
}

// Safe from any goroutine
func (eb *EventBus) Publish(typ string, data interface{}) {
    eb.mainQueue <- Event{Type: typ, Data: data}
}
```

## Computer Vision Pipeline

### Template Matching Algorithm

**Normalized Cross-Correlation (NCC):**

```
For each position (x, y) in search area:
    1. Extract window of size (tw, th) from frame
    2. Compute mean of template and window
    3. Subtract means (normalize)
    4. Compute dot product
    5. Divide by magnitudes (normalize)
    6. Result is correlation coefficient [-1, 1]

Best match = position with highest coefficient
```

**Implementation:**
```go
func normalizedCrossCorrelation(
    frame, template image.Image,
    x, y int,
) float32 {
    // Extract window
    window := extractWindow(frame, x, y, tw, th)

    // Compute means
    meanT := computeMean(template)
    meanW := computeMean(window)

    // Normalize and correlate
    var dotProduct, magT, magW float64
    for i := 0; i < tw*th; i++ {
        t := float64(templatePixels[i]) - meanT
        w := float64(windowPixels[i]) - meanW

        dotProduct += t * w
        magT += t * t
        magW += w * w
    }

    return float32(dotProduct / (math.Sqrt(magT) * math.Sqrt(magW)))
}
```

### Multi-Scale Matching (Future)

For different emulator scales:
```go
func (s *Service) FindTemplateMultiScale(
    frame image.Image,
    template Template,
) (found bool, x, y int, scale float32) {
    scales := []float32{0.8, 0.9, 1.0, 1.1, 1.2}

    for _, scale := range scales {
        scaled := resizeTemplate(template, scale)
        if found, x, y := s.FindTemplate(frame, scaled); found {
            return true, x, y, scale
        }
    }
    return false, 0, 0, 0
}
```

## Action System

### Action Primitives

**Basic Actions:**
- `Click(x, y)` - Single tap at coordinates
- `Swipe(x1, y1, x2, y2, duration)` - Swipe gesture
- `Sleep(duration)` - Wait/delay
- `Key(keycode)` - Send key press
- `Text(string)` - Input text

**CV Actions:**
- `FindAndClickCenter(template)` - Find template, click center
- `WaitFor(template, retries)` - Wait until template appears
- `ClickIfFound(template)` - Conditional click
- `FindAndClickOffset(template, dx, dy)` - Click with offset

**Compound Actions:**
- `Until(condition, retries, action)` - Repeat until condition
- `While(condition, action)` - Repeat while condition
- `Sequence(actions...)` - Execute in order

### Action Composition

**Example: Wonder Pick Flow**
```go
func (l *Library) DoWonderPick() error {
    // Navigate to Wonder Pick
    if err := l.NavigateToWonderPick(); err != nil {
        return err
    }

    // Select card
    if err := l.SelectWonderPickCard(); err != nil {
        return err
    }

    // Wait for result
    if err := l.WaitForResult(); err != nil {
        return err
    }

    // Return to shop
    return l.NavigateToShop()
}

func (l *Library) SelectWonderPickCard() error {
    return l.Action().
        Until(templates.CardSelected, 45, func() error {
            return l.Action().
                Click(270, 480).  // Click center card
                Sleep(100 * time.Millisecond).
                Execute()
        }).
        Execute()
}
```

## Future Architecture

### Database Layer (Planned)

**Schema:**
```
accounts
  - id (primary key)
  - uid (game account ID)
  - created_at
  - last_used
  - total_runs
  - total_cards

cards
  - id
  - account_id (foreign key)
  - name
  - rarity
  - pack_type
  - obtained_at

runs
  - id
  - account_id
  - started_at
  - ended_at
  - wonder_picks
  - packs_opened
  - errors

statistics
  - account_id
  - date
  - metric_type
  - value
```

**Access Layer:**
```go
type Database struct {
    db *sql.DB
}

func (d *Database) SaveCard(card *Card) error
func (d *Database) GetAccountStats(accountID string) (*Stats, error)
func (d *Database) RecordRun(run *Run) error
```

### OCR Engine (Planned)

**Tesseract-free Approach:**
```go
type OCREngine struct {
    charTemplates map[rune]image.Image
}

func (ocr *OCREngine) RecognizeText(
    region image.Image,
) (string, error) {
    // Character segmentation
    chars := segmentCharacters(region)

    // Match each character
    var result strings.Builder
    for _, char := range chars {
        recognized := ocr.matchCharacter(char)
        result.WriteRune(recognized)
    }

    return result.String(), nil
}
```

**Use Cases:**
- Read shinedust count
- Extract card names
- Detect error messages
- Read mission text

### Discord Integration (Planned)

```go
type DiscordWebhook struct {
    url string
}

func (dw *DiscordWebhook) NotifyCard(card *Card) error {
    payload := map[string]interface{}{
        "embeds": []map[string]interface{}{
            {
                "title":       "Card Obtained!",
                "description": card.Name,
                "color":       colorForRarity(card.Rarity),
                "thumbnail":   card.ImageURL,
            },
        },
    }
    return dw.post(payload)
}
```

### Multi-Bot Coordinator (Partial)

**Current State:** Stubbed in `internal/coordinator/`

**Future Design:**
```go
type Coordinator struct {
    bots    map[int]*bot.Bot
    accounts *AccountPool
}

func (c *Coordinator) StartAllBots() error {
    for _, bot := range c.bots {
        account := c.accounts.GetAvailable()
        bot.SetAccount(account)
        go bot.Run()
    }
}

func (c *Coordinator) BalanceLoad() {
    // Distribute accounts evenly
    // Handle failures
    // Coordinate Wonder Pick groups
}
```

## Summary

The PocketTCG Bot architecture is designed for:
- **Reliability** - Thread-safe, error-handled, recoverable
- **Performance** - Cached CV, persistent ADB, efficient matching
- **Maintainability** - Clear separation, dependency injection, testable
- **Extensibility** - Plugin actions, configurable behavior, modular design
- **Portability** - Pure Go, no external CV dependencies

Key architectural decisions:
1. Fyne for cross-platform GUI
2. Pure Go CV for zero dependencies
3. Builder pattern for readable action scripts
4. Dependency injection to avoid circular dependencies
5. EventBus for thread-safe GUI updates
6. Multi-instance design from day one

This architecture positions the project for growth from prototype to production-ready bot while maintaining code quality and ease of collaboration.
