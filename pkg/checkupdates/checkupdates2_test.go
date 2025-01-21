package checkupdates_test

import (
	"testing"

	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func TestShowUpgrades(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		desc            string
		inputUpdateCmd  []string
		inputUpgradeCmd []string
		outputExpected  string
		errExpected     error
	}{
		{
			desc:            "on testfiles/three_dogs.txt",
			inputUpdateCmd:  []string{"ls"},
			inputUpgradeCmd: []string{"cat", "testfiles/three_dogs.txt"},
			outputExpected:  "waldi\nbello\nrex",
			errExpected:     nil,
		},
		{
			desc:            "with homebrew (app needs to be installed)",
			inputUpdateCmd:  []string{"/opt/homebrew/bin/brew", "update"},
			inputUpgradeCmd: []string{"/opt/homebrew/bin/brew", "outdated", "-g"},
			outputExpected:  "",
			errExpected:     nil,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			c, err := checkupdates.NewChecker(
				checkupdates.WithUpdate(tt.inputUpdateCmd...),
				checkupdates.WithUpgradeable(tt.inputUpgradeCmd...),
			)

			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			upgrades, err := c.Upgradable()

			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if got, want := upgrades, tt.outputExpected; got != want {
				t.Errorf("got=%v, want=%v", got, want)
			}
		})
	}
}
