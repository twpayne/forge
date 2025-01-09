# forge

forge is a simple utility to clone and open local and remote git repos.

forge is primarily designed for my personal workflow, but might be useful to
others who use VSCode, work on multiple GitHub projects, and use VSCode's Remote
SSH extension. There are certainly many bugs.

## Installation

Run:

```console
$ go install github.com/twpayne/forge/cmd/forge@latest
$ go install github.com/twpayne/forge/cmd/forge-gui@latest
```

## Command line interface

    forge [flags] [remote:]pattern

`pattern` can be either `repo`, `owner/repo`, or `forge/owner/repo`.

Other flags control the action taken:

| flag | Action |
| - | - |
| none | Open the working copy in VSCode |
| `-c` | Clone the repo if it does not exist |
| `-s` | Open the working copy in a shell |
| `-w` | Open the project's repo in your web browser |
| `-d` | Open the project's documentation on pkg.go.dev in your web browser |

## Graphical user interface

`forge-gui` is a simple GUI using [Fyne](https://fyne.io/). It is designed to be
launched from a shortcut key (I use `CapsLock+J` with [this Hammerspoon
config](https://github.com/twpayne/dotfiles/commit/68a9663f5ae52c7347bf6a063438e1f5a457182a)).

Shortcuts:

| Key                | Action                                                             |
| ------------------ | ------------------------------------------------------------------ |
| `Escape`           | Quit                                                               |
| `Enter` or `Alt+C` | Open the working copy in VSCode                                    |
| `Alt+S`            | Open the working copy in a shell                                   |
| `Alt+W`            | Open the project's repo in your web browser                        |
| `Alt+P`            | Open the project's documentation on pkg.go.dev in your web browser |

## License

MIT
