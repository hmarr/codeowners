package codeowners

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRule(t *testing.T) {
	examples := []struct {
		name     string
		rule     string
		expected Rule
		err      string
	}{
		// Success cases
		{
			name: "username owners",
			rule: "file.txt @user",
			expected: Rule{
				Pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
			},
		},
		{
			name: "team owners",
			rule: "file.txt @org/team",
			expected: Rule{
				Pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "email owners",
			rule: "file.txt foo@example.com",
			expected: Rule{
				Pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "foo@example.com", Type: "email"}},
			},
		},
		{
			name: "multiple owners",
			rule: "file.txt @user @org/team foo@example.com",
			expected: Rule{
				Pattern: mustBuildPattern(t, "file.txt"),
				Owners: []Owner{
					{Value: "user", Type: "username"},
					{Value: "org/team", Type: "team"},
					{Value: "foo@example.com", Type: "email"},
				},
			},
		},
		{
			name: "complex patterns",
			rule: "d?r/* @user",
			expected: Rule{
				Pattern: mustBuildPattern(t, "d?r/*"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
			},
		},
		{
			name: "pattern with space",
			rule: "foo\\ bar @user",
			expected: Rule{
				Pattern: mustBuildPattern(t, "foo\\ bar"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
			},
		},
		{
			name: "comments",
			rule: "file.txt @user # some comment",
			expected: Rule{
				Pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
				Comment: "some comment",
			},
		},

		// Error cases
		{
			name: "empty rule",
			rule: "",
			err:  "unexpected end of rule",
		},
		{
			name: "no owners",
			rule: "pattern # but no more",
			err:  "unexpected end of rule",
		},
		{
			name: "malformed patterns",
			rule: "file.[txt @user",
			err:  "unexpected character '[' at position 6",
		},
		{
			name: "malformed owners",
			rule: "file.txt missing-at-sign",
			err:  "invalid owner at position 10",
		},
	}

	for _, e := range examples {
		t.Run("parses "+e.name, func(t *testing.T) {
			actual, err := parseRule(e.rule)
			if e.err != "" {
				assert.EqualError(t, err, e.err)
			} else {
				assert.Equal(t, e.expected, actual)
			}
		})
	}
}

func mustBuildPattern(t *testing.T, pat string) pattern {
	p, err := newPattern(pat)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
