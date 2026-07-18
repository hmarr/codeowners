package codeowners

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFileIn(t *testing.T) {
	tests := []struct {
		name     string
		present  []string
		expected string
	}{
		{
			name:     "no CODEOWNERS file",
			present:  []string{"README.md"},
			expected: "",
		},
		{
			// GitHub searches .github/ before the root
			name:     "root and .github/",
			present:  []string{"CODEOWNERS", ".github/CODEOWNERS"},
			expected: ".github/CODEOWNERS",
		},
		{
			// GitLab searches the root before docs/
			name:     "root and docs/",
			present:  []string{"docs/CODEOWNERS", "CODEOWNERS"},
			expected: "CODEOWNERS",
		},
		{
			// GitLab searches docs/ before .gitlab/
			name:     "docs/ and .gitlab/",
			present:  []string{".gitlab/CODEOWNERS", "docs/CODEOWNERS"},
			expected: "docs/CODEOWNERS",
		},
		{
			name:     "all standard locations",
			present:  []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS", ".gitlab/CODEOWNERS"},
			expected: ".github/CODEOWNERS",
		},
		{
			name:     "only .gitlab/",
			present:  []string{".gitlab/CODEOWNERS"},
			expected: ".gitlab/CODEOWNERS",
		},
		{
			name:     "only docs/",
			present:  []string{"docs/CODEOWNERS"},
			expected: "docs/CODEOWNERS",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, path := range test.present {
				fullPath := filepath.Join(dir, path)
				require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
				require.NoError(t, os.WriteFile(fullPath, []byte("* @org/team\n"), 0o644))
			}

			expected := test.expected
			if expected != "" {
				expected = filepath.Join(dir, expected)
			}
			assert.Equal(t, expected, findFileIn(dir))
		})
	}
}

// A directory at a standard location shouldn't be mistaken for a CODEOWNERS file
func TestFindFileInIgnoresDirectories(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "CODEOWNERS"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".github", "CODEOWNERS"), []byte("* @org/team\n"), 0o644))

	assert.Equal(t, filepath.Join(dir, ".github", "CODEOWNERS"), findFileIn(dir))
}
