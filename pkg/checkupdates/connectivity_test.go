package checkupdates

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestUpgradableProceedsWhenConnectivityCheckFails(t *testing.T) {
	oldDial := netDialTimeout
	netDialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, errors.New("probe failed")
	}
	t.Cleanup(func() {
		netDialTimeout = oldDial
	})

	c, err := NewChecker(
		WithUpdate("echo", "update"),
		WithUpgradeable("echo", "upgrade"),
	)
	if err != nil {
		t.Fatalf("unexpected error creating checker: %v", err)
	}

	out, err := c.Upgradable()
	if err != nil {
		t.Fatalf("expected Upgradable to continue after connectivity failure, got error: %v", err)
	}
	if out != "upgrade" {
		t.Fatalf("expected upgrade output, got %q", out)
	}
}
