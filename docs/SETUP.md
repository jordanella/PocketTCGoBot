# PocketTCG Bot - Setup Guide

Complete setup guide for installing, configuring, and running the PocketTCGoBot.

## Table of Contents

- [System Requirements](#system-requirements)
- [Installation](#installation)
- [MuMu Player Setup](#mumu-player-setup)
- [Configuration](#configuration)
- [Account Management](#account-management)
- [First Run](#first-run)
- [Troubleshooting](#troubleshooting)
- [Advanced Configuration](#advanced-configuration)

## System Requirements

### Hardware
- **CPU:** Intel/AMD x64 processor (8+ cores recommended)
- **RAM:** 16GB+ recommended for multiple instances
- **Storage:** 8GB per emulator instance
- **Display:** 1920x1080 or higher recommended

### Software
- **Operating System:** Windows 10/11 (primary), macOS (untested)
- **MuMu Player:** Version 12.0+ recommended
- **Pokemon TCG Pocket:** Latest version APK
- **ADB:** Included with MuMu Player
- **Go:** 1.23+ (only if building from source)

## Installation

### Option 1: Pre-built Binary (Recommended)

1. Download the latest release from [Releases](link-to-releases)
2. Extract `pocket-bot-gui.exe` to your desired folder
3. Download the `bin` folder (contains templates and sample config)
4. Place `bin` folder in the same directory as the executable

```
PocketTCGoBot/
├── pocket-bot-gui.exe
└── bin/
    ├── templates/
    ├── accounts/
    └── Settings.ini
```

### Option 2: Build from Source

1. **Install Go**
   - Download from [golang.org](https://golang.org/dl/)
   - Verify: `go version` (should show 1.23+)

2. **Clone Repository**
```bash
git clone <repository-url>
cd PocketTCGoBot
```

3. **Install Dependencies**
```bash
go mod download
```

4. **Build**
```bash
# Windows
go build -o pocket-bot-gui.exe ./cmd/bot-gui

# Linux/macOS
go build -o pocket-bot-gui ./cmd/bot-gui
```

5. **Verify Build**
```bash
./pocket-bot-gui.exe --help
```

## MuMu Player Setup

### 1. Install MuMu Player

1. Download MuMu Player from [official site](https://www.mumuplayer.com/)
2. Install to default location: `C:\Program Files\Netease\MuMuPlayer-12.0`
3. Launch MuMu Player to complete initial setup

### 2. Configure Emulator Instance

**Recommended Settings:**
- **Resolution:** 540x960 (portrait) or 1080x1920 (for Scale125)
- **DPI:** 240
- **Android Version:** Android 9 or later
- **Root Access:** Enabled (required for account injection)

**Steps:**
1. Open MuMu Multi-Instance Manager
2. Create new instance or modify existing
3. Click settings gear icon
4. Set resolution to 540x960 portrait
5. Enable root access
6. Save and restart instance

### 3. Install Pokemon TCG Pocket

**Option A: Google Play Store**
1. Launch emulator instance
2. Sign in to Google account
3. Open Play Store
4. Search "Pokemon TCG Pocket"
5. Install

**Option B: APK File**
1. Download Pokemon TCG Pocket APK
2. Drag APK file onto MuMu Player window
3. Wait for installation
4. Launch app to verify

### 4. Enable ADB Debugging

1. In Pokemon TCG Pocket, go to settings
2. Tap "About" 7 times to enable Developer Options
3. Go back to Settings
4. Enable "USB Debugging" in Developer Options

### 5. Verify ADB Connection

1. Open command prompt/terminal
2. Navigate to MuMu installation folder
```bash
cd "C:\Program Files\Netease\MuMuPlayer-12.0\shell"
```

3. Connect to emulator
```bash
adb connect 127.0.0.1:16384
# Port varies by instance: 16384, 16416, 16448, etc.
```

4. Verify connection
```bash
adb devices
# Should show: 127.0.0.1:16384	device
```

### 6. Multiple Instances Setup

**Port Mapping:**
- Instance 0: Port 16384
- Instance 1: Port 16416
- Instance 2: Port 16448
- Instance 3: Port 16480
- *(increments by 32)*

**Create Multiple Instances:**
1. Open MuMu Multi-Instance Manager
2. Click "Clone" or "Create" for each instance
3. Configure each with same settings
4. Start instances in order (0, 1, 2, ...)

**Window Positioning:**
The bot will automatically arrange windows based on `Settings.ini`:
```ini
Columns = 3          # 3 columns
rowGap = 0           # No gap between rows
```

## Configuration

### 1. Copy Example Configuration

```bash
# If no Settings.ini exists
cp bin/Settings.example.ini bin/Settings.ini
```

### 2. Edit Settings.ini

Open `bin/Settings.ini` in a text editor.

#### Basic Settings

```ini
[UserSettings]
# Emulator Configuration
Columns = 3                                          # Window grid columns
rowGap = 0                                           # Pixels between rows
SelectedMonitorIndex = 0                             # Monitor number (0 = primary)
folderPath = C:\Program Files\Netease\MuMuPlayer-12.0
defaultLanguage = en                                 # en, cn, de, es, fr, it, ja, ko, pt, th

# Deletion Method
deleteMethod = Create Bots                           # Create Bots, Inject 13P, Inject 96P, Inject Missions
```

#### Account & Injection Settings

```ini
# Injection Configuration
injectSortMethod = CreationDate                      # CreationDate, Random, Sequential
injectMinPacks = 10                                  # Minimum packs for injection
injectMaxPacks = 50                                  # Maximum packs for injection

# Account Waiting
waitForEligibleAccounts = false                      # Wait for eligible accounts
maxWaitHours = 24                                    # Max hours to wait
```

#### Pack Preferences

```ini
# Pack Opening (true = open, false = skip)
openMewtwo = true
openCharizard = true
openPikachu = true
openMew = true
openCelebi = true
openMythical1 = false
openMythical2 = false
# ... (see full list in Settings.ini)
```

#### Star Requirements

```ini
# Minimum stars to keep packs
minStars = 3                                         # Global minimum
minStarsShiny = 1                                    # Shiny packs minimum

# Per-pack star requirements
minStarsMewtwo = 3
minStarsCharizard = 3
minStarsPikachu = 3
# ... (customize per pack type)
```

#### Card Validation

```ini
# Special card checks
CheckShinyPackOnly = false                           # Only validate shiny packs
TrainerCheck = false                                 # Check for trainer cards
FullArtCheck = false                                 # Check for full art cards
RainbowCheck = false                                 # Check for rainbow cards
ShinyCheck = false                                   # Check for shiny cards
CrownCheck = false                                   # Check for crown cards
ImmersiveCheck = false                               # Check for immersive cards
```

#### Mission Settings

```ini
skipMissionsInjectMissions = false                   # Skip missions on inject
claimSpecialMissions = true                          # Auto-claim special missions
claimDailyMission = true                             # Auto-claim daily missions
```

#### Resource Management

```ini
spendHourGlass = false                               # Use hourglasses for Wonder Picks
openExtraPack = false                                # Open extra daily pack
```

#### Social Features

```ini
FriendID = YOUR_FRIEND_ID                            # Your friend code
checkWPthanks = true                                 # Check Wonder Pick thanks
showcaseEnabled = true                               # Enable showcase
```

#### Performance Tuning

```ini
# Timing (milliseconds)
spdMissionClaimDelay = 500
spdMissionClaimPress = 100
spdMissionReturnDelay = 1000
# ... (see full list in Settings.ini)

# Swipe Configuration
swipeSpeed = 200                                     # Swipe duration (ms)
slowMotion = false                                   # Slow motion mode (debugging)
waitTime = 1000                                      # Default wait between actions
```

#### Logging

```ini
verboseLogging = 2                                   # 0=off, 1=basic, 2=detailed, 3=debug
```

### 3. Validate Configuration

Run the bot with `--validate` flag (if implemented):
```bash
./pocket-bot-gui.exe --validate
```

Or manually check:
- MuMu folder path exists
- Pack preferences sum to at least 1 true value
- Star requirements are reasonable (1-5)

## Account Management

### Account XML Format

```xml
<?xml version="1.0" encoding="utf-8"?>
<map>
    <string name="deviceAccount">{"uid":"12345678","token":"..."}</string>
</map>
```

### Adding Accounts

#### Option 1: Extract from Emulator

1. Launch bot GUI
2. Go to "Accounts" tab
3. Click "Extract Account"
4. Select emulator instance
5. Account saved to `bin/accounts/account_12345678.xml`

#### Option 2: Manual Placement

1. Obtain account XML file
2. Copy to `bin/accounts/`
3. Naming convention: `account_<uid>.xml` or any `.xml` file

### Account Organization

```
bin/accounts/
├── active/
│   ├── account_001.xml
│   ├── account_002.xml
│   └── account_003.xml
├── completed/
│   └── account_old.xml
└── backups/
    └── account_001_backup.xml
```

Bot will load all `.xml` files from `bin/accounts/` directory (non-recursive by default).

### Account Metadata (Future)

Stored in database (when implemented):
- UID
- Creation date
- Last used
- Total runs
- Cards obtained
- Current status (active, banned, completed)

## First Run

### 1. Pre-flight Checklist

- [ ] MuMu Player installed and configured
- [ ] Pokemon TCG Pocket installed on emulator
- [ ] ADB connection verified
- [ ] Settings.ini configured
- [ ] At least one account XML in bin/accounts/
- [ ] Templates present in bin/templates/

### 2. Launch Bot

```bash
./pocket-bot-gui.exe
```

### 3. Initial Configuration

1. **Dashboard Tab**
   - Verify bot status shows "Initialized"
   - Check emulator instance count

2. **ADB Test Tab**
   - Click "Detect Instances"
   - Verify all MuMu instances are detected
   - Test ADB connection for each instance
   - Verify port numbers are correct

3. **Config Tab**
   - Review loaded configuration
   - Make any final adjustments
   - Save if modified

4. **Accounts Tab**
   - Verify accounts are loaded
   - Check account count
   - Review account metadata

### 4. Test Run

1. Select a single emulator instance
2. Click "Start Bot" (or similar button)
3. Watch for:
   - ADB connection established
   - App launched
   - Account injected
   - Template matching working
   - Actions executing

4. Monitor the "Logs" tab for any errors

### 5. Common First-Run Issues

**Issue: ADB Connection Failed**
- Check MuMu Player is running
- Verify correct port in Settings.ini
- Try manual ADB connect: `adb connect 127.0.0.1:16384`

**Issue: Template Not Found**
- Verify `bin/templates/` directory exists
- Check template PNG files are present (236+ files)
- Ensure window scaling matches template expectations

**Issue: Account Injection Failed**
- Verify root access enabled on emulator
- Check account XML format is correct
- Ensure Pokemon TCG Pocket is closed during injection

**Issue: Window Not Positioned**
- Check monitor index in Settings.ini
- Verify columns and rowGap settings
- May require admin privileges for window manipulation

## Troubleshooting

### ADB Issues

**Problem: Device offline**
```bash
adb kill-server
adb start-server
adb connect 127.0.0.1:16384
```

**Problem: Multiple ADB instances**
```bash
# Use MuMu's ADB exclusively
cd "C:\Program Files\Netease\MuMuPlayer-12.0\shell"
adb devices
```

**Problem: Permission denied**
- Enable root access in emulator settings
- Restart emulator after enabling root

### Template Matching Issues

**Problem: Templates not matching**
- Check emulator resolution (should be 540x960)
- Verify DPI is 240
- Adjust threshold in template definition (lower = more lenient)

**Problem: False positives**
- Increase threshold in template definition
- Add region constraints to templates
- Use more specific template images

### Performance Issues

**Problem: Bot running slowly**
- Reduce frame cache duration
- Increase action delays in Settings.ini
- Close unnecessary applications
- Reduce number of simultaneous instances

**Problem: High CPU usage**
- Enable frame caching (if disabled)
- Increase sleep durations between actions
- Reduce verboseLogging level

### Account Issues

**Problem: Account not injecting**
- Verify XML format is correct
- Check file permissions
- Ensure root access enabled
- Try manual injection via ADB:
```bash
adb push account.xml /sdcard/deviceAccount.xml
adb shell su -c 'cp /sdcard/deviceAccount.xml /data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml'
```

**Problem: Wrong account loaded**
- Clear app data before injection
- Force stop app: `adb shell am force-stop jp.pokemon.pokemontcgp`
- Re-inject and restart app

## Advanced Configuration

### Custom Template Threshold

Edit `pkg/templates/templates.go`:
```go
var MyTemplate = Template{
    Name:      "MyTemplate",
    Threshold: 0.75,  // Lower = more lenient (0.6-0.95)
}
```

### Custom Actions

Create custom action sequences in `internal/actions/library.go`:
```go
func (l *Library) MyCustomAction() error {
    return l.Action().
        Click(100, 200).
        Sleep(1 * time.Second).
        FindAndClickCenter(templates.Button).
        Execute()
}
```

### Multi-Monitor Setup

```ini
[UserSettings]
SelectedMonitorIndex = 1  # Secondary monitor
```

Monitor indices:
- 0: Primary monitor
- 1: Secondary monitor (left or right)
- 2: Tertiary monitor

### Database Configuration (Future)

```ini
[Database]
enabled = true
path = bin/bot_data.db
```

### Discord Webhook (Future)

```ini
[Discord]
webhookURL = https://discord.com/api/webhooks/...
notifyOnCard = true
notifyOnError = true
```

## Next Steps

After successful setup:
1. Review [CONTRIBUTING.md](../CONTRIBUTING.md) to understand codebase
2. Read [ARCHITECTURE.md](../ARCHITECTURE.md) for system design
3. Check [README.md](../README.md) for feature roadmap
4. Join Discord community (if available)
5. Report issues on GitHub

## Support

- **Issues:** [GitHub Issues](link)
- **Discussions:** [GitHub Discussions](link)
- **Discord:** [Coming soon]

Happy botting!
