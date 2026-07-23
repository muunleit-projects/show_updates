package checkupdates_test

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func TestUpgradableProceedsWhenConnectivityCheckFails(t *testing.T) {
	c, err := checkupdates.NewChecker(
		checkupdates.WithDialTimeoutFunc(func(network, address string, timeout time.Duration) (net.Conn, error) {
			return nil, errors.New("probe failed")
		}),
		checkupdates.WithUpdate("echo", "update"),
		checkupdates.WithUpgradeable("echo", "upgrade"),
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
