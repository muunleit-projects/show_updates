package checkupdates_test

import (
	"testing"

	cu "github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func TestShowUpdates(t *testing.T) {
	t.Parallel()

	c, err := cu.NewChecker(
		// cu.WithConnectionTries(4),
		// cu.WithConnectedTrue(),
		cu.WithUpdate("ls"),
		cu.WithUpgradeable("cat", "testfiles/three_dogs.txt"),
	)
	if err != nil {
		t.Fatal(err)
	}

	want := "waldi" + "\n" +
		"bello" + "\n" +
		"rex"

	got, err := c.Upgradable()
	if err != nil {
		t.Fatal(err)
	}

	if want != got {
		t.Errorf("\nwant %v \ngot \n%v", want, got)
	}
}

func TestNewCheckerInvalidInputs(t *testing.T) {
	t.Parallel()

	t.Run("invalid update command", func(t *testing.T) {
		_, err := cu.NewChecker(cu.WithUpdate(""))
		if err == nil {
			t.Error("expected error for empty update command, got nil")
		}
	})

	t.Run("invalid upgradeable command", func(t *testing.T) {
		_, err := cu.NewChecker(cu.WithUpgradeable(""))
		if err == nil {
			t.Error("expected error for empty upgradeable command, got nil")
		}
	})

	t.Run("invalid connection timeout", func(t *testing.T) {
		_, err := cu.NewChecker(cu.WithConnectionTimeout(-1))
		if err == nil {
			t.Error("expected error for negative connection timeout, got nil")
		}
	})
}

func TestNewCheckerDefaults(t *testing.T) {
	t.Parallel()

	_, err := cu.NewChecker(cu.WithConnectedTrue())
	if err != nil {
		t.Fatalf("unexpected error creating default checker: %v", err)
	}

	// Verify it executes successfully with overridden commands (so we don't call real brew update)
	c2, err := cu.NewChecker(
		cu.WithConnectedTrue(),
		cu.WithUpdate("echo", "mock_update"),
		cu.WithUpgradeable("echo", "mock_upgrade"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out, err := c2.Upgradable()
	if err != nil {
		t.Fatalf("unexpected error on Upgradable: %v", err)
	}
	if out != "mock_upgrade" {
		t.Errorf("expected mock_upgrade, got %q", out)
	}
}
