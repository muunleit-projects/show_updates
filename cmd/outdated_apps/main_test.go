package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildErrorMessage(t *testing.T) {
	t.Parallel()

	t.Run("log written", func(t *testing.T) {
		t.Parallel()

		got := buildErrorMessage(errors.New("boom"), "/x/outdated_apps.log", nil)
		want := "Error: boom\n\nDetails were written to /x/outdated_apps.log"

		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("log failed", func(t *testing.T) {
		t.Parallel()

		got := buildErrorMessage(errors.New("boom"), "/x/outdated_apps.log", errors.New("read-only"))

		// The original error must always be visible.
		if !strings.HasPrefix(got, "Error: boom") || !strings.Contains(got, "read-only") {
			t.Fatalf("unexpected message %q", got)
		}
	})
}

func TestFormatNoUpdatesMessage(t *testing.T) {
	t.Parallel()

	got := formatNoUpdatesMessage(7)
	want := "No updates found.\nWindow closes in 7"

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLogFilePath(t *testing.T) {
	t.Parallel()

	if got, want := logFilePath("/Users/me", nil), "/Users/me/Library/Logs/outdated_apps.log"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	fallback := filepath.Join(os.TempDir(), "outdated_apps.log")

	if got := logFilePath("", errors.New("no home")); got != fallback {
		t.Errorf("got %q, want %q", got, fallback)
	}
}

func TestWriteLogCreatesDirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Library", "Logs", "outdated_apps.log")

	if err := writeLog(path, errors.New("first")); err != nil {
		t.Fatalf("writeLog: %v", err)
	}

	if err := writeLog(path, errors.New("second")); err != nil {
		t.Fatalf("writeLog: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	if !strings.Contains(string(data), "first") || !strings.Contains(string(data), "second") {
		t.Errorf("log should contain both entries, got %q", data)
	}
}

func TestCommandError(t *testing.T) {
	t.Parallel()

	if err := commandError("x", []byte("ignored"), nil); err != nil {
		t.Errorf("expected nil for successful command, got %v", err)
	}

	base := errors.New("exit status 1")

	err := commandError("could not open Terminal", []byte("  not allowed to send Apple events\n"), base)
	if !errors.Is(err, base) {
		t.Errorf("error should wrap the original error: %v", err)
	}

	want := "could not open Terminal: exit status 1: not allowed to send Apple events"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err, want)
	}

	if got := commandError("x", nil, base).Error(); got != "x: exit status 1" {
		t.Errorf("got %q without output", got)
	}
}

func TestLoadingFrame(t *testing.T) {
	t.Parallel()

	for n, want := range []string{"•", "••", "•••", "•", "••"} {
		if got := loadingFrame(n); got != want {
			t.Errorf("loadingFrame(%d) = %q, want %q", n, got, want)
		}
	}
}
