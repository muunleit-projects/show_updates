package checkupdates

import (
	"errors"
	"testing"
)

func TestFindBrew(t *testing.T) {
	t.Parallel()

	notInPath := func(string) (string, error) { return "", errors.New("not found") }

	for _, tt := range []struct {
		desc     string
		lookPath func(string) (string, error)
		existing map[string]bool
		want     string
		wantErr  error
	}{
		{
			desc:     "brew in PATH wins",
			lookPath: func(string) (string, error) { return "/custom/bin/brew", nil },
			existing: map[string]bool{"/opt/homebrew/bin/brew": true},
			want:     "/custom/bin/brew",
		},
		{
			desc:     "Apple Silicon location",
			lookPath: notInPath,
			existing: map[string]bool{"/opt/homebrew/bin/brew": true, "/usr/local/bin/brew": true},
			want:     "/opt/homebrew/bin/brew",
		},
		{
			desc:     "Intel location",
			lookPath: notInPath,
			existing: map[string]bool{"/usr/local/bin/brew": true},
			want:     "/usr/local/bin/brew",
		},
		{
			desc:     "not installed",
			lookPath: notInPath,
			existing: map[string]bool{},
			wantErr:  ErrBrewNotFound,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			got, err := findBrew(tt.lookPath, func(p string) bool { return tt.existing[p] })
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewCheckerSkipsBrewLookupWhenBothCommandsSet(t *testing.T) {
	t.Parallel()

	// Works on machines without Homebrew because no lookup is needed.
	c, err := NewChecker(WithUpdate("true"), WithUpgradeable("true"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.update[0] != "true" || c.upgrade[0] != "true" {
		t.Errorf("commands were overwritten: update=%v upgrade=%v", c.update, c.upgrade)
	}
}

func TestNewCheckerDefaultsUseBrew(t *testing.T) {
	t.Parallel()

	c, err := NewChecker(WithUpgradeable("true"))
	if errors.Is(err, ErrBrewNotFound) {
		t.Skip("Homebrew not installed on this machine")
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.update) != 2 || c.update[1] != "update" {
		t.Errorf("default update command = %v, want [<brew> update]", c.update)
	}

	if c.upgrade[0] != "true" {
		t.Errorf("custom upgrade command was overwritten: %v", c.upgrade)
	}

	if c.connectionTimeout != timeoutTime {
		t.Errorf("default timeout = %v, want %v", c.connectionTimeout, timeoutTime)
	}
}
