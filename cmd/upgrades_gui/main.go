package main

import (
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func main() {
	upgrades, err := checkupdates.Upgradable()
	if err != nil {
		os.Exit(1)
	}

	a := app.New()
	w := a.NewWindow("brew updates")
	w.SetContent(widget.NewLabel(upgrades))

	w.ShowAndRun()
}
