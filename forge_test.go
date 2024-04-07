package forge

import (
	"strconv"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestArgRx(t *testing.T) {
	for i, tc := range []struct {
		arg     string
		matches map[string]string
	}{
		{
			arg: "repo",
			matches: map[string]string{
				"repo": "repo",
			},
		},
		{
			arg: "user/repo",
			matches: map[string]string{
				"repo": "repo",
				"user": "user",
			},
		},
		{
			arg: "forge/user/repo",
			matches: map[string]string{
				"forge": "forge",
				"repo":  "repo",
				"user":  "user",
			},
		},
		{
			arg: "forge/user/repo@remote",
			matches: map[string]string{
				"forge":  "forge",
				"repo":   "repo",
				"user":   "user",
				"remote": "remote",
			},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			matches := argRx.FindStringSubmatch(tc.arg)
			assert.NotZero(t, matches)
			for subexpName, subexpValue := range tc.matches {
				assert.Equal(t, subexpValue, matches[argRx.SubexpIndex(subexpName)])
			}
		})
	}
}
