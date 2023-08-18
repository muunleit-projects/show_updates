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

// func TestNewCheckerInvalidInputs(t *testing.T) {
// 	t.Parallel()
// }
