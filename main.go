package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/twpayne/go-shell"
	"golang.org/x/sys/unix"
)

const (
	defaultEditor    = "code"
	defaultForge     = "github.com"
	defaultSourceDir = "src"
	defaultUser      = "twpayne"
)

func runCommand(nameAndArgs ...string) error {
	name, args := nameAndArgs[0], nameAndArgs[1:]
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommands(commands [][]string) error {
	for _, nameAndArgs := range commands {
		if err := runCommand(nameAndArgs...); err != nil {
			return err
		}
	}
	return nil
}

func runMain() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	create := pflag.BoolP("create", "c", false, "create repo")
	srcDir := pflag.String("source", filepath.Join(homeDir, defaultSourceDir), "source directory")
	editor := pflag.StringP("editor", "e", defaultEditor, "editor")
	execShell := pflag.BoolP("shell", "s", false, "exec shell instead of editor")
	pflag.Parse()

	if pflag.NArg() != 1 {
		return fmt.Errorf("syntax: %s [[forge/]user/]repo", filepath.Base(os.Args[0]))
	}

	var forge, user, repo string
	switch components := strings.SplitN(pflag.Arg(0), "/", 3); len(components) {
	case 1:
		forge, user, repo = defaultForge, defaultUser, components[0]
	case 2:
		forge, user, repo = defaultForge, components[0], components[1]
	case 3:
		forge, user, repo = components[0], components[1], components[2]
	}

	repoDir := filepath.Join(*srcDir, forge, user, repo)

	switch _, err := os.Stat(repoDir); {
	case errors.Is(err, fs.ErrNotExist):
		var repoURL string
		if user == defaultUser {
			repoURL = "git@" + forge + ":" + user + "/" + repo + ".git"
		} else {
			repoURL = "https://" + forge + "/" + user + "/" + repo + ".git"
		}
		var commands [][]string
		if *create {
			commands = [][]string{
				{"git", "init", repoDir},
				{"git", "remote", "add", "origin", repoURL},
			}
		} else {
			commands = [][]string{
				{"git", "clone", repoURL, repoDir},
			}
		}
		if err := runCommands(commands); err != nil {
			return err
		}
	case err != nil:
		return err
	}

	if err := os.Chdir(repoDir); err != nil {
		return err
	}

	var argv []string
	if *execShell {
		currentUserShell, _ := shell.CurrentUserShell()
		argv = []string{currentUserShell}
	} else {
		argv = []string{*editor, repoDir}
	}
	argv0, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	return unix.Exec(argv0, argv, os.Environ())
}

func main() {
	switch err, exitError := runMain(), (&exec.ExitError{}); {
	case errors.As(err, &exitError):
		os.Exit(exitError.ExitCode())
	case err != nil:
		fmt.Println(err)
		os.Exit(1)
	}
}
