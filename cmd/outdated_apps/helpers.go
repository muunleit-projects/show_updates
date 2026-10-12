package main

// Helpers without any Fyne dependency, kept separate so the logic is easy to
// test and reason about.

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

const (
	logFileName        string      = "outdated_apps.log"
	logDirPermissions  os.FileMode = 0o755
	logFilePermissions os.FileMode = 0o644
)

// logFilePath returns ~/Library/Logs/outdated_apps.log, the usual place for
// app logs on macOS (visible in Console.app). If the home directory is
// unknown, it falls back to the temp directory.
func logFilePath(home string, homeErr error) string {
	if homeErr != nil || home == "" {
		return filepath.Join(os.TempDir(), logFileName)
	}

	return filepath.Join(home, "Library", "Logs", logFileName)
}

func defaultLogFilePath() string {
	return logFilePath(os.UserHomeDir())
}

// writeLog appends err to the log file at path.
func writeLog(path string, err error) error {
	if mkErr := os.MkdirAll(filepath.Dir(path), logDirPermissions); mkErr != nil {
		return mkErr
	}

	f, openErr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFilePermissions)
	if openErr != nil {
		return openErr
	}

	l := log.New(f, "", log.LstdFlags)
	l.Println(err.Error())

	return f.Close()
}

// errorMessage logs err and returns the text shown to the user.
func errorMessage(err error) string {
	path := defaultLogFilePath()

	return buildErrorMessage(err, path, writeLog(path, err))
}

/*
buildErrorMessage always shows the error itself. It adds where the log was
written, or why writing the log failed.
*/
func buildErrorMessage(err error, logPath string, logErr error) string {
	msg := "Error: " + err.Error()

	if logErr != nil {
		return msg + "\n\n(Could not write log file: " + logErr.Error() + ")"
	}

	return msg + "\n\nDetails were written to " + logPath
}

// commandError turns a failed command into an error that includes its output.
func commandError(action string, output []byte, err error) error {
	if err == nil {
		return nil
	}

	if out := strings.TrimSpace(string(output)); out != "" {
		return fmt.Errorf("%s: %w: %s", action, err, out)
	}

	return fmt.Errorf("%s: %w", action, err)
}

func formatNoUpdatesMessage(countdown int) string {
	return "No updates found.\nWindow closes in " + strconv.Itoa(countdown)
}

var loadingFrames = []string{"•", "••", "•••"}

// loadingFrame returns the dots for animation step n.
func loadingFrame(n int) string {
	if n < 0 {
		n = -n
	}

	return loadingFrames[n%len(loadingFrames)]
}

func updatesTitle(n int) string {
	if n == 1 {
		return "1 update available"
	}

	return strconv.Itoa(n) + " updates available"
}

// packageLabel is the left column of a list row: the name, marked for casks.
func packageLabel(p checkupdates.Package) string {
	if p.Kind == checkupdates.Cask {
		return p.Name + " (app)"
	}

	return p.Name
}

// versionsLabel is the right column, e.g. "2.44.0 -> 2.45.1".
func versionsLabel(p checkupdates.Package) string {
	installed := strings.Join(p.InstalledVersions, ", ")

	var label string

	switch {
	case installed != "" && p.CurrentVersion != "":
		label = installed + " -> " + p.CurrentVersion
	case p.CurrentVersion != "":
		label = "-> " + p.CurrentVersion
	default:
		label = installed
	}

	if p.Pinned {
		label += " (pinned)"
	}

	return label
}

// pinnedHint explains why pinned packages will not be upgraded, or returns
// "" if none are pinned.
func pinnedHint(pkgs []checkupdates.Package) string {
	var pinned []string

	for _, p := range pkgs {
		if p.Pinned {
			pinned = append(pinned, p.Name)
		}
	}

	switch len(pinned) {
	case 0:
		return ""
	case 1:
		return pinned[0] + " is pinned and will be skipped by brew upgrade."
	default:
		return strings.Join(pinned, ", ") + " are pinned and will be skipped by brew upgrade."
	}
}
