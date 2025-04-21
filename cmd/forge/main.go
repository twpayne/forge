package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/spf13/pflag"
	"github.com/twpayne/go-shell"
	"golang.org/x/sys/unix"

	"github.com/twpayne/forge/pkg/forge"
)

func run() error {
	clone := pflag.BoolP("clone", "c", false, "clone repo if it does not already exist")
	execShell := pflag.BoolP("shell", "s", false, "exec shell in working directory")
	list := pflag.BoolP("list", "l", false, "list repos")
	goDoc := pflag.BoolP("doc", "d", false, "open pkg.go.dev documentation")
	open := pflag.BoolP("open", "o", false, "open folder")
	web := pflag.BoolP("web", "w", false, "open home page")
	pflag.Parse()
	if pflag.NArg() != 1 && !*list {
		return fmt.Errorf("expected exactly 1 argument, got %d", pflag.NArg())
	}
	pattern := pflag.Arg(0)

	var repo *forge.Repo
	reposCache := forge.NewReposersCache()
	repos, err := reposCache.FindRepos(pattern)
	switch {
	case err != nil:
		return err
	case *list:
		for _, repo := range repos {
			fmt.Println(repo.WorkingDir)
		}
		return nil
	case len(repos) == 0:
		if !*clone {
			return fmt.Errorf("%s: not found", pattern)
		}
		components := strings.Split(pattern, "/")
		if len(components) < 1 || 3 < len(components) {
			return fmt.Errorf("%s: invalid pattern", pattern)
		}
		if len(components) < 2 {
			components = []string{"twpayne", components[0]}
		}
		if len(components) < 3 {
			components = []string{"github.com", components[0], components[1]}
		}
		userHomerDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		name := strings.Join(components, "/")
		workingDir := path.Join(userHomerDir, "src", name)
		repo = &forge.Repo{
			Name:           name,
			WorkingDir:     workingDir,
			VSCodeOpenArgs: []string{workingDir},
		}
		cmd := exec.Command("git", "clone", "https://"+repo.Name+".git", repo.WorkingDir)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	case len(repos) == 1:
		repo = repos[0]
	default:
		repoNames := make([]string, len(repos))
		for i, repo := range repos {
			repoNames[i] = repo.Name
		}
		return fmt.Errorf("%s: ambiguous pattern (%s)", pattern, strings.Join(repoNames, ", "))
	}

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
	case *open:
		execArgv = []string{"open", repo.WorkingDir}
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
