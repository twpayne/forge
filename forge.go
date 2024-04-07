package forge

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
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/twpayne/go-xdg/v6"
)

const defaultForge = "github.com"

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

func NewConfigFromFile(name string) (*Config, error) {
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

func NewDefaultConfig() (*Config, error) {
	bds, err := xdg.NewBaseDirectorySpecification()
	if err != nil {
		return nil, err
	}

	for _, configDir := range bds.ConfigDirs {
		name := path.Join(configDir, "forge", "forge.toml")
		switch config, err := NewConfigFromFile(name); {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return nil, err
		default:
			return config, nil
		}
	}

	return nil, fs.ErrNotExist
}

func (c *Config) ParseRepoFromArg(arg string) (*Repo, error) {
	defaultUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	var repo Repo
	if alias, ok := c.Aliases[arg]; ok {
		repo = Repo{
			RepoDir: alias.RepoDir,
			Forge:   firstNonZero(alias.Forge, c.Forge, defaultForge),
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
			Forge:  firstNonZero(match[argRx.SubexpIndex("forge")], c.Forge, defaultForge),
			User:   firstNonZero(match[argRx.SubexpIndex("user")], c.User, defaultUser.Username),
			Repo:   match[argRx.SubexpIndex("repo")],
			Remote: match[argRx.SubexpIndex("remote")],
		}

		if repo.Remote == "" && repo.User == "_" && repo.Repo != "" {
			var candidateUsers []string
			forgeDirEntries, err := os.ReadDir(path.Join(c.SourceDir, repo.Forge))
			if err != nil {
				return nil, err
			}
			for _, forgeDirEntry := range forgeDirEntries {
				if !forgeDirEntry.IsDir() {
					continue
				}
				candidateUser := forgeDirEntry.Name()
				if candidateUser == "." || candidateUser == ".." {
					continue
				}
				switch fileInfo, err := os.Stat(path.Join(c.SourceDir, repo.Forge, candidateUser, repo.Repo)); {
				case errors.Is(err, fs.ErrNotExist):
				case err != nil:
					return nil, err
				case fileInfo.IsDir():
					candidateUsers = append(candidateUsers, candidateUser)
				}
			}
			switch len(candidateUsers) {
			case 0:
				return nil, fmt.Errorf("%s/_/%s: no user found", repo.Forge, repo.Repo)
			case 1:
				repo.User = candidateUsers[0]
				repo.RepoDir = path.Join(c.SourceDir, repo.Forge, repo.User, repo.Repo)
			default:
				return nil, fmt.Errorf("%s/_/%s: multiple users found: %s", repo.Forge, repo.Repo, strings.Join(candidateUsers, ", "))
			}
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

func (r *Repo) CloneCmds(config *Config) []*exec.Cmd {
	return []*exec.Cmd{
		exec.Command("git", "clone", r.GitURL(config), r.RepoDir),
	}

}

func (r *Repo) GitURL(config *Config) string {
	if r.User == config.User {
		return "git@" + r.Forge + ":" + r.User + "/" + r.Repo + ".git"
	}
	return "https://" + r.Forge + "/" + r.User + "/" + r.Repo + ".git"
}

func (r *Repo) GoDocURL() string {
	// FIXME add package version
	return "https://pkg.go.dev/" + r.Forge + "/" + r.User + "/" + r.Repo
}

func (r *Repo) InitWithRemoteCmds(config *Config) []*exec.Cmd {
	return []*exec.Cmd{
		exec.Command("git", "init", r.RepoDir),
		exec.Command("git", "remote", "add", "origin", r.GitURL(config)),
	}
}

func (r *Repo) URL() string {
	return "https://" + r.Forge + "/" + r.User + "/" + r.Repo
}

func (r *Repo) VSCodeRemoteURL() string {
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
