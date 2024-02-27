package main

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
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
		// message "no updates" and close window after some seconds, because I was
		// tired of closing it every time by hand if there was nothing to do
		w.SetContent(noUpdates(a))

	default:
		w.SetContent(container.NewVBox(
			widget.NewLabel(upgrades),
			widget.NewButton("open Terminal", openTerminal),
		))
	}

	w.ShowAndRun()
}

// noUpdates messages "no updates" with a countdown and close window after after
// the countdown went off, because I was tired of closing it every time by hand
// if there was nothing to do. It needs the fyne.App (a) to do the closing
func noUpdates(a fyne.App) fyne.CanvasObject {
	countdown := 15
	str := binding.NewString()

	go func() {
		for countdown > 0 {
			str.Set("No updates found. \nWindow closes in " + strconv.Itoa(countdown))

			countdown--

			time.Sleep(time.Second)
		}
		a.Quit()
	}()

	return widget.NewLabelWithData(str)
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
