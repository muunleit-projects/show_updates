# Show Updates (Homebrew Update Helper)

[![CI](https://github.com/muunleit-projects/show_updates/actions/workflows/ci.yml/badge.svg)](https://github.com/muunleit-projects/show_updates/actions/workflows/ci.yml)

A lightweight macOS helper utility that checks for Homebrew updates and provides a friendly interface to upgrade outdated packages. The project offers a Go API, a graphical user interface (GUI) built with [Fyne](https://fyne.io/), and a simple command-line interface (CLI).

---

## Features

- **Automatic Check & List**: Automatically runs `brew update` and retrieves the list of outdated packages.
- **GUI Helper (`outdated_apps`)**:
  - Displays list of outdated packages in a clean window.
  - One-click button to trigger update: runs an AppleScript to launch MacOS Terminal and execute `brew upgrade -g -y`.
  - Auto-quits after a 15-second countdown if no updates are found (saving you from closing the window manually).
  - Error logging to `/tmp/outdated_apps.log` for easy troubleshooting.
- **CLI Utility (`upgrades_cli`)**: A simple console application that logs the current upgradable packages to the terminal.
- **Core Library (`checkupdates`)**: A robust, configurable package that handles:
  - A quick connectivity probe to `github.com:443` (3 s by default). It never blocks the check; if `brew update` fails, the probe result is added to the error to make network problems easier to spot.
  - Automatic fallback lookup of standard Homebrew paths (`/opt/homebrew/bin/brew` and `/usr/local/bin/brew`) if `PATH` is limited under GUI execution.
  - Custom command overrides.

---

## Directory Structure

```text
├── cmd/
│   ├── outdated_apps/     # Graphical user interface application (Fyne)
│   │   ├── main.go        # GUI logic and AppleScript launcher
│   │   └── Icon.png       # Application icon
│   └── upgrades_cli/      # Command-line interface application
│       └── main.go        # Basic CLI printing outdated packages
├── pkg/
│   └── checkupdates/      # Core API package
│       ├── checkupdates.go # Check logic, path-finding, and connectivity checks
│       └── *_test.go       # Tests for checker logic
├── LICENSE                # MIT License
├── README.md              # Project documentation
└── TODO                   # Project backlog/goals
```

---

## Prerequisites

- **macOS**
- **Homebrew** installed (`brew` executable available)
- **Go 1.25.0+**
- **Fyne dependencies** (standard compiler environment on macOS)

---

## Installation & Usage

### Running in Development

To run the GUI utility:

```bash
go run ./cmd/outdated_apps
```

To run the CLI utility:

```bash
go run ./cmd/upgrades_cli
```

### Building and Packaging

The GUI app is best packaged as a native macOS `.app` bundle using Fyne's packaging tool:

1. Install the Fyne packaging tool if you haven't:

   ```bash
   go install fyne.io/fyne/v2/cmd/fyne@latest
   ```

2. Package the app from the repository root (the GUI's `main` package lives in `cmd/outdated_apps`, so `--src` is required):
   ```bash
   fyne package -os darwin \
     -src ./cmd/outdated_apps \
     -icon Icon.png \
     -name "Outdated Apps" \
     -appID com.muunleit.showupdates \
     -appVersion "$(date +'%Y.%m.%d')"
   ```
   `-icon` is resolved relative to `-src`. Flag names differ slightly between versions of the fyne tool; check `fyne package --help` if one is rejected.

   This generates an `Outdated Apps.app` bundle, which you can move to your `/Applications` folder or set up to run automatically at login.

### Testing

```bash
go test ./...
```

The tests for `pkg/checkupdates` override the brew commands and run without Homebrew installed (e.g. on Linux CI).

---

## Configuration API (`pkg/checkupdates`)

For developers looking to integrate this checker into other Go projects, the `checkupdates` package exposes functional options:

```go
import (
    "time"
    "github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

// Create a custom checker
checker, err := checkupdates.NewChecker(
    checkupdates.WithConnectionTimeout(10 * time.Second),
    checkupdates.WithUpdate("/usr/local/bin/brew", "update"),
)
```
