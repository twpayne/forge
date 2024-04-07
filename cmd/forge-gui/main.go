package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var (
	argRx      = regexp.MustCompile(`\A((?:(?P<forge>[^/]+)/)?(?:(?P<user>[^/]+)/))?(?P<repo>[^/@]+)(?:@(?P<remote>[^/]+))?`) // FIXME use .*? instead of [^/] and [^/@]
	errInvalid = errors.New("invalid")
)

func main() {
	a := app.New()

	w := a.NewWindow("Forge")

	repoEntry := widget.NewEntry()
	repoEntry.PlaceHolder = "[[forge/]user/]repo[@remote]|alias"
	repoEntry.Validator = func(text string) error {
		if !argRx.MatchString(text) {
			return errInvalid
		}
		return nil
	}
	repoEntry.OnSubmitted = func(text string) {
		fmt.Printf("code %s\n", text)
		w.Close()
	}

	openVSCodeAndQuit := func() {
		fmt.Printf("code %s\n", repoEntry.Text)
		w.Close()
	}

	ctrlC := &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierControl}
	w.Canvas().AddShortcut(ctrlC, func(shortcut fyne.Shortcut) {
		log.Println("We tapped Ctrl+C")
	})

	w.SetContent(container.NewVBox(
		repoEntry,
		container.NewHBox(
			widget.NewButton("Code", openVSCodeAndQuit),
			widget.NewButton("Shell", func() {
			}),
			widget.NewButton("Website", func() {
			}),
			widget.NewButton("Doc", func() {
			}),
		),
	))

	w.ShowAndRun()
}
