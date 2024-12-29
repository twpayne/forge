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
	dirs := strings.Split(string(output), "\x00")
	repos := make([]*Repo, 0, len(dirs))
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		name, workingDir := getNameAndWorkingDir(dir)
		repo := &Repo{
			Name:           name,
			WorkingDir:     workingDir,
			VSCodeOpenArgs: []string{workingDir},
		}
		repos = append(repos, repo)
	}
	return repos, nil

}
