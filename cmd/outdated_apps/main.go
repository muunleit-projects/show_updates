package main

import (
	"errors"
	"fmt"
	"io"
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
	logfile string = "/tmp/outdated_apps.log"
)

func getBrewPath() string {
	brewPath, err := exec.LookPath("brew")
	if err == nil {
		return brewPath
	}
	// check common paths since PATH might be limited in GUI apps
	commonPaths := []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}
	for _, p := range commonPaths {
		if _, e := os.Stat(p); e == nil {
			return p
		}
	}
	return "brew" // fallback to raw string if not found
}

func main() {
	a := app.New()
	w := a.NewWindow("Outdated Apps")

	// Resize to a more comfortable, readable window size
	w.Resize(fyne.NewSize(480, 360))

	openTerminal := func() {
		brewPath := getBrewPath()
		exec.Command("osascript",
			"-e", `tell application "Terminal"`,
			"-e", `activate`,
			"-e", `do script "`+brewPath+` upgrade -g -y"`,
			"-e", `end tell`).Run()
		a.Quit()
	}

	// Loading State UI
	title := widget.NewLabelWithStyle("Homebrew Update Helper", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	progress := widget.NewProgressBarInfinite()
	progress.Start()
	statusLabel := widget.NewLabelWithStyle("Checking for updates...", fyne.TextAlignCenter, fyne.TextStyle{})

	w.SetContent(container.NewBorder(
		title,
		nil,
		nil,
		nil,
		container.NewCenter(container.NewVBox(progress, statusLabel)),
	))

	go func() {
		upgrades, err := checkupdates.Upgradable()
		progress.Stop()

		switch {
		case err != nil:
			err := logError(err)
			errorMsg := "Error checking for updates"
			if err != nil {
				errorMsg = fmt.Sprintf("Error: %s", err.Error())
			} else {
				errorMsg = fmt.Sprintf("Error: see %s", logfile)
			}
			w.SetContent(container.NewBorder(
				title,
				widget.NewButton("Quit", func() { a.Quit() }),
				nil,
				nil,
				container.NewCenter(widget.NewLabel(errorMsg)),
			))

		case len(upgrades) == 0 || upgrades == "":
			w.SetContent(noUpdates(a))

		default:
			upgradesLabel := widget.NewLabel(upgrades)
			upgradesLabel.Wrapping = fyne.TextWrapWord

			scroll := container.NewVScroll(upgradesLabel)
			scroll.SetMinSize(fyne.NewSize(400, 180))

			upgradeInAppBtn := widget.NewButton("Upgrade All (In-App)", func() {
				runInAppUpgrade(w, a)
			})
			upgradeInAppBtn.Importance = widget.HighImportance

			terminalBtn := widget.NewButton("Open in Terminal", openTerminal)
			cancelBtn := widget.NewButton("Cancel", func() {
				a.Quit()
			})

			w.SetContent(container.NewBorder(
				widget.NewLabelWithStyle("Outdated Packages Found", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				container.NewVBox(
					upgradeInAppBtn,
					container.NewGridWithColumns(2, terminalBtn, cancelBtn),
				),
				nil,
				nil,
				scroll,
			))
		}
	}()

	w.ShowAndRun()
}

func runInAppUpgrade(w fyne.Window, a fyne.App) {
	title := widget.NewLabelWithStyle("Upgrading Packages...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	progress := widget.NewProgressBarInfinite()
	progress.Start()

	logEntry := widget.NewMultiLineEntry()
	logEntry.Disable()
	logEntry.Wrapping = fyne.TextWrapWord

	scrollLog := container.NewVScroll(logEntry)
	scrollLog.SetMinSize(fyne.NewSize(400, 180))

	// Password / Stdin Input Container
	inputField := widget.NewPasswordEntry()
	inputField.SetPlaceHolder("Enter password / input if prompted...")

	var cmdStdin io.WriteCloser

	sendInput := func() {
		text := inputField.Text
		if text == "" {
			return
		}
		if cmdStdin != nil {
			_, _ = cmdStdin.Write([]byte(text + "\n"))
		}
		inputField.SetText("")
	}

	inputField.OnSubmitted = func(_ string) {
		sendInput()
	}

	sendBtn := widget.NewButton("Send", sendInput)
	inputContainer := container.NewBorder(nil, nil, nil, sendBtn, inputField)

	closeBtn := widget.NewButton("Close", func() {
		a.Quit()
	})
	closeBtn.Disable() // Disabled during upgrading

	w.SetContent(container.NewBorder(
		container.NewVBox(title, progress),
		container.NewVBox(inputContainer, closeBtn),
		nil,
		nil,
		scrollLog,
	))

	go func() {
		brewPath := getBrewPath()
		cmd := exec.Command(brewPath, "upgrade", "-g", "-y")

		var err error
		cmdStdin, err = cmd.StdinPipe()
		if err != nil {
			logEntry.SetText("Error getting stdin: " + err.Error())
			closeBtn.Enable()
			progress.Stop()
			return
		}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			logEntry.SetText("Error getting stdout: " + err.Error())
			closeBtn.Enable()
			progress.Stop()
			return
		}
		cmd.Stderr = cmd.Stdout // combine stderr and stdout

		if err := cmd.Start(); err != nil {
			logEntry.SetText("Error starting upgrade: " + err.Error())
			closeBtn.Enable()
			progress.Stop()
			return
		}

		// Buffer to read stdout/stderr in real-time
		buf := make([]byte, 1024)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				logEntry.SetText(logEntry.Text + chunk)
				scrollLog.ScrollToBottom()
			}
			if err != nil {
				break
			}
		}

		// Wait for command completion
		err = cmd.Wait()
		progress.Stop()
		closeBtn.Enable()

		if err != nil {
			logEntry.SetText(logEntry.Text + "\n\nUpgrade failed: " + err.Error())
			title.SetText("Upgrade Failed!")
		} else {
			logEntry.SetText(logEntry.Text + "\n\nUpgrade completed successfully!")
			title.SetText("Upgrade Complete!")
			// Auto-quit after 10 seconds if successful
			go func() {
				time.Sleep(10 * time.Second)
				a.Quit()
			}()
		}
	}()
}

/*
noUpdates messages "no updates" with a countdown and close window after the
countdown went off.
*/
func noUpdates(a fyne.App) fyne.CanvasObject {
	countdown := 15
	str := binding.NewString()

	quitBtn := widget.NewButton("Close Now", func() {
		a.Quit()
	})

	go func() {
		for countdown > 0 {
			str.Set("No updates found. \nWindow closes in " + strconv.Itoa(countdown))
			countdown--
			time.Sleep(time.Second)
		}
		a.Quit()
	}()

	return container.NewBorder(
		widget.NewLabelWithStyle("Homebrew Update Helper", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		quitBtn,
		nil,
		nil,
		container.NewCenter(widget.NewLabelWithData(str)),
	)
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
