# PocketTCG Bot - API Reference

Developer reference for the PocketTCG Bot's internal APIs and interfaces.

## Table of Contents

- [Bot Interface](#bot-interface)
- [Action Builder API](#action-builder-api)
- [Computer Vision API](#computer-vision-api)
- [ADB Controller API](#adb-controller-api)
- [Emulator Manager API](#emulator-manager-api)
- [Account Manager API](#account-manager-api)
- [Template Definitions](#template-definitions)

## Bot Interface

### BotInterface

Interface implemented by `bot.Bot` to provide services to actions.

```go
type BotInterface interface {
    GetADB() *adb.Controller
    GetCV() *cv.Service
    GetContext() context.Context
    GetEmulator() *emulator.MuMu
    GetMonitor() *monitor.ErrorMonitor
    GetConfig() *Config
    GetState() State
    GetScreenHistory() *ScreenHistory
}
```

**Usage:**
```go
func (ab *ActionBuilder) doSomething() error {
    adb := ab.bot.GetADB()
    cv := ab.bot.GetCV()
    ctx := ab.bot.GetContext()

    // Use services...
}
```

## Action Builder API

### ActionBuilder

Fluent API for building action sequences.

```go
type ActionBuilder struct {
    bot   BotInterface
    steps []ActionStep
}
```

### Basic Actions

#### Click
```go
func (ab *ActionBuilder) Click(x, y int) *ActionBuilder
```
Single tap at coordinates.

**Example:**
```go
l.Action().
    Click(270, 480).
    Execute()
```

#### Swipe
```go
func (ab *ActionBuilder) Swipe(x1, y1, x2, y2, durationMs int) *ActionBuilder
```
Swipe gesture from (x1, y1) to (x2, y2).

**Example:**
```go
l.Action().
    Swipe(270, 700, 270, 300, 200). // Swipe up
    Execute()
```

#### Sleep
```go
func (ab *ActionBuilder) Sleep(duration time.Duration) *ActionBuilder
```
Wait/delay.

**Example:**
```go
l.Action().
    Click(100, 200).
    Sleep(500 * time.Millisecond).
    Click(100, 300).
    Execute()
```

#### Key
```go
func (ab *ActionBuilder) Key(keycode int) *ActionBuilder
```
Send Android keycode.

**Common Keycodes:**
- `3` - HOME
- `4` - BACK
- `24` - VOLUME_UP
- `25` - VOLUME_DOWN

**Example:**
```go
l.Action().
    Key(4). // Press BACK
    Execute()
```

### Computer Vision Actions

#### FindAndClickCenter
```go
func (ab *ActionBuilder) FindAndClickCenter(template Template) *ActionBuilder
```
Find template and click its center.

**Example:**
```go
l.Action().
    FindAndClickCenter(templates.Button).
    Execute()
```

#### FindAndClickOffset
```go
func (ab *ActionBuilder) FindAndClickOffset(
    template Template,
    offsetX, offsetY int,
) *ActionBuilder
```
Find template and click with offset from center.

**Example:**
```go
l.Action().
    FindAndClickOffset(templates.Card, 0, 50). // Click 50px below center
    Execute()
```

#### WaitFor
```go
func (ab *ActionBuilder) WaitFor(
    template Template,
    maxRetries int,
) *ActionBuilder
```
Wait until template appears (with retries).

**Example:**
```go
l.Action().
    WaitFor(templates.Shop, 30). // Try 30 times
    Execute()
```

#### ClickIfFound
```go
func (ab *ActionBuilder) ClickIfFound(template Template) *ActionBuilder
```
Click template if found (don't error if not found).

**Example:**
```go
l.Action().
    ClickIfFound(templates.CloseButton). // Click only if present
    Execute()
```

### Loop Constructs

#### Until
```go
func (ab *ActionBuilder) Until(
    condition Template,
    maxRetries int,
    action func() error,
) *ActionBuilder
```
Repeat action until condition is met.

**Example:**
```go
l.Action().
    Until(templates.CardSelected, 45, func() error {
        return l.Action().
            Click(270, 480).
            Sleep(100 * time.Millisecond).
            Execute()
    }).
    Execute()
```

#### While
```go
func (ab *ActionBuilder) While(
    condition func() bool,
    action func() error,
) *ActionBuilder
```
Repeat action while condition is true.

**Example:**
```go
l.Action().
    While(func() bool {
        return !ab.bot.GetContext().Done()
    }, func() error {
        // Do work...
        return nil
    }).
    Execute()
```

### Execution

#### Execute
```go
func (ab *ActionBuilder) Execute() error
```
Execute all queued actions.

**Returns:** First error encountered, or nil if all succeed.

## Computer Vision API

### Service

```go
type Service struct {
    mu             sync.RWMutex
    capturer       Capturer
    templates      map[string]image.Image
    cache          *FrameCache
    titleBarHeight int
}
```

### Methods

#### FindTemplate
```go
func (s *Service) FindTemplate(
    frame image.Image,
    template Template,
) (found bool, x, y int, confidence float32)
```
Find template in frame.

**Parameters:**
- `frame` - Image to search in (nil to capture fresh frame)
- `template` - Template definition

**Returns:**
- `found` - True if match above threshold
- `x, y` - Center coordinates of match
- `confidence` - Match confidence (0.0-1.0)

**Example:**
```go
frame := cv.CaptureFrame()
found, x, y, conf := cv.FindTemplate(frame, templates.Button)
if found {
    log.Printf("Found at (%d, %d) with %.2f confidence", x, y, conf)
}
```

#### CaptureFrame
```go
func (s *Service) CaptureFrame() image.Image
```
Capture current screen (with caching).

**Example:**
```go
frame := cv.CaptureFrame()
// Use frame for multiple template matches
```

#### LoadTemplate
```go
func (s *Service) LoadTemplate(template Template) error
```
Load template PNG from disk.

**Example:**
```go
err := cv.LoadTemplate(templates.MyTemplate)
```

#### SetCacheDuration
```go
func (s *Service) SetCacheDuration(duration time.Duration)
```
Set frame cache duration.

**Example:**
```go
cv.SetCacheDuration(200 * time.Millisecond)
```

### Template

```go
type Template struct {
    Name      string
    Region    *Region
    Threshold float32
    Scale     float32
}

type Region struct {
    X1, Y1, X2, Y2 int
}
```

**Example:**
```go
var MyButton = Template{
    Name:      "MyButton",
    Region:    &Region{X1: 0, Y1: 500, X2: 540, Y2: 960},
    Threshold: 0.85,
}
```

## ADB Controller API

### Controller

```go
type Controller struct {
    mu          sync.Mutex
    port        int
    adbPath     string
    isConnected bool
}
```

### Methods

#### Connect
```go
func (c *Controller) Connect() error
```
Connect to ADB device.

**Example:**
```go
err := adb.Connect()
```

#### Shell
```go
func (c *Controller) Shell(command string) error
```
Execute shell command.

**Example:**
```go
adb.Shell("input tap 270 480")
adb.Shell("am force-stop jp.pokemon.pokemontcgp")
```

#### Push
```go
func (c *Controller) Push(localPath, remotePath string) error
```
Push file to device.

**Example:**
```go
adb.Push("account.xml", "/sdcard/deviceAccount.xml")
```

#### Pull
```go
func (c *Controller) Pull(remotePath, localPath string) error
```
Pull file from device.

**Example:**
```go
adb.Pull("/sdcard/deviceAccount.xml", "extracted_account.xml")
```

#### Tap
```go
func (c *Controller) Tap(x, y int) error
```
Convenience method for tap.

**Example:**
```go
adb.Tap(270, 480)
```

#### Swipe
```go
func (c *Controller) Swipe(x1, y1, x2, y2, durationMs int) error
```
Convenience method for swipe.

**Example:**
```go
adb.Swipe(270, 700, 270, 300, 200)
```

#### Launch
```go
func (c *Controller) Launch(packageName string) error
```
Launch app by package name.

**Example:**
```go
adb.Launch("jp.pokemon.pokemontcgp")
```

#### ForceStop
```go
func (c *Controller) ForceStop(packageName string) error
```
Force stop app.

**Example:**
```go
adb.ForceStop("jp.pokemon.pokemontcgp")
```

## Emulator Manager API

### Manager

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
```

### Methods

#### DiscoverInstances
```go
func (m *Manager) DiscoverInstances() ([]*Instance, error)
```
Find all MuMu instances.

**Example:**
```go
instances, err := manager.DiscoverInstances()
for _, inst := range instances {
    log.Printf("Instance %d at port %d", inst.Index, inst.MuMu.Port)
}
```

#### GetInstance
```go
func (m *Manager) GetInstance(index int) (*Instance, error)
```
Get specific instance.

**Example:**
```go
inst, err := manager.GetInstance(0)
```

### MuMu

```go
type MuMu struct {
    index      int
    port       int
    hwnd       uintptr
    folderPath string
    columns    int
    rowGap     int
    scale      Scale
    monitor    int
}
```

### Methods

#### PositionWindow
```go
func (m *MuMu) PositionWindow() error
```
Position window in grid layout.

**Example:**
```go
mumu.PositionWindow()
```

#### LaunchAPK
```go
func (m *MuMu) LaunchAPK(packageName string) error
```
Launch app on this instance.

**Example:**
```go
mumu.LaunchAPK("jp.pokemon.pokemontcgp")
```

#### KillAPK
```go
func (m *MuMu) KillAPK(packageName string) error
```
Kill app on this instance.

**Example:**
```go
mumu.KillAPK("jp.pokemon.pokemontcgp")
```

## Account Manager API

### Injector

```go
type Injector struct {
    adb *adb.Controller
}
```

### Methods

#### Inject
```go
func (i *Injector) Inject(account *Account) error
```
Inject account XML to device.

**Example:**
```go
account, _ := accounts.LoadAccount("bin/accounts/account_123.xml")
err := injector.Inject(account)
```

#### Extract
```go
func (i *Injector) Extract() (*Account, error)
```
Extract account from device.

**Example:**
```go
account, err := injector.Extract()
accounts.SaveAccount(account, "bin/accounts/extracted.xml")
```

#### Backup
```go
func (i *Injector) Backup() error
```
Backup current account before injection.

**Example:**
```go
injector.Backup()
injector.Inject(newAccount)
```

### Loader

```go
type Loader struct {
    accountsDir string
}
```

### Methods

#### LoadAllAccounts
```go
func (l *Loader) LoadAllAccounts() ([]*Account, error)
```
Load all accounts from directory.

**Example:**
```go
loader := accounts.NewLoader("bin/accounts")
accounts, err := loader.LoadAllAccounts()
```

#### LoadAccount
```go
func (l *Loader) LoadAccount(filename string) (*Account, error)
```
Load specific account.

**Example:**
```go
account, err := loader.LoadAccount("account_123.xml")
```

### Account

```go
type Account struct {
    UID       string
    Token     string
    CreatedAt time.Time
    Metadata  map[string]interface{}
}
```

## Template Definitions

### Accessing Templates

```go
import "jordanella.com/pocket-tcg-go/pkg/templates"

// Use pre-defined templates
templates.Button
templates.Card
templates.Shop
templates.WonderPick
```

### Common Templates

#### UI Navigation
- `templates.Home` - Home button
- `templates.Shop` - Shop button
- `templates.Menu` - Menu button
- `templates.Back` - Back button

#### Wonder Pick
- `templates.WonderPick` - Wonder Pick button
- `templates.Card` - Generic card
- `templates.CardSelected` - Selected card indicator
- `templates.Skip` - Skip button

#### Pack Opening
- `templates.Pack` - Pack icon
- `templates.OpenPack` - Open pack button
- `templates.PokeGoldPack` - Special gold pack

#### Cards
- `templates.1Star1` through `templates.1Star6` - 1-star borders
- `templates.3Diamond1` through `templates.3Diamond5` - 3-diamond borders
- `templates.Rainbow1` through `templates.Rainbow5` - Rainbow borders
- `templates.FullArt1` through `templates.FullArt5` - Full art borders

#### Language Detection
- `templates.99en` - English indicator
- `templates.99cn` - Chinese indicator
- `templates.99de` - German indicator

### Defining Custom Templates

```go
// In pkg/templates/templates.go
var MyNewTemplate = Template{
    Name:      "MyNewTemplate",           // Must match PNG filename
    Region:    &Region{X1: 0, Y1: 0, X2: 540, Y2: 960},
    Threshold: 0.8,                       // 0.0-1.0 (higher = stricter)
    Scale:     1.0,
}
```

Then create `bin/templates/MyNewTemplate.png`.

## Error Handling

### Standard Error Pattern

```go
if err := someOperation(); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Context Cancellation

```go
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue work
}
```

### Action Error Recovery

```go
func (l *Library) RobustAction() error {
    err := l.Action().
        FindAndClickCenter(templates.Button).
        Execute()

    if err != nil {
        // Retry logic
        return l.Action().
            Sleep(1 * time.Second).
            FindAndClickCenter(templates.Button).
            Execute()
    }

    return nil
}
```

## Thread Safety

### Mutex Usage

```go
type SafeService struct {
    mu   sync.RWMutex
    data map[string]interface{}
}

func (s *SafeService) Get(key string) interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data[key]
}

func (s *SafeService) Set(key string, val interface{}) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data[key] = val
}
```

### Context for Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Do work
        }
    }
}()
```

## Complete Example

```go
package main

import (
    "log"
    "time"
    "jordanella.com/pocket-tcg-go/internal/actions"
    "jordanella.com/pocket-tcg-go/internal/bot"
    "jordanella.com/pocket-tcg-go/pkg/templates"
)

func main() {
    // Create bot
    cfg := bot.DefaultConfig()
    b, err := bot.New(0, cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Initialize services
    if err := b.Initialize(); err != nil {
        log.Fatal(err)
    }
    defer b.Shutdown()

    // Create action library
    lib := actions.NewLibrary(b)
    b.SetActions(lib)

    // Execute custom action
    err = lib.Action().
        FindAndClickCenter(templates.Home).
        Sleep(500 * time.Millisecond).
        FindAndClickCenter(templates.Shop).
        WaitFor(templates.WonderPick, 30).
        FindAndClickCenter(templates.WonderPick).
        Execute()

    if err != nil {
        log.Printf("Action failed: %v", err)
    }
}
```

## Additional Resources

- [ARCHITECTURE.md](../ARCHITECTURE.md) - System design and patterns
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Development guidelines
- [SETUP.md](SETUP.md) - Configuration and setup
- [README.md](../README.md) - Project overview
