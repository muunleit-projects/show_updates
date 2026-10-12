package checkupdates_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	cu "github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

// None of these tests need Homebrew: both commands are always overridden.

const emptyJSON = `{"formulae":[],"casks":[]}`

// printEmpty is an outdated command that reports no updates.
var printEmpty = []string{"echo", emptyJSON}

func TestOutdated(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc        string
		updateCmd   []string
		outdatedCmd []string
		wantNames   []string
		wantErr     string
		wantErrIs   error
	}{
		{
			desc:        "reads packages from brew JSON",
			updateCmd:   []string{"true"},
			outdatedCmd: []string{"cat", "testfiles/outdated.json"},
			wantNames:   []string{"git", "node", "firefox", "visual-studio-code"},
		},
		{
			desc:        "no updates",
			updateCmd:   []string{"true"},
			outdatedCmd: []string{"cat", "testfiles/empty.json"},
			wantNames:   []string{},
		},
		{
			desc:        "stderr warnings do not break parsing",
			updateCmd:   []string{"true"},
			outdatedCmd: []string{"sh", "-c", "echo 'Warning: something' >&2; echo '" + emptyJSON + "'"},
			wantNames:   []string{},
		},
		{
			desc:        "non-JSON output is an error, not 'no updates'",
			updateCmd:   []string{"true"},
			outdatedCmd: []string{"echo", "waldi"},
			wantErrIs:   cu.ErrUnexpectedOutput,
		},
		{
			desc:        "failing update command",
			updateCmd:   []string{"sh", "-c", "echo update broke >&2; exit 1"},
			outdatedCmd: printEmpty,
			wantErr:     "update broke",
		},
		{
			desc:        "failing outdated command reports stderr",
			updateCmd:   []string{"true"},
			outdatedCmd: []string{"sh", "-c", "echo outdated broke >&2; exit 1"},
			wantErr:     "outdated broke",
		},
		{
			desc:        "missing executable",
			updateCmd:   []string{"this-command-does-not-exist-42"},
			outdatedCmd: printEmpty,
			wantErr:     "this-command-does-not-exist-42",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			c, err := cu.NewChecker(
				cu.WithConnectedTrue(),
				cu.WithUpdate(tt.updateCmd...),
				cu.WithOutdated(tt.outdatedCmd...),
			)
			if err != nil {
				t.Fatalf("NewChecker: unexpected error: %v", err)
			}

			pkgs, err := c.Outdated(context.Background())

			if tt.wantErr != "" || tt.wantErrIs != nil {
				if err == nil {
					t.Fatalf("expected error, got nil (packages %v)", pkgs)
				}

				if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}

				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error %q is not %v", err, tt.wantErrIs)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			names := make([]string, len(pkgs))
			for i, p := range pkgs {
				names[i] = p.Name
			}

			if strings.Join(names, ",") != strings.Join(tt.wantNames, ",") {
				t.Errorf("got %v, want %v", names, tt.wantNames)
			}
		})
	}
}

func TestOutdatedPackageFunc(t *testing.T) {
	t.Parallel()

	pkgs, err := cu.Outdated(context.Background(),
		cu.WithConnectedTrue(),
		cu.WithUpdate("true"),
		cu.WithOutdated("cat", "testfiles/outdated.json"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pkgs) != 4 {
		t.Errorf("got %d packages, want 4", len(pkgs))
	}

	if _, err := cu.Outdated(context.Background(), cu.WithUpdate("")); err == nil {
		t.Error("expected option error to be returned")
	}
}

func TestNewCheckerInvalidOptions(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc    string
		opts    []cu.Option
		wantErr string
	}{
		{"empty update command", []cu.Option{cu.WithUpdate(""), cu.WithOutdated("true")}, "empty update command"},
		{"no update command at all", []cu.Option{cu.WithUpdate(), cu.WithOutdated("true")}, "empty update command"},
		{"empty outdated command", []cu.Option{cu.WithUpdate("true"), cu.WithOutdated("")}, "empty outdated command"},
		{"no outdated command at all", []cu.Option{cu.WithUpdate("true"), cu.WithOutdated()}, "empty outdated command"},
		{"negative connection timeout", []cu.Option{cu.WithConnectionTimeout(-1)}, "timeout"},
		{"zero connection timeout", []cu.Option{cu.WithConnectionTimeout(0)}, "timeout"},
		{"nil dial function", []cu.Option{cu.WithDialTimeoutFunc(nil)}, "dial function"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			// Must not panic, and the error must come from the option itself,
			// not from a missing brew installation.
			_, err := cu.NewChecker(tt.opts...)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}
