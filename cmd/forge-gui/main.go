package main

// FIXME find out how to open VSCode remote SSH URLs on darwin
// FIXME find out how to open a terminal on darwin

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/twpayne/forge/pkg/forge"
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

	patternBinding := binding.NewString()

	reposersCache := forge.NewReposersCache()
	getRepo := func() *forge.Repo {
		pattern, err := patternBinding.Get()
		if err != nil {
			fmt.Println(err)
			return nil
		}
		switch repos, err := reposersCache.FindRepos(pattern); {
		case err != nil:
			return nil
		case len(repos) == 1:
			return repos[0]
		default:
			return nil
		}
	}

	openVSCode := func() bool {
		repo := getRepo()
		if repo == nil {
			return false
		}
		var cmd *exec.Cmd
		switch {
		case runtime.GOOS == "darwin" && repo.Host == "":
			script := fmt.Sprintf(`tell application "Visual Studio Code" to open %q`, repo.WorkingDir)
			cmd = exec.Command("osascript", "-e", script)
		case runtime.GOOS != "darwin":
			cmd = exec.Command("code", repo.VSCodeOpenArgs...)
		default:
			return false
		}
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}

	openShell := func() bool {
		repo := getRepo()
		if repo == nil {
			return false
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("wezterm", "cli", "--spawn", repo.WorkingDir) // FIXME
		default:
			cmd = exec.Command("gnome-terminal", "--working-directory", repo.WorkingDir)
		}
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}

	openURL := func(urlStr string) bool {
		url, err := url.Parse(urlStr)
		if err != nil {
			fmt.Println(err)
			return false
		}
		if err := app.OpenURL(url); err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}

	openPkgGoDev := func() bool {
		repo := getRepo()
		if repo == nil {
			return false
		}
		if !openURL(repo.PkgGoDevURL()) {
			return false
		}
		return true
	}

	openWebsite := func() bool {
		repo := getRepo()
		if repo == nil {
			return false
		}
		if !openURL(repo.URL()) {
			return false
		}
		return true
	}

	window := app.NewWindow("Forge")

	repoEntry := newEntryWithShortcuts()
	repoEntry.Bind(patternBinding)
	repoEntry.Validator = func(text string) error {
		if repo := getRepo(); repo == nil {
			return errors.New("no match")
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
