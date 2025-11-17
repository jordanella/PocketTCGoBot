# App Management Actions

## Overview

These actions allow routines to launch and kill the Pokemon TCG Pocket application on Android devices.

## Actions

### LaunchApp

Launches the Pokemon TCG Pocket app (or a custom app if specified).

**YAML Syntax:**
```yaml
- launchapp:
```

**With Custom Package:**
```yaml
- launchapp:
    package: "com.example.app"
    activity: "com.example.app.MainActivity"
```

**Parameters:**
- `package` (optional): The package name to launch. Defaults to `jp.pokemon.pokemontcgp`
- `activity` (optional): The activity to start. Defaults to `jp.pokemon.pokemontcgp.startup.MainActivity`

**Example Usage:**
```yaml
name: "Start Game"
steps:
  - launchapp:
  - sleep: 5000  # Wait for app to load
```

### KillApp

Force-stops the Pokemon TCG Pocket app (or a custom app if specified).

**YAML Syntax:**
```yaml
- killapp:
```

**With Custom Package:**
```yaml
- killapp:
    package: "com.example.app"
```

**Parameters:**
- `package` (optional): The package name to stop. Defaults to `jp.pokemon.pokemontcgp`

**Example Usage:**
```yaml
name: "Restart Game"
steps:
  - killapp:
  - sleep: 2000  # Wait for app to fully stop
  - launchapp:
  - sleep: 5000  # Wait for app to load
```

## Common Patterns

### Restart App on Error

```yaml
name: "Handle Stuck State"
steps:
  - ifimagefound:
      template: "error_screen"
      then:
        - killapp:
        - sleep: 2000
        - launchapp:
        - sleep: 5000
```

### Fresh Start Routine

```yaml
name: "Fresh Start"
steps:
  # Ensure clean state
  - killapp:
  - sleep: 2000

  # Launch app
  - launchapp:
  - sleep: 5000

  # Wait for main menu
  - waitforimage:
      template: "main_menu"
      timeout: 30000
```

### Recovery Routine

```yaml
name: "Recovery"
steps:
  # Try to recover by restarting
  - killapp:
  - sleep: 3000
  - launchapp:
  - sleep: 8000

  # Navigate back to safe state
  - runroutine:
      name: "navigate_to_main_menu"
```

## Notes

- **Launch Delay**: Always add a sleep after `launchapp` to give the app time to start (5-10 seconds recommended)
- **Kill Delay**: Add a short sleep (2-3 seconds) after `killapp` before launching again to ensure the app fully stops
- **Error Handling**: These actions use ADB commands which may fail if the device is not connected
- **Multiple Apps**: You can manage different apps by specifying the `package` parameter
- **Activity Names**: The activity parameter for `launchapp` must be the full qualified class name

## ADB Commands Used

- `LaunchApp`: Uses `am start -n <package>/<activity>`
- `KillApp`: Uses `am force-stop <package>`

These are standard Android Debug Bridge commands that work on all Android devices.
