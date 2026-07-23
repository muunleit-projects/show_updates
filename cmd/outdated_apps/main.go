package main

import (
	"errors"
	"fmt"
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

const (
	logfile       string = "/tmp/outdated_apps.log"
	minfieldwidth int    = 240
)

func main() {
	a := app.New()
	w := a.NewWindow("Outdated Apps")

	w.Resize(fyne.NewSize(float32(minfieldwidth), 320))

	openTerminal := func() {
		exec.Command("osascript",
			"-e", `tell application "Terminal"`,
			"-e", `activate`,
			"-e", `do script "brew upgrade -g -y"`,
			"-e", `end tell`).Run()
		a.Quit()
	}

	loadingTitle := widget.NewLabelWithStyle("Checking for updates…", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	loadingStatus := widget.NewLabelWithStyle("Refreshing Homebrew package info", fyne.TextAlignCenter, fyne.TextStyle{})
	loadingHint := widget.NewLabelWithStyle("This may take a moment while Homebrew checks for updates.", fyne.TextAlignCenter, fyne.TextStyle{})
	loadingDots := widget.NewLabel("•")
	loadingDots.TextStyle = fyne.TextStyle{Bold: true}
	loadingDots.Alignment = fyne.TextAlignCenter

	w.SetContent(container.NewCenter(container.NewVBox(loadingTitle, loadingStatus, loadingHint, loadingDots)))

	go func() {
		frames := []string{"•", "••", "•••"}
		for {
			for _, frame := range frames {
				loadingDots.SetText(frame)
				time.Sleep(400 * time.Millisecond)
			}
		}
	}()

	go func() {
		upgrades, err := checkupdates.Upgradable()

		switch {
		case err != nil:
			logErr := logError(err)
			message := buildErrorMessage(logErr, logfile)
			w.SetContent(container.NewCenter(container.NewVBox(
				widget.NewLabelWithStyle("Update check failed", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel(message),
				widget.NewButton("Quit", func() { a.Quit() }),
			)))
		case len(upgrades) == 0 || upgrades == "":
			w.SetContent(noUpdates(a))

		default:
			upgradesLabel := widget.NewLabel(upgrades)
			upgradesLabel.Wrapping = fyne.TextWrapWord
			upgradesLabel.Alignment = fyne.TextAlignCenter

			w.SetContent(container.NewBorder(
				widget.NewLabelWithStyle("Updates available", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				container.NewVBox(
					widget.NewButton("Open Terminal and Upgrade", openTerminal),
					widget.NewButton("Quit", func() { a.Quit() }),
				),
				nil,
				nil,
				container.NewVScroll(upgradesLabel),
			))
		}
	}()

	w.ShowAndRun()
}

/*
noUpdates messages "no updates" with a countdown and close window after the
countdown went off, because I was tired of closing it every time by hand if
there  was nothing to do. It needs the fyne.App (a) to do the closing.
*/
func noUpdates(a fyne.App) fyne.CanvasObject {
	countdown := 15
	str := binding.NewString()

	go func() {
		for countdown > 0 {
			str.Set(formatNoUpdatesMessage(countdown))
			countdown--
			time.Sleep(time.Second)
		}

		a.Quit()
	}()

	return container.NewCenter(container.NewVBox(
		widget.NewLabelWithStyle("All up to date", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithData(str),
		widget.NewButton("Close Now", func() { a.Quit() }),
	))
}

func buildErrorMessage(err error, logfilePath string) string {
	if err == nil {
		return fmt.Sprintf("Error: see %s", logfilePath)
	}
	return fmt.Sprintf("Error: %s", err.Error())
}

func formatNoUpdatesMessage(countdown int) string {
	return "No updates found.\nWindow closes in " + strconv.Itoa(countdown)
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
