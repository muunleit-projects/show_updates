package main

import (
	"fmt"
	"os/exec"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func main() {
	a := app.New()
	w := a.NewWindow("outdated Apps")
	upgrades, err := checkupdates.Upgradable()

	switch {
	case err != nil:
		w.SetContent(
			widget.NewLabel(
				fmt.Sprint("Error: ", err.Error(), "\n", upgrades)))
	case upgrades != "":
		w.SetContent(
			widget.NewButton(upgrades, func() {
				exec.Command("open", "/System/Applications/Utilities/Terminal.app").Run()
				a.Quit()
			}))
	default:
		w.SetContent(
			widget.NewLabel("no updates"))
	}

	w.ShowAndRun()
}
