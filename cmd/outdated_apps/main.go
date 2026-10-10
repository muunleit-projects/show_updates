package main

import (
	"context"
	"os/exec"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/muunleit-projects/show_updates/pkg/checkupdates"
)

const (
	appID            string = "com.muunleit.showupdates"
	minfieldwidth    int    = 360
	windowHeight     int    = 320
	loadingDelay            = 400 * time.Millisecond
	countdownSeconds int    = 15
)

/*
Threading: Fyne (since 2.6) requires every change to widgets or windows that
happens outside of a Fyne callback to go through fyne.Do. All background work
in this file (the update check, the loading animation, the countdown and the
Terminal launch) therefore wraps its UI changes in fyne.Do. Data bindings are
safe to set from any goroutine.
*/

func main() {
	a := app.NewWithID(appID)
	w := a.NewWindow("Outdated Apps")
	w.Resize(fyne.NewSize(float32(minfieldwidth), float32(windowHeight)))

	showError := func(err error) {
		w.SetContent(buildErrorView(errorMessage(err), a))
	}

	openTerminal := func() {
		// osascript may block on the macOS automation permission prompt, so
		// keep it off the UI thread.
		go func() {
			err := runUpgradeInTerminal()
			fyne.Do(func() {
				if err != nil {
					showError(err)
					return
				}

				a.Quit()
			})
		}()
	}

	loadingDots := newLoadingDots()
	stopLoading := make(chan struct{})

	w.SetContent(buildLoadingView(loadingDots))
	go animateLoadingDots(loadingDots, stopLoading)

	go func() {
		pkgs, err := checkupdates.Outdated(context.Background())

		close(stopLoading)

		fyne.Do(func() {
			switch {
			case err != nil:
				showError(err)
			case len(pkgs) == 0:
				w.SetContent(noUpdates(a))
			default:
				w.SetContent(buildUpdatesView(pkgs, openTerminal, a))
			}
		})
	}()

	w.ShowAndRun()
}

// runUpgradeInTerminal opens Terminal.app and starts the upgrade there, so the
// user can follow the output and answer password prompts.
func runUpgradeInTerminal() error {
	output, err := exec.Command(
		"osascript",
		"-e", `tell application "Terminal"`,
		"-e", `activate`,
		"-e", `do script "brew upgrade -g -y"`,
		"-e", `end tell`,
	).CombinedOutput()

	return commandError("could not open Terminal", output, err)
}

func buildLoadingView(loadingDots *widget.Label) fyne.CanvasObject {
	loadingTitle := widget.NewLabelWithStyle(
		"Checking for updates…",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	loadingStatus := widget.NewLabelWithStyle(
		"Refreshing Homebrew package info",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)
	loadingHint := widget.NewLabelWithStyle(
		"This may take a moment while Homebrew checks for updates.",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)
	loadingHint.Wrapping = fyne.TextWrapWord

	return container.NewCenter(
		container.NewVBox(loadingTitle, loadingStatus, loadingHint, loadingDots),
	)
}

func newLoadingDots() *widget.Label {
	loadingDots := widget.NewLabel("•")
	loadingDots.TextStyle = fyne.TextStyle{Bold: true}
	loadingDots.Alignment = fyne.TextAlignCenter

	return loadingDots
}

// animateLoadingDots cycles the dots until stop is closed.
func animateLoadingDots(loadingDots *widget.Label, stop <-chan struct{}) {
	ticker := time.NewTicker(loadingDelay)
	defer ticker.Stop()

	for frame := 1; ; frame++ {
		select {
		case <-stop:
			return
		case <-ticker.C:
			text := loadingFrame(frame)
			fyne.Do(func() { loadingDots.SetText(text) })
		}
	}
}

func buildErrorView(message string, a fyne.App) fyne.CanvasObject {
	messageLabel := widget.NewLabel(message)
	messageLabel.Wrapping = fyne.TextWrapWord

	return container.NewBorder(
		widget.NewLabelWithStyle(
			"Update check failed",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		widget.NewButton("Quit", func() { a.Quit() }),
		nil,
		nil,
		container.NewVScroll(messageLabel),
	)
}

func buildUpdatesView(pkgs []checkupdates.Package, openTerminal func(), a fyne.App) fyne.CanvasObject {
	list := widget.NewList(
		func() int { return len(pkgs) },
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			name.Truncation = fyne.TextTruncateEllipsis

			versions := widget.NewLabel("")
			versions.Alignment = fyne.TextAlignTrailing

			// Objects order is [center, right] = [name, versions].
			return container.NewBorder(nil, nil, nil, versions, name)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			p := pkgs[id]
			row := item.(*fyne.Container)
			name := row.Objects[0].(*widget.Label)
			versions := row.Objects[1].(*widget.Label)

			name.SetText(packageLabel(p))
			versions.SetText(versionsLabel(p))

			if p.Pinned {
				versions.Importance = widget.WarningImportance
			} else {
				versions.Importance = widget.LowImportance
			}

			versions.Refresh()
		},
	)

	bottom := container.NewVBox(
		widget.NewButton("Open Terminal and Upgrade", openTerminal),
		widget.NewButton("Quit", func() { a.Quit() }),
	)

	if hint := pinnedHint(pkgs); hint != "" {
		hintLabel := widget.NewLabel(hint)
		hintLabel.Wrapping = fyne.TextWrapWord
		hintLabel.Importance = widget.LowImportance
		bottom = container.NewVBox(hintLabel, bottom)
	}

	return container.NewBorder(
		widget.NewLabelWithStyle(
			updatesTitle(len(pkgs)),
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		bottom,
		nil,
		nil,
		list,
	)
}

/*
noUpdates messages "no updates" with a countdown and close window after the
countdown went off, because I was tired of closing it every time by hand if
there  was nothing to do. It needs the fyne.App (a) to do the closing.
*/
func noUpdates(a fyne.App) fyne.CanvasObject {
	str := binding.NewString()
	_ = str.Set(formatNoUpdatesMessage(countdownSeconds))

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for countdown := countdownSeconds - 1; countdown > 0; countdown-- {
			<-ticker.C
			_ = str.Set(formatNoUpdatesMessage(countdown))
		}

		<-ticker.C
		fyne.Do(a.Quit)
	}()

	return container.NewCenter(container.NewVBox(
		widget.NewLabelWithStyle(
			"All up to date",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		widget.NewLabelWithData(str),
		widget.NewButton("Close Now", func() { a.Quit() }),
	))
}
