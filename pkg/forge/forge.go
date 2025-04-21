package forge

import (
	"cmp"
	_ "embed"
	"slices"
	"strings"
)

//go:embed list-repos.sh
var listReposSh string

type ReposersCache struct {
	localRepos  []*Repo
	reposByHost map[string][]*Repo
}

func NewReposersCache() *ReposersCache {
	return &ReposersCache{
		reposByHost: make(map[string][]*Repo),
	}
}

func (c *ReposersCache) FindRepos(arg string) ([]*Repo, error) {
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
	return findRepos(repos, pattern), nil
}

func findRepos(repos []*Repo, pattern string) []*Repo {
	if pattern == "" {
		slices.SortFunc(repos, func(a, b *Repo) int {
			return cmp.Compare(a.Name, b.Name)
		})
		return repos
	}
	patternComponents := strings.Split(pattern, "/")
	var matchingRepos []*Repo
	for _, repo := range repos {
		repoComponents := strings.Split(repo.Name, "/")
		if len(repoComponents) < len(patternComponents) {
			continue
		}
		if slices.Equal(repoComponents[len(repoComponents)-len(patternComponents):], patternComponents) {
			matchingRepos = append(matchingRepos, repo)
		}
	}
	slices.SortFunc(matchingRepos, func(a, b *Repo) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return matchingRepos
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
