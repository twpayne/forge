package main

// FIXME add named forges (e.g. github as an alias for github.com, as done for remotes)
// FIXME should we read .config/forge/forge.toml on remote machine to get sourceDir?
// FIXME add support for per-forge source dirs, i.e. src/src/go

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
	"github.com/twpayne/go-shell"
	"github.com/twpayne/go-xdg/v6"
	"golang.org/x/sys/unix"
)

type Config struct {
	User      string            `toml:"user"`
	Editor    string            `toml:"editor"`
	Forge     string            `toml:"forge"`
	SourceDir string            `toml:"sourceDir"`
	Remotes   map[string]Remote `toml:"remote"`
	Aliases   map[string]Alias  `toml:"alias"`
}

type Remote struct {
	Hostname  string `toml:"hostname"`
	SourceDir string `toml:"sourceDir"`
}

type Alias struct {
	Forge   string `toml:"forge"`
	User    string `toml:"user"`
	Repo    string `toml:"repo"`
	RepoDir string `toml:"repoDir"`
	Remote  string `toml:"remote"`
}

var argRx = regexp.MustCompile(`\A((?:(?P<forge>[^/]+)/)?(?:(?P<user>[^/]+)/))?(?P<repo>[^/@]+)(?:@(?P<remote>[^/]+))?`) // FIXME use .*? instead of [^/] and [^/@]

func runMain() error {
	bds, err := xdg.NewBaseDirectorySpecification()
	if err != nil {
		return err
	}

	var config Config
FOR:
	for _, configDir := range bds.ConfigDirs {
		switch configFile, err := os.Open(filepath.Join(configDir, "forge", "forge.toml")); {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return err
		default:
			defer configFile.Close()
			if err := toml.NewDecoder(configFile).Decode(&config); err != nil {
				return err
			}
			break FOR
		}
	}
	if reflect.ValueOf(config).IsZero() {
		return errors.New("no config")
	}

	create := pflag.BoolP("create", "c", false, "create repo")
	sourceDir := pflag.StringP("source", "S", config.SourceDir, "source directory")
	editor := pflag.StringP("editor", "e", config.Editor, "editor")
	execShell := pflag.BoolP("shell", "s", false, "exec shell instead of editor")
	verbose := pflag.BoolP("verbose", "v", false, "verbose")
	pflag.Parse()
	if pflag.NArg() != 1 {
		return fmt.Errorf("syntax: %s [flags] [[forge/]user/]repo[@remote]|alias", filepath.Base(os.Args[0]))
	}

	if *verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	arg := pflag.Arg(0)
	var repoDir, forge, user, repo, remote string
	if alias, ok := config.Aliases[arg]; ok {
		repoDir = alias.RepoDir
		forge = alias.Forge
		user = alias.User
		repo = alias.Repo
		remote = alias.Remote
		// FIXME allow overriding of sourceDir
	} else {
		match := argRx.FindStringSubmatch(arg)
		if len(match) == 0 {
			return fmt.Errorf("%s: invalid argument", arg)
		}
		forge = firstNonZero(match[argRx.SubexpIndex("forge")], config.Forge)
		user = firstNonZero(match[argRx.SubexpIndex("user")], config.User)
		repo = match[argRx.SubexpIndex("repo")]
		remote = match[argRx.SubexpIndex("remote")]
	}

	if remote != "" {
		var hostname string
		if remoteConfig, ok := config.Remotes[remote]; ok {
			*sourceDir = firstNonZero(remoteConfig.SourceDir, *sourceDir)
			hostname = firstNonZero(remoteConfig.Hostname, remote)
		}
		if repoDir == "" {
			repoDir = filepath.Join(*sourceDir, forge, user, repo)
		}
		folderURI := "vscode-remote://ssh-remote+" + hostname + repoDir
		return execArgv([]string{*editor, "--folder-uri", folderURI})
	}

	if forge != "" && user == "_" && repo != "" {
		var candidateUsers []string
		forgeDirEntries, err := os.ReadDir(filepath.Join(*sourceDir, forge))
		if err != nil {
			return err
		}
		for _, forgeDirEntry := range forgeDirEntries {
			if !forgeDirEntry.IsDir() {
				continue
			}
			candidateUser := forgeDirEntry.Name()
			if candidateUser == "." || candidateUser == ".." {
				continue
			}
			switch fileInfo, err := os.Stat(filepath.Join(*sourceDir, forge, candidateUser, repo)); {
			case errors.Is(err, fs.ErrNotExist):
			case err != nil:
				return err
			case fileInfo.IsDir():
				candidateUsers = append(candidateUsers, candidateUser)
			}
		}
		switch len(candidateUsers) {
		case 0:
			return fmt.Errorf("%s/_/%s: no user found", forge, repo)
		case 1:
			user = candidateUsers[0]
		default:
			return fmt.Errorf("%s/_/%s: multiple users found: %s", forge, repo, strings.Join(candidateUsers, ", "))
		}
	}

	if repoDir == "" {
		repoDir = filepath.Join(*sourceDir, forge, user, repo)
	}

	switch _, err := os.Stat(repoDir); {
	case errors.Is(err, fs.ErrNotExist):
		var repoURL string
		if user == config.User {
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
	return execArgv(argv)
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

func execArgv(argv []string) error {
	argv0, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	slog.Debug("exec", slog.String("argv0", argv0), slog.Any("argv", argv))
	return unix.Exec(argv0, argv, os.Environ())
}

func firstNonZero[E comparable](es ...E) E {
	var zero E
	for _, e := range es {
		if e != zero {
			return e
		}
	}
	return zero
}

func runCommand(nameAndArgs ...string) error {
	name, args := nameAndArgs[0], nameAndArgs[1:]
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	slog.Debug("run", slog.String("name", name), slog.Any("args", args))
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
