package main

import (
	"log"
	"os"
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

func main() {
	const logfile string = "/tmp/outdated_apps.log"
	var s string

	a := app.New()
	w := a.NewWindow("outdated Apps")
	w.Resize(fyne.Size{Width: 200})
	upgrades, err := checkupdates.Upgradable()

	switch {
	case err != nil:
		wErr := writeError(err)
		if wErr != nil {
			s = "Error: " + wErr.Error()
			break
		}
		s = "Error: look at the logfile: " + logfile
	case upgrades != "":
		s = upgrades
	default:
		s = "no updates"
	}

	w.SetContent(widget.NewButton(s, func() {
		exec.Command("open", "/System/Applications/Utilities/Terminal.app").Run()
		a.Quit()
	}))

	w.ShowAndRun()
}

func writeError(err error) error {
	f, ferr := os.OpenFile("/tmp/outdated_apps.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o644)
	if ferr != nil {
		return ferr
	}
	defer func() {
		f.Close()
	}()

	l := log.Default()
	l.SetOutput(f)
	l.Println(err.Error())
	return nil
}
