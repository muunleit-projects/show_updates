package main

import (
	"fmt"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func main() {
	var s string

	a := app.New()
	w := a.NewWindow("brew updates")
	upgrades, err := checkupdates.Upgradable()

	switch {
	case err != nil:
		s = fmt.Sprint("Error: ", err.Error())
	case upgrades != "":
		s = upgrades
	default:
		s = "no updates"
	}

	w.SetContent(widget.NewLabel(s))
	w.ShowAndRun()
}
