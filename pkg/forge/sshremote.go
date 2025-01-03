package forge

import (
	_ "embed"
	"os/exec"
	"strings"
)

type SSHRemote struct {
	host string
}

func NewRemoteSSH(host string) *SSHRemote {
	return &SSHRemote{
		host: host,
	}
}

func (r *SSHRemote) Repos() ([]*Repo, error) {
	output, err := exec.Command("ssh", r.host, "sh", "-c", listReposSh).Output()
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
			Host:           r.host,
			WorkingDir:     workingDir,
			VSCodeOpenArgs: []string{"--folder-uri", "vscode-remote://ssh-remote+" + r.host + workingDir},
		}
		repos = append(repos, repo)
	}
	return repos, nil
}
