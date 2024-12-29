package forge

import (
	_ "embed"
	"strings"

	"github.com/sahilm/fuzzy"
)

//go:embed list-repos.sh
var listReposSh string

type reposSource []*Repo

func (s reposSource) Len() int            { return len(s) }
func (s reposSource) String(i int) string { return s[i].Name }

type ReposersCache struct {
	localRepos  []*Repo
	reposByHost map[string][]*Repo
}

func NewReposersCache() *ReposersCache {
	return &ReposersCache{
		reposByHost: make(map[string][]*Repo),
	}
}

func (c *ReposersCache) FindRepo(arg string) (*Repo, error) {
	var repos []*Repo
	var pattern string
	if host, patternStr, ok := strings.Cut(arg, ":"); ok {
		if _, ok := c.reposByHost[host]; !ok {
			repos, err := NewRemoteSSH(host).Repos()
			if err != nil {
				return nil, err
			}
			c.reposByHost[host] = repos
		}
		pattern = patternStr
		repos = c.reposByHost[host]
	} else {
		pattern = arg
		if c.localRepos == nil {
			var err error
			c.localRepos, err = NewLocal().Repos()
			if err != nil {
				return nil, err
			}
		}
		repos = c.localRepos
	}
	matches := fuzzy.FindFrom(pattern, reposSource(repos))
	if len(matches) == 0 {
		return nil, nil
	}
	for _, repo := range repos {
		if repo.Name == matches[0].Str {
			return repo, nil
		}
	}
	return nil, nil
}

func getNameAndWorkingDir(dir string) (name string, workingDir string) {
	components := strings.Split(dir, "/")
	if components[len(components)-1] == "" {
		components = components[:len(components)-1]
	}
	if components[len(components)-1] == ".git" {
		components = components[:len(components)-1]
	}
	name = strings.Join(components[4:], "/")
	workingDir = strings.Join(components, "/")
	return
}
