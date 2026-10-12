package checkupdates_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func failingDial(calls *atomic.Int32) func(string, string, time.Duration) (net.Conn, error) {
	return func(_, _ string, _ time.Duration) (net.Conn, error) {
		calls.Add(1)
		return nil, errors.New("probe failed")
	}
}

func TestOutdatedProceedsWhenConnectivityCheckFails(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	c, err := checkupdates.NewChecker(
		checkupdates.WithDialTimeoutFunc(failingDial(&calls)),
		checkupdates.WithUpdate("echo", "update"),
		checkupdates.WithOutdated(printEmpty...),
	)
	if err != nil {
		t.Fatalf("unexpected error creating checker: %v", err)
	}

	start := time.Now()

	pkgs, err := c.Outdated(context.Background())
	if err != nil {
		t.Fatalf("expected Outdated to continue after connectivity failure, got error: %v", err)
	}

	if len(pkgs) != 0 {
		t.Fatalf("expected no packages, got %v", pkgs)
	}

	// The probe must not retry in a loop and block the user.
	if n := calls.Load(); n != 1 {
		t.Errorf("expected exactly one probe, got %d", n)
	}

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Outdated took %v after a failed probe, expected it to continue right away", elapsed)
	}
}

func TestOutdatedReportsConnectivityWhenUpdateFails(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	c, err := checkupdates.NewChecker(
		checkupdates.WithDialTimeoutFunc(failingDial(&calls)),
		checkupdates.WithUpdate("false"),
		checkupdates.WithOutdated(printEmpty...),
	)
	if err != nil {
		t.Fatalf("unexpected error creating checker: %v", err)
	}

	_, err = c.Outdated(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "connectivity check failed") ||
		!strings.Contains(err.Error(), "probe failed") {
		t.Errorf("expected connectivity hint in error, got: %v", err)
	}
}

func TestConnectivityProbeArguments(t *testing.T) {
	t.Parallel()

	var (
		gotNetwork, gotAddress string
		gotTimeout             time.Duration
	)

	c, err := checkupdates.NewChecker(
		checkupdates.WithConnectionTimeout(1234*time.Millisecond),
		checkupdates.WithDialTimeoutFunc(func(network, address string, timeout time.Duration) (net.Conn, error) {
			gotNetwork, gotAddress, gotTimeout = network, address, timeout
			client, server := net.Pipe()
			_ = server.Close()

			return client, nil
		}),
		checkupdates.WithUpdate("true"),
		checkupdates.WithOutdated(printEmpty...),
	)
	if err != nil {
		t.Fatalf("unexpected error creating checker: %v", err)
	}

	if _, err := c.Outdated(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotNetwork != "tcp" || gotAddress != "github.com:443" {
		t.Errorf("probe dialed %s %s, want tcp github.com:443", gotNetwork, gotAddress)
	}

	if gotTimeout != 1234*time.Millisecond {
		t.Errorf("probe timeout %v, want 1.234s", gotTimeout)
	}
}

func TestWithConnectedTrueSkipsProbe(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	c, err := checkupdates.NewChecker(
		checkupdates.WithConnectedTrue(),
		checkupdates.WithDialTimeoutFunc(failingDial(&calls)),
		checkupdates.WithUpdate("true"),
		checkupdates.WithOutdated(printEmpty...),
	)
	if err != nil {
		t.Fatalf("unexpected error creating checker: %v", err)
	}

	if _, err := c.Outdated(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n := calls.Load(); n != 0 {
		t.Errorf("expected no probe, got %d", n)
	}
}
