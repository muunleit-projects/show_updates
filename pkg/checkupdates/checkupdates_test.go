package checkupdates_test

import (
	"strings"
	"testing"

	cu "github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

// None of these tests need Homebrew: both commands are always overridden.

func TestUpgradable(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc       string
		updateCmd  []string
		upgradeCmd []string
		want       string
		wantErr    string
	}{
		{
			desc:       "reads package list from file",
			updateCmd:  []string{"true"},
			upgradeCmd: []string{"cat", "testfiles/three_dogs.txt"},
			want:       "waldi\nbello\nrex",
		},
		{
			desc:       "trims surrounding whitespace",
			updateCmd:  []string{"true"},
			upgradeCmd: []string{"echo", "  mock_upgrade  "},
			want:       "mock_upgrade",
		},
		{
			desc:       "empty output means no updates",
			updateCmd:  []string{"true"},
			upgradeCmd: []string{"true"},
			want:       "",
		},
		{
			desc:       "stderr warnings are not part of the package list",
			updateCmd:  []string{"true"},
			upgradeCmd: []string{"sh", "-c", "echo 'Warning: something' >&2; echo pkg"},
			want:       "pkg",
		},
		{
			desc:       "failing update command",
			updateCmd:  []string{"sh", "-c", "echo update broke >&2; exit 1"},
			upgradeCmd: []string{"echo", "never"},
			wantErr:    "update broke",
		},
		{
			desc:       "failing upgrade command reports stderr",
			updateCmd:  []string{"true"},
			upgradeCmd: []string{"sh", "-c", "echo outdated broke >&2; exit 1"},
			wantErr:    "outdated broke",
		},
		{
			desc:       "missing executable",
			updateCmd:  []string{"this-command-does-not-exist-42"},
			upgradeCmd: []string{"true"},
			wantErr:    "this-command-does-not-exist-42",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			c, err := cu.NewChecker(
				cu.WithConnectedTrue(),
				cu.WithUpdate(tt.updateCmd...),
				cu.WithUpgradeable(tt.upgradeCmd...),
			)
			if err != nil {
				t.Fatalf("NewChecker: unexpected error: %v", err)
			}

			got, err := c.Upgradable()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (output %q)", tt.wantErr, got)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewCheckerInvalidOptions(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc    string
		opt     func() error
		wantErr string
	}{
		{
			desc:    "empty update command",
			wantErr: "empty update command",
			opt: func() error {
				_, err := cu.NewChecker(cu.WithUpdate(""), cu.WithUpgradeable("true"))
				return err
			},
		},
		{
			desc:    "no update command at all",
			wantErr: "empty update command",
			opt: func() error {
				_, err := cu.NewChecker(cu.WithUpdate(), cu.WithUpgradeable("true"))
				return err
			},
		},
		{
			desc:    "empty upgradeable command",
			wantErr: "empty upgradeable command",
			opt: func() error {
				_, err := cu.NewChecker(cu.WithUpdate("true"), cu.WithUpgradeable(""))
				return err
			},
		},
		{
			desc:    "no upgradeable command at all",
			wantErr: "empty upgradeable command",
			opt: func() error {
				_, err := cu.NewChecker(cu.WithUpdate("true"), cu.WithUpgradeable())
				return err
			},
		},
		{
			desc:    "negative connection timeout",
			wantErr: "timeout",
			opt: func() error {
				_, err := cu.NewChecker(cu.WithConnectionTimeout(-1))
				return err
			},
		},
		{
			desc:    "zero connection timeout",
			wantErr: "timeout",
			opt: func() error {
				_, err := cu.NewChecker(cu.WithConnectionTimeout(0))
				return err
			},
		},
		{
			desc:    "nil dial function",
			wantErr: "dial function",
			opt: func() error {
				_, err := cu.NewChecker(cu.WithDialTimeoutFunc(nil))
				return err
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			// Must not panic, and the error must come from the option itself,
			// not from a missing brew installation.
			err := tt.opt()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}
