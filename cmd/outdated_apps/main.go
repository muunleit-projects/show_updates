package main

import (
	"errors"
	"log"
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

const logfile string = "/tmp/outdated_apps.log"

func main() {
	var c fyne.CanvasObject

	a := app.New()
	w := a.NewWindow("outdated Apps")
	w.Resize(fyne.Size{Width: 200})

	upgrades, err := checkupdates.Upgradable()
	switch {
	case err != nil:
		err := logError(err)
		if err != nil {
			c = widget.NewLabel("Error: " + err.Error())
		}
		c = widget.NewLabel("Error: see " + logfile)
	case len(upgrades) == 0 || upgrades == "":
		c = widget.NewLabel("no updates")
	default:
		c = container.NewVBox(
			widget.NewLabel(upgrades),
			widget.NewButton("open Terminal", openTerminal),
		)
	}

	w.SetContent(c)
	w.ShowAndRun()
}

// func makeUI() fyne.CanvasObject {
// 	upgrades, err := checkupdates.Upgradable()
// 	if err != nil {
// 		err := logError(err)
// 		if err != nil {
// 			return widget.NewLabel("Error: " + err.Error())
// 		}
// 		return widget.NewLabel("Error: " + logfile)
// 	}
// 	if upgrades == "" {
// 		return widget.NewLabel("no updates")
// 	}
// 	return container.NewVBox(
// 		widget.NewLabel(upgrades),
// 		widget.NewButton("open Terminal", openTerminal),
// 	)
// }

func logError(err error) error {
	f, fError := os.OpenFile(
		logfile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o644)
	if fError != nil {
		return errors.Join(fError, err)
	}
	defer func() {
		f.Close()
	}()

	l := log.New(f, "", log.LstdFlags)
	l.Println(err.Error())
	return nil
}

func openTerminal() {
	exec.Command("open", "/System/Applications/Utilities/Terminal.app").Run()
}
