package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

var samplePackages = []checkupdates.Package{
	{Name: "git", Kind: checkupdates.Formula, InstalledVersions: []string{"2.44.0"}, CurrentVersion: "2.45.1"},
	{Name: "firefox", Kind: checkupdates.Cask, InstalledVersions: []string{"125.0"}, CurrentVersion: "126.0"},
}

func fakeOutdated(pkgs []checkupdates.Package, err error) outdatedFunc {
	return func(context.Context, ...checkupdates.Option) ([]checkupdates.Package, error) {
		return pkgs, err
	}
}

func runCLI(t *testing.T, args []string, f outdatedFunc) (int, string, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), args, &stdout, &stderr, f)

	return code, stdout.String(), stderr.String()
}

func TestRunPrintsOnePackagePerLine(t *testing.T) {
	t.Parallel()

	code, out, _ := runCLI(t, nil, fakeOutdated(samplePackages, nil))

	if code != exitOK {
		t.Errorf("exit code %d, want %d", code, exitOK)
	}

	want := "git (2.44.0) < 2.45.1\nfirefox (125.0) < 126.0\n"
	if out != want {
		t.Errorf("stdout %q, want %q", out, want)
	}
}

func TestRunUpToDate(t *testing.T) {
	t.Parallel()

	code, out, errOut := runCLI(t, []string{"-exit-code"}, fakeOutdated([]checkupdates.Package{}, nil))

	if code != exitOK {
		t.Errorf("exit code %d, want %d", code, exitOK)
	}

	if out != "" {
		t.Errorf("stdout should be empty, got %q", out)
	}

	if !strings.Contains(errOut, "up to date") {
		t.Errorf("stderr %q should say up to date", errOut)
	}
}

func TestRunExitCode(t *testing.T) {
	t.Parallel()

	if code, _, _ := runCLI(t, []string{"-exit-code"}, fakeOutdated(samplePackages, nil)); code != exitUpdates {
		t.Errorf("exit code %d, want %d", code, exitUpdates)
	}
}

func TestRunJSON(t *testing.T) {
	t.Parallel()

	code, out, _ := runCLI(t, []string{"-json"}, fakeOutdated(samplePackages, nil))
	if code != exitOK {
		t.Fatalf("exit code %d", code)
	}

	var got []checkupdates.Package
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}

	if len(got) != 2 || got[1].Kind != checkupdates.Cask || got[0].CurrentVersion != "2.45.1" {
		t.Errorf("unexpected JSON content: %+v", got)
	}

	// No updates must still be valid JSON: an empty array, not null.
	_, out, _ = runCLI(t, []string{"-json"}, fakeOutdated([]checkupdates.Package{}, nil))
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("expected [], got %q", out)
	}
}

func TestRunError(t *testing.T) {
	t.Parallel()

	code, out, errOut := runCLI(t, nil, fakeOutdated(nil, errors.New("brew exploded")))

	if code != exitError {
		t.Errorf("exit code %d, want %d", code, exitError)
	}

	if out != "" || !strings.Contains(errOut, "brew exploded") {
		t.Errorf("stdout %q, stderr %q", out, errOut)
	}
}

func TestRunPassesTimeout(t *testing.T) {
	t.Parallel()

	// The timeout flag must reach the checker: a hanging update command is
	// killed after the given timeout.
	f := func(ctx context.Context, opts ...checkupdates.Option) ([]checkupdates.Package, error) {
		opts = append(opts,
			checkupdates.WithConnectedTrue(),
			checkupdates.WithUpdate("sleep", "30"),
			checkupdates.WithOutdated("true"),
		)

		return checkupdates.Outdated(ctx, opts...)
	}

	start := time.Now()
	code, _, errOut := runCLI(t, []string{"-timeout", "200ms"}, f)

	if code != exitError || !strings.Contains(errOut, "timed out") {
		t.Errorf("exit code %d, stderr %q", code, errOut)
	}

	if time.Since(start) > 5*time.Second {
		t.Errorf("timeout flag was not applied")
	}
}

func TestRunUsageErrors(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"-nope"}, {"extra"}, {"-timeout", "soon"}} {
		if code, _, _ := runCLI(t, args, fakeOutdated(nil, nil)); code != exitUsage {
			t.Errorf("args %v: exit code %d, want %d", args, code, exitUsage)
		}
	}

	if code, _, _ := runCLI(t, []string{"-h"}, fakeOutdated(nil, nil)); code != exitOK {
		t.Errorf("-h: exit code %d, want %d", code, exitOK)
	}
}
