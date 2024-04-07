package main

// FIXME add named forges (e.g. github as an alias for github.com, as done for remotes)
// FIXME add support for per-forge source dirs, i.e. src/src/go

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/spf13/pflag"
	"github.com/twpayne/go-shell"
	"golang.org/x/sys/unix"

	"github.com/twpayne/forge"
)

func run() error {
	config, err := forge.NewDefaultConfig()
	if err != nil {
		return err
	}

	create := pflag.BoolP("create", "c", false, "create repo")
	dryRun := pflag.BoolP("dry-run", "n", false, "dry run")
	execShell := pflag.BoolP("shell", "s", false, "exec shell instead of editor")
	goDoc := pflag.BoolP("doc", "d", false, "open pkg.go.dev documentation in web browser")
	verbose := pflag.BoolP("verbose", "v", false, "verbose")
	web := pflag.BoolP("web", "w", false, "open repo in web browser")
	pflag.Parse()

	if pflag.NArg() != 1 {
		return fmt.Errorf("syntax: %s [flags] [[forge/]user/]repo[@remote]|alias", path.Base(os.Args[0]))
	}

	repo, err := config.ParseRepoFromArg(pflag.Arg(0))
	if err != nil {
		return err
	}

	var url string
	var chdir string
	var execArgv []string

	if repo.Remote == "" && !*goDoc && !*web {
		switch _, err := os.Stat(repo.RepoDir); {
		case err == nil:
		case errors.Is(err, fs.ErrNotExist):
			var cmds []*exec.Cmd
			if *create {
				cmds = repo.InitWithRemoteCmds(config)
			} else {
				cmds = repo.CloneCmds(config)
			}
			for _, cmd := range cmds {
				if *verbose {
					fmt.Println(strings.Join(cmd.Args, " "))
				}
				if !*dryRun {
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						return err
					}
				}
			}
		default:
			return err
		}
	}

	switch {
	case *execShell:
		currentUserShell, _ := shell.CurrentUserShell()
		chdir = repo.RepoDir
		execArgv = []string{currentUserShell}
	case *goDoc:
		url = repo.PkgGoDevURL()
	case *web:
		url = repo.URL()
	default:
		if repo.Remote == "" {
			execArgv = []string{config.Editor, repo.RepoDir}
		} else {
			execArgv = []string{config.Editor, "--folder-uri", repo.VSCodeRemoteURL()}
		}
	}

	if url != "" {
		switch runtime.GOOS {
		case "darwin":
			execArgv = []string{"open", url}
		default:
			execArgv = []string{"xdg-open", url}
		}
	}

	if chdir != "" {
		if *verbose {
			fmt.Println("cd", chdir)
		}
		if !*dryRun {
			if err := os.Chdir(chdir); err != nil {
				return err
			}
		}
	}

	if execArgv != nil {
		execArgv0, err := exec.LookPath(execArgv[0])
		if err != nil {
			return err
		}
		if *verbose {
			fmt.Println("exec", execArgv0, strings.Join(execArgv[1:], " "))
		}
		if !*dryRun {
			return unix.Exec(execArgv0, execArgv, os.Environ())
		}
	}

	return nil
}

func main() {
	switch err, exitError := run(), (&exec.ExitError{}); {
	case errors.As(err, &exitError):
		os.Exit(exitError.ExitCode())
	case err != nil:
		fmt.Println(err)
		os.Exit(1)
	}
}
