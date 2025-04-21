package forge

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestFindRepos(t *testing.T) {
	repos := []*Repo{
		{
			Name: "github.com/golang/go",
		},
		{
			Name: "github.com/halostatue/chezmoi",
		},
		{
			Name: "github.com/twpayne/chezmoi",
		},
		{
			Name: "github.com/twpayne/forge",
		},
	}
	for _, tc := range []struct {
		name          string
		pattern       string
		expectedNames []string
	}{
		{
			name: "empty",
			expectedNames: []string{
				"github.com/golang/go",
				"github.com/halostatue/chezmoi",
				"github.com/twpayne/chezmoi",
				"github.com/twpayne/forge",
			},
		},
		{
			name:          "exact_repo",
			pattern:       "forge",
			expectedNames: []string{"github.com/twpayne/forge"},
		},
		{
			name:          "exact_owner_repo",
			pattern:       "twpayne/forge",
			expectedNames: []string{"github.com/twpayne/forge"},
		},
		{
			name:          "ambiguous_repo",
			pattern:       "chezmoi",
			expectedNames: []string{"github.com/halostatue/chezmoi", "github.com/twpayne/chezmoi"},
		},
		{
			name:          "exact_owner_ambiguous_repo",
			pattern:       "halostatue/chezmoi",
			expectedNames: []string{"github.com/halostatue/chezmoi"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := findRepos(repos, tc.pattern)
			if tc.expectedNames == nil {
				assert.Zero(t, actual)
			} else {
				actualNames := make([]string, len(actual))
				for i, repo := range actual {
					actualNames[i] = repo.Name
				}
				assert.Equal(t, tc.expectedNames, actualNames)
			}
		})
	}
}
