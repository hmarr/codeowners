package codeowners

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRule(t *testing.T) {
	examples := []struct {
		name          string
		rule          string
		ownerMatchers []OwnerMatcher
		expected      Rule
		err           string
	}{
		// Success cases
		{
			name: "username with dots",
			rule: "file.txt @user.name",
			expected: Rule{
				pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "user.name", Type: "username"}},
			},
		},
		{
			name: "username with underscore",
			rule: "file.txt @user_name",
			expected: Rule{
				pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "user_name", Type: "username"}},
			},
		},
		{
			name: "username owners",
			rule: "file.txt @user",
			expected: Rule{
				pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
			},
		},
		{
			name: "team owners",
			rule: "file.txt @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "email owners",
			rule: "file.txt foo@example.com",
			expected: Rule{
				pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "foo@example.com", Type: "email"}},
			},
		},
		{
			name: "multiple owners",
			rule: "file.txt @user @org/team foo@example.com",
			expected: Rule{
				pattern: mustBuildPattern(t, "file.txt"),
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
				pattern: mustBuildPattern(t, "d?r/*"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
			},
		},
		{
			name: "pattern with space",
			rule: "foo\\ bar @user",
			expected: Rule{
				pattern: mustBuildPattern(t, "foo\\ bar"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
			},
		},
		{
			name: "comments",
			rule: "file.txt @user # some comment",
			expected: Rule{
				pattern: mustBuildPattern(t, "file.txt"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
				Comment: "some comment",
			},
		},
		{
			name: "pattern with no owners",
			rule: "pattern",
			expected: Rule{
				pattern: mustBuildPattern(t, "pattern"),
				Owners:  nil,
				Comment: "",
			},
		},
		{
			name: "pattern with no owners and comment",
			rule: "pattern # but no more",
			expected: Rule{
				pattern: mustBuildPattern(t, "pattern"),
				Owners:  nil,
				Comment: "but no more",
			},
		},
		{
			name: "pattern with no owners with whitespace",
			rule: "pattern ",
			expected: Rule{
				pattern: mustBuildPattern(t, "pattern"),
				Owners:  nil,
				Comment: "",
			},
		},
		{
			name: "pattern with leading and trailing whitespace",
			rule: " pattern @user ",
			expected: Rule{
				pattern: mustBuildPattern(t, "pattern"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
				Comment: "",
			},
		},
		{
			name: "pattern with leading and trailing whitespace and no owner",
			rule: " pattern ",
			expected: Rule{
				pattern: mustBuildPattern(t, "pattern"),
				Owners:  nil,
				Comment: "",
			},
		},

		// Error cases
		{
			name: "empty rule",
			rule: "",
			err:  "unexpected end of rule",
		},
		{
			name: "malformed patterns",
			rule: "file.{txt @user",
			err:  "unexpected character '{' at position 6",
		},
		{
			name: "patterns with brackets",
			rule: "file.[cC] @user",
			err:  "unexpected character '[' at position 6",
		},
		{
			name: "malformed owners",
			rule: "file.txt missing-at-sign",
			err:  "invalid owner format 'missing-at-sign' at position 10",
		},
		{
			name: "malformed owners trailing dot",
			rule: "file.txt @trailing-dot.",
			err:  "invalid owner format '@trailing-dot.' at position 10",
		},
		{
			name: "email owners without email matcher",
			rule: "file.txt foo@example.com",
			ownerMatchers: []OwnerMatcher{
				OwnerMatchFunc(MatchTeamOwner),
				OwnerMatchFunc(MatchUsernameOwner),
			},
			err: "invalid owner format 'foo@example.com' at position 10",
		},
		{
			name: "team owners without team matcher",
			rule: "file.txt @org/team",
			ownerMatchers: []OwnerMatcher{
				OwnerMatchFunc(MatchEmailOwner),
				OwnerMatchFunc(MatchUsernameOwner),
			},
			err: "invalid owner format '@org/team' at position 10",
		},
		{
			name: "username owners without username matcher",
			rule: "file.txt @user",
			ownerMatchers: []OwnerMatcher{
				OwnerMatchFunc(MatchEmailOwner),
				OwnerMatchFunc(MatchTeamOwner),
			},
			err: "invalid owner format '@user' at position 10",
		},
	}

	for _, e := range examples {
		t.Run("parses "+e.name, func(t *testing.T) {
			opts := parseOptions{ownerMatchers: DefaultOwnerMatchers}
			if e.ownerMatchers != nil {
				opts.ownerMatchers = e.ownerMatchers
			}
			actual, err := parseRule(e.rule, opts, nil)
			if e.err != "" {
				assert.EqualError(t, err, e.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, e.expected, actual)
			}
		})
	}
}

func TestParseSection(t *testing.T) {
	examples := []struct {
		name          string
		rule          string
		ownerMatchers []OwnerMatcher
		expected      Section
		err           string
	}{
		// Success cases
		{
			name: "match sections",
			rule: "[Section]",
			expected: Section{
				Name:    "Section",
				Owners:  nil,
				Comment: "",
			},
		},
		{
			name: "match sections with spaces",
			rule: "[Section Spaces]",
			expected: Section{
				Name:    "Section Spaces",
				Owners:  nil,
				Comment: "",
			},
		},
		{
			name: "match sections with optional approval",
			rule: "^[Section]",
			expected: Section{
				Name:             "Section",
				Owners:           nil,
				Comment:          "",
				ApprovalOptional: true,
			},
		},
		{
			name: "match sections with approval count",
			rule: "^[Section][2]",
			expected: Section{
				Name:             "Section",
				Owners:           nil,
				Comment:          "",
				ApprovalOptional: true,
				ApprovalCount:    2,
			},
		},
		{
			name: "match sections with owner",
			rule: "[Section-B-User] @the-b-user",
			expected: Section{
				Name:    "Section-B-User",
				Owners:  []Owner{{Value: "the-b-user", Type: "username"}},
				Comment: "",
			},
		},
		{
			name: "match sections with comment",
			rule: "[Section] # some comment",
			expected: Section{
				Name:    "Section",
				Owners:  nil,
				Comment: "some comment",
			},
		},
		{
			name: "match sections with owner and comment",
			rule: "[Section] @the/a/team # some comment",
			expected: Section{
				Name:    "Section",
				Owners:  []Owner{{Value: "the/a/team", Type: "team"}},
				Comment: "some comment",
			},
			ownerMatchers: []OwnerMatcher{
				OwnerMatchFunc(MatchTeamOwner),
			},
		},

		// Error cases
		// TODO
	}

	for _, e := range examples {
		t.Run("parses Sections "+e.name, func(t *testing.T) {
			opts := parseOptions{ownerMatchers: DefaultOwnerMatchers}
			if e.ownerMatchers != nil {
				opts.ownerMatchers = e.ownerMatchers
			}
			actual, err := parseSection(e.rule, opts)
			if e.err != "" {
				assert.EqualError(t, err, e.err)
			} else {
				assert.NoError(t, err)
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
