# Show Updates (Homebrew Update Helper)

[![CI](https://github.com/muunleit-projects/show_updates/actions/workflows/ci.yml/badge.svg)](https://github.com/muunleit-projects/show_updates/actions/workflows/ci.yml)

A lightweight macOS helper utility that checks for Homebrew updates and provides a friendly interface to upgrade outdated packages. The project offers a Go API, a graphical user interface (GUI) built with [Fyne](https://fyne.io/), and a simple command-line interface (CLI).

---

## Features

- **Automatic Check & List**: Automatically runs `brew update` and retrieves the list of outdated packages.
- **GUI Helper (`outdated_apps`)**:
  - Lists outdated formulae and casks with installed and new version, e.g. `git  2.44.0 → 2.45.1`; casks are marked "(app)".
  - Pinned packages are highlighted, with a note that `brew upgrade` will skip them.
  - One-click button to trigger update: runs an AppleScript to launch MacOS Terminal and execute `brew upgrade -g -y`.
  - Auto-quits after a 15-second countdown if no updates are found (saving you from closing the window manually).
  - Errors are shown in the window and logged to `~/Library/Logs/outdated_apps.log` (also visible in Console.app).
  - If Terminal cannot be opened (e.g. missing automation permission), the reason is shown instead of the app silently quitting.
- **CLI Utility (`upgrades_cli`)**: Prints one outdated package per line (or JSON with `-json`). With `-exit-code` it exits with status 2 when updates are available, so it works in scripts.
- **Core Library (`checkupdates`)**: A robust, configurable package that handles:
  - A quick connectivity probe to `github.com:443` (3 s by default). It never blocks the check; if `brew update` fails, the probe result is added to the error to make network problems easier to spot.
  - Automatic fallback lookup of standard Homebrew paths (`/opt/homebrew/bin/brew` and `/usr/local/bin/brew`) if `PATH` is limited under GUI execution.
  - Structured results parsed from `brew outdated --json=v2 --greedy`, instead of raw text.
  - Custom command overrides.
  - A timeout per brew command (10 minutes by default, `WithCommandTimeout`) so a stuck `brew update` cannot hang the app; cancellation via `context.Context`.

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
go run ./cmd/upgrades_cli               # one package per line
go run ./cmd/upgrades_cli -json         # JSON array
go run ./cmd/upgrades_cli -exit-code    # exit status 2 if updates exist
go run ./cmd/upgrades_cli -timeout 5m   # timeout per brew command
```

### Building and Packaging

The GUI app is best packaged as a native macOS `.app` bundle using Fyne's packaging tool:

1. Install the Fyne packaging tool if you haven't:

   ```bash
   go install fyne.io/tools/cmd/fyne@latest
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

For developers looking to integrate this checker into other Go projects:

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

// Defaults: brew from PATH or the standard locations, 10 min per command.
pkgs, err := checkupdates.Outdated(ctx)

// Or with options:
checker, err := checkupdates.NewChecker(
    checkupdates.WithConnectionTimeout(10 * time.Second),
    checkupdates.WithCommandTimeout(5 * time.Minute),
    checkupdates.WithUpdate("/usr/local/bin/brew", "update"),
)
pkgs, err = checker.Outdated(ctx)

for _, p := range pkgs {
    fmt.Println(p) // "git (2.44.0) < 2.45.1"
    // p.Name, p.Kind (Formula/Cask), p.InstalledVersions, p.CurrentVersion, p.Pinned
}
```

`ParseOutdatedJSON` is exported for parsing `brew outdated --json=v2` output you obtained yourself. Errors wrap `ErrBrewNotFound`, `ErrCommandTimeout` or `ErrUnexpectedOutput` for use with `errors.Is`.
