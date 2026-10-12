package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
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

func TestUpdatesTitle(t *testing.T) {
	t.Parallel()

	for n, want := range map[int]string{1: "1 update available", 2: "2 updates available", 12: "12 updates available"} {
		if got := updatesTitle(n); got != want {
			t.Errorf("updatesTitle(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestRowLabels(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		pkg          checkupdates.Package
		wantName     string
		wantVersions string
	}{
		{
			checkupdates.Package{Name: "git", Kind: checkupdates.Formula, InstalledVersions: []string{"2.44.0"}, CurrentVersion: "2.45.1"},
			"git", "2.44.0 → 2.45.1",
		},
		{
			checkupdates.Package{Name: "firefox", Kind: checkupdates.Cask, InstalledVersions: []string{"125.0"}, CurrentVersion: "126.0"},
			"firefox (app)", "125.0 → 126.0",
		},
		{
			checkupdates.Package{Name: "node", Kind: checkupdates.Formula, InstalledVersions: []string{"20", "21"}, CurrentVersion: "22", Pinned: true},
			"node", "20, 21 → 22 (pinned)",
		},
		{
			checkupdates.Package{Name: "x", CurrentVersion: "2"},
			"x", "→ 2",
		},
		{
			checkupdates.Package{Name: "y", InstalledVersions: []string{"1"}},
			"y", "1",
		},
	} {
		if got := packageLabel(tt.pkg); got != tt.wantName {
			t.Errorf("packageLabel = %q, want %q", got, tt.wantName)
		}

		if got := versionsLabel(tt.pkg); got != tt.wantVersions {
			t.Errorf("versionsLabel = %q, want %q", got, tt.wantVersions)
		}
	}
}

func TestPinnedHint(t *testing.T) {
	t.Parallel()

	a := checkupdates.Package{Name: "a"}
	b := checkupdates.Package{Name: "b", Pinned: true}
	c := checkupdates.Package{Name: "c", Pinned: true}

	if got := pinnedHint([]checkupdates.Package{a}); got != "" {
		t.Errorf("expected no hint, got %q", got)
	}

	if got := pinnedHint([]checkupdates.Package{a, b}); got != "b is pinned and will be skipped by brew upgrade." {
		t.Errorf("unexpected hint %q", got)
	}

	if got := pinnedHint([]checkupdates.Package{b, a, c}); got != "b, c are pinned and will be skipped by brew upgrade." {
		t.Errorf("unexpected hint %q", got)
	}
}
