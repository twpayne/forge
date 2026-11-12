package forge

import (
	"os/exec"
	"strings"
)

type Local struct{}

func NewLocal() *Local {
	return &Local{}
}

func (l *Local) Repos() ([]*Repo, error) {
	output, err := exec.Command("sh", "-c", listReposSh).Output()
	if err != nil {
		return nil, err
	}
	gitEntries := strings.Split(string(output), "\x00")
	repos := make([]*Repo, 0, len(gitEntries))
	for _, gitEntry := range gitEntries {
		if gitEntry == "" {
			continue
		}
		name, workingDir := getNameAndWorkingDir(gitEntry)
		repo := &Repo{
			Name:           name,
			WorkingDir:     workingDir,
			VSCodeOpenArgs: []string{workingDir},
		}
		repos = append(repos, repo)
	}
	return repos, nil

}
