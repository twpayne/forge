package main

// FIXME Open VS Code remotes does not seem to work when launched via Hammerspoon

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/twpayne/forge"
)

var errInvalid = errors.New("invalid")

// A returnButton is a Button that can also be tapped by pressing the Return
// key.
type returnButton struct {
	widget.Button
}

func newReturnButton(text string, onTapped func()) *returnButton {
	returnButton := &returnButton{}
	returnButton.ExtendBaseWidget(returnButton)
	returnButton.Text = text
	returnButton.OnTapped = onTapped
	return returnButton
}

func (b *returnButton) TypedKey(keyEvent *fyne.KeyEvent) {
	if keyEvent.Name == fyne.KeyReturn {
		b.Tapped(nil)
	} else {
		b.Button.TypedKey(keyEvent)
	}
}

type entryWithShortcuts struct {
	widget.Entry
	KeyEvents              map[fyne.KeyName]func()
	DesktopCustomShortcuts map[desktop.CustomShortcut]func()
}

func newEntryWithShortcuts() *entryWithShortcuts {
	entry := &entryWithShortcuts{
		KeyEvents:              make(map[fyne.KeyName]func()),
		DesktopCustomShortcuts: make(map[desktop.CustomShortcut]func()),
	}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *entryWithShortcuts) TypedKey(keyEvent *fyne.KeyEvent) {
	if f := e.KeyEvents[keyEvent.Name]; f != nil {
		f()
	} else {
		e.Entry.TypedKey(keyEvent)
	}
}

func (e *entryWithShortcuts) TypedShortcut(shortcut fyne.Shortcut) {
	if desktopCustomShortcut, ok := shortcut.(*desktop.CustomShortcut); ok {
		if f := e.DesktopCustomShortcuts[*desktopCustomShortcut]; f != nil {
			f()
			return
		}
	}
	e.Entry.TypedShortcut(shortcut)
}

func main() {
	app := app.New()
	doAndQuit := func(f func() bool) func() {
		return func() {
			if f() {
				app.Quit()
			}
		}
	}

	config, err := forge.NewDefaultConfig()
	if err != nil {
		// FIXME add Enter shortcut to close
		w := app.NewWindow("Forge Error")
		closeButton := widget.NewButton("Quit", app.Quit)
		w.SetContent(container.NewVBox(
			container.NewHBox(
				widget.NewIcon(theme.ErrorIcon()),
				widget.NewLabel(err.Error()),
			),
			closeButton,
		))
		w.Canvas().Focus(closeButton)
		w.ShowAndRun()
		return
	}

	argBinding := binding.NewString()
	getRepo := func() (*forge.Repo, bool) {
		argStr, err := argBinding.Get()
		if err != nil {
			log.Println(err)
			return nil, false
		}
		if !forge.ArgRx.MatchString(argStr) {
			log.Printf("%s: invalid", argBinding)
			return nil, false
		}
		repo, err := config.ParseRepoFromArg(argStr)
		if err != nil {
			log.Println(err)
			return nil, false
		}
		return repo, true
	}

	openVSCode := func() bool {
		repo, ok := getRepo()
		if !ok {
			return false
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			script := fmt.Sprintf(`tell application "Visual Studio Code" to open %q`, repo.RepoDir)
			cmd = exec.Command("osascript", "-e", script)
		default:
			if repo.Remote == "" {
				cmd = exec.Command(config.Editor, repo.RepoDir)
			} else {
				cmd = exec.Command(config.Editor, "--folder-uri", repo.VSCodeRemoteURL())
			}
		}
		if err := cmd.Run(); err != nil {
			log.Println(err)
			return false
		}
		return true
	}

	openShell := func() bool {
		repo, ok := getRepo()
		if !ok {
			return false
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			script := strings.Join([]string{
				`tell application "iTerm2"`,
				`  set newWindow to (create window with default profile)`,
				`  tell current session of newWindow`,
				`	  write text "cd ` + repo.RepoDir + `"`,
				`  end tell`,
				`end tell`,
			}, "\n")
			cmd = exec.Command("osascript", "-e", script)
		default:
			cmd = exec.Command("gnome-terminal", "--working-directory", repo.RepoDir)
		}
		if err := cmd.Run(); err != nil {
			log.Println(err)
			return false
		}
		return true
	}

	openURL := func(urlStr string) bool {
		url, err := url.Parse(urlStr)
		if err != nil {
			log.Println(err)
			return false
		}
		if err := app.OpenURL(url); err != nil {
			log.Println(err)
			return false
		}
		return true
	}

	openPkgGoDev := func() bool {
		repo, ok := getRepo()
		if !ok {
			return false
		}
		if !openURL(repo.PkgGoDevURL()) {
			return false
		}
		return true
	}

	openWebsite := func() bool {
		repo, ok := getRepo()
		if !ok {
			return false
		}
		if !openURL(repo.URL()) {
			return false
		}
		return true
	}

	window := app.NewWindow("Forge")

	repoEntry := newEntryWithShortcuts()
	repoEntry.Bind(argBinding)
	repoEntry.PlaceHolder = forge.ArgPlaceHolder
	repoEntry.Validator = func(text string) error {
		if !forge.ArgRx.MatchString(text) {
			return errInvalid
		}
		return nil
	}
	repoEntry.KeyEvents[fyne.KeyEscape] = app.Quit
	repoEntry.KeyEvents[fyne.KeyReturn] = doAndQuit(openVSCode)
	for keyName, f := range map[fyne.KeyName]func(){
		fyne.KeyC: doAndQuit(openVSCode),
		fyne.KeyS: doAndQuit(openShell),
		fyne.KeyW: doAndQuit(openWebsite),
		fyne.KeyP: doAndQuit(openPkgGoDev),
	} {
		desktopCustomShortcut := desktop.CustomShortcut{
			KeyName:  keyName,
			Modifier: fyne.KeyModifierAlt,
		}
		repoEntry.DesktopCustomShortcuts[desktopCustomShortcut] = f
	}

	window.SetContent(container.NewVBox(
		repoEntry,
		container.NewHBox(
			newReturnButton("Code", doAndQuit(openVSCode)),
			widget.NewButton("Shell", doAndQuit(openShell)),
			widget.NewButton("Website", doAndQuit(openWebsite)),
			widget.NewButton("pkg.go.dev", doAndQuit(openPkgGoDev)),
		),
	))

	window.Canvas().Focus(repoEntry)
	window.CenterOnScreen()
	window.ShowAndRun()
}
