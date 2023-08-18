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
	a := app.New()
	w := a.NewWindow("outdated Apps")

	w.Resize(fyne.Size{Width: 200})

	openTerminal := func() {
		exec.Command("open", "/System/Applications/Utilities/Terminal.app").Run()
		a.Quit()
	}

	upgrades, err := checkupdates.Upgradable()

	switch {
	case err != nil:
		err := logError(err)
		if err != nil {
			w.SetContent(widget.NewLabel("Error: " + err.Error()))
		}

		w.SetContent(widget.NewLabel("Error: see " + logfile))
	case len(upgrades) == 0 || upgrades == "":
		w.SetContent(widget.NewLabel("no updates"))
	default:
		w.SetContent(container.NewVBox(
			widget.NewLabel(upgrades),
			widget.NewButton("open Terminal", openTerminal),
		))
	}

	w.ShowAndRun()
}

func logError(err error) error {
	f, fError := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
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
