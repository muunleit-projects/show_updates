package checkupdates_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cu "github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func TestOutdatedKillsHangingUpdate(t *testing.T) {
	t.Parallel()

	c, err := cu.NewChecker(
		cu.WithConnectedTrue(),
		cu.WithCommandTimeout(200*time.Millisecond),
		cu.WithUpdate("sleep", "30"),
		cu.WithOutdated(printEmpty...),
	)
	if err != nil {
		t.Fatalf("unexpected error creating checker: %v", err)
	}

	start := time.Now()

	_, err = c.Outdated(context.Background())
	if !errors.Is(err, cu.ErrCommandTimeout) {
		t.Fatalf("expected ErrCommandTimeout, got: %v", err)
	}

	if !strings.Contains(err.Error(), "200ms") {
		t.Errorf("expected timeout duration in error, got: %v", err)
	}

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("hanging command was not killed in time, took %v", elapsed)
	}
}

func TestOutdatedKillsHangingOutdatedCheck(t *testing.T) {
	t.Parallel()

	c, err := cu.NewChecker(
		cu.WithConnectedTrue(),
		cu.WithCommandTimeout(200*time.Millisecond),
		cu.WithUpdate("true"),
		cu.WithOutdated("sleep", "30"),
	)
	if err != nil {
		t.Fatalf("unexpected error creating checker: %v", err)
	}

	if _, err := c.Outdated(context.Background()); !errors.Is(err, cu.ErrCommandTimeout) {
		t.Fatalf("expected ErrCommandTimeout, got: %v", err)
	}
}

func TestOutdatedContextCancel(t *testing.T) {
	t.Parallel()

	c, err := cu.NewChecker(
		cu.WithConnectedTrue(),
		cu.WithUpdate("sleep", "30"),
		cu.WithOutdated(printEmpty...),
	)
	if err != nil {
		t.Fatalf("unexpected error creating checker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)

	start := time.Now()

	_, err = c.Outdated(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}

	// A cancellation by the caller is not reported as our own timeout.
	if errors.Is(err, cu.ErrCommandTimeout) {
		t.Errorf("cancellation reported as command timeout: %v", err)
	}

	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("command was not stopped on cancel, took %v", elapsed)
	}
}

func TestWithCommandTimeoutRejectsNonPositive(t *testing.T) {
	t.Parallel()

	for _, d := range []time.Duration{0, -time.Second} {
		_, err := cu.NewChecker(cu.WithCommandTimeout(d), cu.WithUpdate("true"), cu.WithOutdated(printEmpty...))
		if err == nil || !strings.Contains(err.Error(), "command timeout") {
			t.Errorf("WithCommandTimeout(%v): expected error, got %v", d, err)
		}
	}
}
