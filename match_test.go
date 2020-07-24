package codeowners

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	examples := []struct {
		name    string
		pattern string
		paths   map[string]bool
	}{
		{
			name:    "single-segment pattern",
			pattern: "foo",
			paths: map[string]bool{
				"foo":         true,
				"foo/":        true,
				"foo/bar":     true,
				"bar/foo":     true,
				"bar/foo/baz": true,
				"bar/baz":     false,
			},
		},
		{
			name:    "single-segment pattern with leading slash",
			pattern: "/foo",
			paths: map[string]bool{
				"foo":         true,
				"fool":        false,
				"foo/":        true,
				"foo/bar":     true,
				"bar/foo":     false,
				"bar/foo/baz": false,
				"bar/baz":     false,
			},
		},
		{
			name:    "single-segment pattern with trailing slash",
			pattern: "foo/",
			paths: map[string]bool{
				"foo":         false,
				"foo/":        true,
				"foo/bar":     true,
				"bar/foo":     false,
				"bar/foo/baz": true,
				"bar/baz":     false,
			},
		},
		{
			name:    "single-segment pattern with leading and trailing slash",
			pattern: "/foo/",
			paths: map[string]bool{
				"foo":         false,
				"foo/":        true,
				"foo/bar":     true,
				"bar/foo":     false,
				"bar/foo/baz": false,
				"bar/baz":     false,
			},
		},
		{
			name:    "multi-segment pattern",
			pattern: "foo/bar",
			paths: map[string]bool{
				"foo/bar":         true,
				"foo/bart":        false,
				"foo/bar/baz":     true,
				"baz/foo/bar":     false,
				"baz/foo/bar/qux": false,
			},
		},
		{
			name:    "multi-segment pattern with leading slash",
			pattern: "/foo/bar",
			paths: map[string]bool{
				"foo/bar":         true,
				"foo/bar/baz":     true,
				"baz/foo/bar":     false,
				"baz/foo/bar/qux": false,
			},
		},
		{
			name:    "multi-segment pattern with trailing slash",
			pattern: "foo/bar/",
			paths: map[string]bool{
				"foo/bar":         false,
				"foo/bar/baz":     true,
				"baz/foo/bar":     false,
				"baz/foo/bar/qux": false,
			},
		},
		{
			name:    "multi-segment pattern with leading and trailing slash",
			pattern: "/foo/bar/",
			paths: map[string]bool{
				"foo/bar":         false,
				"foo/bar/baz":     true,
				"baz/foo/bar":     false,
				"baz/foo/bar/qux": false,
			},
		},
		{
			name:    "single segment pattern with wildcard",
			pattern: "f*",
			paths: map[string]bool{
				"foo":         true,
				"foo/":        true,
				"foo/bar":     true,
				"bar/foo":     true,
				"bar/foo/baz": true,
				"bar/baz":     false,
				"xfoo":        false,
			},
		},
		{
			name:    "single segment pattern with leading slash and wildcard",
			pattern: "/f*",
			paths: map[string]bool{
				"foo":         true,
				"foo/":        true,
				"foo/bar":     true,
				"bar/foo":     false,
				"bar/foo/baz": false,
				"bar/baz":     false,
				"xfoo":        false,
			},
		},
		{
			name:    "single segment pattern with trailing slash and wildcard",
			pattern: "f*/",
			paths: map[string]bool{
				"foo":         false,
				"foo/":        true,
				"foo/bar":     true,
				"bar/foo":     false,
				"bar/foo/baz": true,
				"bar/baz":     false,
				"xfoo":        false,
			},
		},
		{
			name:    "single segment pattern with leading and trailing slash and wildcard",
			pattern: "/f*/",
			paths: map[string]bool{
				"foo":         false,
				"foo/":        true,
				"foo/bar":     true,
				"bar/foo":     false,
				"bar/foo/baz": false,
				"bar/baz":     false,
				"xfoo":        false,
			},
		},
		{
			name:    "single segment pattern with escaped wildcard",
			pattern: "f\\*o",
			paths: map[string]bool{
				"foo": false,
				"f*o": true,
			},
		},
		{
			name:    "multi-segment pattern with wildcard",
			pattern: "foo/*.txt",
			paths: map[string]bool{
				"foo":                 false,
				"foo/":                false,
				"foo/bar.txt":         true,
				"foo/bar/baz.txt":     false,
				"qux/foo/bar.txt":     false,
				"qux/foo/bar/baz.txt": false,
			},
		},
		{
			name:    "single segment pattern with single-character wildcard",
			pattern: "f?o",
			paths: map[string]bool{
				"foo":  true,
				"fo":   false,
				"fooo": false,
			},
		},
		{
			name:    "single segment pattern with escaped single-character wildcard",
			pattern: "f\\?o",
			paths: map[string]bool{
				"foo": false,
				"f?o": true,
			},
		},
		{
			name:    "single segment pattern with character range",
			pattern: "[Ffb]oo",
			paths: map[string]bool{
				"foo": true,
				"Foo": true,
				"boo": true,
				"too": false,
			},
		},
		{
			name:    "single segment pattern with escaped character range",
			pattern: "[\\]f]o\\[o\\]",
			paths: map[string]bool{
				"fo[o]": true,
				"]o[o]": true,
				"foo":   false,
			},
		},
		{
			name:    "leading double-asterisk wildcard",
			pattern: "**/foo/bar",
			paths: map[string]bool{
				"foo/bar":         true,
				"qux/foo/bar":     true,
				"qux/foo/bar/baz": true,
				"foo/baz/bar":     false,
				"qux/foo/baz/bar": false,
			},
		},
		{
			name:    "leading double-asterisk wildcard with regular wildcard",
			pattern: "**/*bar*",
			paths: map[string]bool{
				"bar":         true,
				"foo/bar":     true,
				"foo/rebar":   true,
				"foo/barrio":  true,
				"foo/qux/bar": true,
			},
		},
		{
			name:    "trailing double-asterisk wildcard",
			pattern: "foo/bar/**",
			paths: map[string]bool{
				"foo/bar":         false,
				"foo/bar/baz":     true,
				"foo/bar/baz/qux": true,
				"qux/foo/bar":     false,
				"qux/foo/bar/baz": false,
			},
		},
		{
			name:    "middle double-asterisk wildcard",
			pattern: "foo/**/bar",
			paths: map[string]bool{
				"foo/bar":              true,
				"foo/bar/baz":          true,
				"foo/qux/bar/baz":      true,
				"foo/qux/quux/bar/baz": true,
				"foo/bar/baz/qux":      true,
				"qux/foo/bar":          false,
				"qux/foo/bar/baz":      false,
			},
		},
		{
			name:    "middle double-asterisk wildcard with trailing slash",
			pattern: "foo/**/",
			paths: map[string]bool{
				"foo/bar":     false,
				"foo/bar/":    true,
				"foo/bar/baz": true,
			},
		},
	}

	tmpRepoPath, cleanup := createGitRepo(t)
	defer cleanup()

	for _, e := range examples {
		ioutil.WriteFile(path.Join(tmpRepoPath, ".gitignore"), []byte(e.pattern+"\n"), 0644)

		t.Run(e.name, func(t *testing.T) {
			for path, shouldMatch := range e.paths {
				gitMatch := gitCheckIgnore(t, tmpRepoPath, path)
				require.Equal(t, gitMatch, shouldMatch, "bad test! pattern=%s path=%s git-match=%v expectation=%v", e.pattern, path, gitMatch, shouldMatch)

				pattern, err := newPattern(e.pattern)
				require.NoError(t, err)

				actual, err := pattern.match(path)
				assert.NoError(t, err)
				if shouldMatch {
					assert.True(t, actual, "expected pattern %s to match path %s", e.pattern, path)
				} else {
					assert.False(t, actual, "expected pattern %s to not match path %s", e.pattern, path)
				}
			}
		})
	}
}

func createGitRepo(t *testing.T) (string, func()) {
	dir, err := ioutil.TempDir("", "codeowners-test-")
	if err != nil {
		t.Fatalf("creating git repo tempdir: %v", err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err = cmd.Run(); err != nil {
		t.Fatalf("initializing git repo: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir) // clean up
	}

	return dir, cleanup
}

func gitCheckIgnore(t *testing.T, dir, path string) bool {
	cmd := exec.Command("git", "check-ignore", path)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		if _, isExitError := err.(*exec.ExitError); isExitError {
			return false
		}
		t.Fatalf("running git check-ignore: %v", err)
	}
	return true
}
