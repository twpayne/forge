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
	gitEntries := strings.Split(string(output), "\x00")
	repos := make([]*Repo, 0, len(gitEntries))
	for _, gitEntry := range gitEntries {
		if gitEntry == "" {
			continue
		}
		name, workingDir := getNameAndWorkingDir(gitEntry)
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
