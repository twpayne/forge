package forge

import (
	"cmp"
	_ "embed"
	"slices"
	"strings"

	"github.com/sahilm/fuzzy"
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
	repos, pattern, err := c.loadRepos(arg)
	if err != nil {
		return nil, err
	}
	if pattern == "" {
		slices.SortFunc(repos, func(a, b *Repo) int {
			return cmp.Compare(a.Name, b.Name)
		})
		return repos, nil
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
	return matchingRepos, nil
}

func (c *ReposersCache) FuzzyFindRepos(arg string) ([]*Repo, error) {
	repos, pattern, err := c.loadRepos(arg)
	if err != nil {
		return nil, err
	}
	matches := fuzzy.FindFrom(pattern, reposSource(repos))
	if len(matches) == 0 {
		return nil, nil
	}
	return repos[matches[0].Index : matches[0].Index+1], nil
}

func (c *ReposersCache) loadRepos(arg string) ([]*Repo, string, error) {
	var repos []*Repo
	var pattern string
	if host, patternStr, ok := strings.Cut(arg, ":"); ok {
		if _, ok := c.reposByHost[host]; !ok {
			repos, err := NewRemoteSSH(host).Repos()
			if err != nil {
				return nil, "", err
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
				return nil, "", err
			}
		}
		repos = c.localRepos
	}
	return repos, pattern, nil
}

type reposSource []*Repo

func (s reposSource) Len() int            { return len(s) }
func (s reposSource) String(i int) string { return s[i].Name }

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
