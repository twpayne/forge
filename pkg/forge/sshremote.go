package forge

import (
	"bytes"
	_ "embed"
	"errors"
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
	// The following is a horrible hack. For some reason, executing sh via ssh
	// on CachyOS invokes fish, not sh. Since it's impossible to write a script
	// that is compatible with both sh and fish, if we get an error and it looks
	// like a fish error, then try again but using the fish shell with a fish
	// script.
	if exitError, ok := errors.AsType[*exec.ExitError](err); ok && bytes.HasPrefix(exitError.Stderr, []byte("fish:")) {
		output, err = exec.Command("ssh", r.host, "fish", "-c", listReposFish).Output()
	}
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
			ZedOpenArgs:    []string{"ssh://" + r.host + workingDir},
		}
		repos = append(repos, repo)
	}
	return repos, nil
}
