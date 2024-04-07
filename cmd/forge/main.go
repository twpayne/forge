package main

// FIXME add named forges (e.g. github as an alias for github.com, as done for remotes)
// FIXME add support for per-forge source dirs, i.e. src/src/go

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
	"github.com/twpayne/go-shell"
	"github.com/twpayne/go-xdg/v6"
	"golang.org/x/sys/unix"
)

var argRx = regexp.MustCompile(`\A((?:(?P<forge>[^/]+)/)?(?:(?P<user>[^/]+)/))?(?P<repo>[^/@]+)(?:@(?P<remote>[^/]+))?`) // FIXME use .*? instead of [^/] and [^/@]

type Config struct {
	User      string                  `toml:"user"`
	Editor    string                  `toml:"editor"`
	Forge     string                  `toml:"forge"`
	SourceDir string                  `toml:"sourceDir"`
	Remotes   map[string]RemoteConfig `toml:"remote"`
	Aliases   map[string]AliasConfig  `toml:"alias"`
}

type RemoteConfig struct {
	Hostname  string `toml:"hostname"`
	SourceDir string `toml:"sourceDir"`
}

type AliasConfig struct {
	Forge   string `toml:"forge"`
	User    string `toml:"user"`
	Repo    string `toml:"repo"`
	RepoDir string `toml:"repoDir"`
	Remote  string `toml:"remote"`
}

type Repo struct {
	Forge     string
	User      string
	Repo      string
	RepoDir   string
	Remote    string
	SourceDir string
	Hostname  string
}

func newConfigFromFile(name string) (*Config, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) parseRepoFromArg(arg string) (*Repo, error) {
	defaultUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	var repo Repo
	if alias, ok := c.Aliases[arg]; ok {
		repo = Repo{
			RepoDir: alias.RepoDir,
			Forge:   firstNonZero(alias.Forge, c.Forge, "github.com"),
			User:    firstNonZero(alias.User, c.User, defaultUser.Username),
			Repo:    alias.Repo,
			Remote:  alias.Remote,
		}
	} else {
		match := argRx.FindStringSubmatch(arg)
		if len(match) == 0 {
			return nil, fmt.Errorf("%s: invalid argument", arg)
		}
		repo = Repo{
			Forge:  firstNonZero(match[argRx.SubexpIndex("forge")], c.Forge, "github.com"),
			User:   firstNonZero(match[argRx.SubexpIndex("user")], c.User, defaultUser.Username),
			Repo:   match[argRx.SubexpIndex("repo")],
			Remote: match[argRx.SubexpIndex("remote")],
		}
	}

	if remoteConfig, ok := c.Remotes[repo.Remote]; ok {
		repo.SourceDir = firstNonZero(remoteConfig.SourceDir, c.SourceDir)
		repo.Hostname = remoteConfig.Hostname
	}

	if repo.RepoDir == "" {
		sourceDir := firstNonZero(repo.SourceDir, c.SourceDir)
		repo.RepoDir = path.Join(sourceDir, repo.Forge, repo.User, repo.Repo)
	}

	return &repo, nil
}

func (r *Repo) goDocURL() string {
	// FIXME add package version
	return "https://pkg.go.dev/" + r.Forge + "/" + r.User + "/" + r.Repo
}

func (r *Repo) url() string {
	return "https://" + r.Forge + "/" + r.User + "/" + r.Repo
}

func (r *Repo) vsCodeRemoteURL() string {
	return "vscode-remote://ssh-remote+" + r.Hostname + r.RepoDir
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

func run() error {
	bds, err := xdg.NewBaseDirectorySpecification()
	if err != nil {
		return err
	}

	var config *Config
FOR:
	for _, configDir := range bds.ConfigDirs {
		name := path.Join(configDir, "forge", "forge.toml")
		switch c, err := newConfigFromFile(name); {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return err
		default:
			config = c
			break FOR
		}
	}
	if config == nil {
		return errors.New("no config")
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

	repo, err := config.parseRepoFromArg(pflag.Arg(0))
	if err != nil {
		return err
	}

	if repo.Remote == "" && repo.User == "_" && repo.Repo != "" {
		var candidateUsers []string
		forgeDirEntries, err := os.ReadDir(path.Join(config.SourceDir, repo.Forge))
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
			switch fileInfo, err := os.Stat(path.Join(config.SourceDir, repo.Forge, candidateUser, repo.Repo)); {
			case errors.Is(err, fs.ErrNotExist):
			case err != nil:
				return err
			case fileInfo.IsDir():
				candidateUsers = append(candidateUsers, candidateUser)
			}
		}
		switch len(candidateUsers) {
		case 0:
			return fmt.Errorf("%s/_/%s: no user found", repo.Forge, repo.Repo)
		case 1:
			repo.User = candidateUsers[0]
			repo.RepoDir = path.Join(config.SourceDir, repo.Forge, repo.User, repo.Repo)
		default:
			return fmt.Errorf("%s/_/%s: multiple users found: %s", repo.Forge, repo.Repo, strings.Join(candidateUsers, ", "))
		}
	}

	var cmds []*exec.Cmd
	var url string
	var chdir string
	var execArgv []string

	if repo.Remote == "" && !*goDoc && !*web {
		switch _, err := os.Stat(repo.RepoDir); {
		case err == nil:
		case errors.Is(err, fs.ErrNotExist):
			var repoURL string
			if repo.User == config.User {
				repoURL = "git@" + repo.Forge + ":" + repo.User + "/" + repo.Repo + ".git"
			} else {
				repoURL = "https://" + repo.Forge + "/" + repo.User + "/" + repo.Repo + ".git"
			}
			if *create {
				cmds = []*exec.Cmd{
					exec.Command("git", "init", repo.RepoDir),
					exec.Command("git", "remote", "add", "origin", repoURL),
				}
			} else {
				cmds = []*exec.Cmd{
					exec.Command("git", "clone", repoURL, repo.RepoDir),
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
		url = repo.goDocURL()
	case *web:
		url = repo.url()
	default:
		if repo.Remote == "" {
			execArgv = []string{config.Editor, repo.RepoDir}
		} else {
			execArgv = []string{config.Editor, "--folder-uri", repo.vsCodeRemoteURL()}
		}
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
