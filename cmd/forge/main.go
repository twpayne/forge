package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/pflag"
	"github.com/twpayne/go-shell"
	"golang.org/x/sys/unix"

	"github.com/twpayne/forge/pkg/forge"
)

func run() error {
	execShell := pflag.BoolP("shell", "s", false, "exec shell in working directory")
	goDoc := pflag.BoolP("doc", "d", false, "open pkg.go.dev documentation")
	web := pflag.BoolP("web", "w", false, "open home page")
	pflag.Parse()
	if pflag.NArg() != 1 {
		return fmt.Errorf("expected exactly 1 argument, got %d", pflag.NArg())
	}
	pattern := pflag.Arg(0)

	reposCache := forge.NewReposersCache()
	repos, err := reposCache.FindRepos(pattern)
	switch {
	case err != nil:
		return err
	case len(repos) == 0:
		return fmt.Errorf("%s: not found", pattern)
	case len(repos) > 2:
		repoNames := make([]string, len(repos))
		for i, repo := range repos {
			repoNames[i] = repo.Name
		}
		return fmt.Errorf("%s: ambiguous pattern (%s)", pattern, strings.Join(repoNames, ", "))
	}
	repo := repos[0]

	var url string
	var chdir string
	var execArgv []string

	switch {
	case *execShell:
		currentUserShell, _ := shell.CurrentUserShell()
		chdir = repo.WorkingDir
		execArgv = []string{currentUserShell}
	case *goDoc:
		url = repo.PkgGoDevURL()
	case *web:
		url = repo.URL()
	default:
		execArgv = append([]string{"code"}, repo.VSCodeOpenArgs...)
	}

	if chdir != "" {
		if err := os.Chdir(chdir); err != nil {
			return err
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

	if execArgv != nil {
		execArgv0, err := exec.LookPath(execArgv[0])
		if err != nil {
			return err
		}
		return unix.Exec(execArgv0, execArgv, os.Environ())
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
