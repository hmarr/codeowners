package codeowners

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFile(t *testing.T) {
	examples := []struct {
		name     string
		contents string
		expected Ruleset
		err      string
	}{
		// Success cases
		{
			name:     "empty file",
			contents: "",
			expected: Ruleset{},
		},
		{
			name:     "single rule",
			contents: "file.txt @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "file.txt"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 1,
				},
			},
		},
		{
			name:     "multiple rules",
			contents: "file.txt @user\nfile2.txt @org/team",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "file.txt"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 1,
				},
				{
					pattern:    mustBuildPattern(t, "file2.txt"),
					Owners:     []Owner{{Value: "org/team", Type: "team"}},
					LineNumber: 2,
				},
			},
		},
		{
			name:     "with blank lines with whitespace",
			contents: "\nfile.txt @user\n \t\nfile2.txt @org/team\n",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "file.txt"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
				},
				{
					pattern:    mustBuildPattern(t, "file2.txt"),
					Owners:     []Owner{{Value: "org/team", Type: "team"}},
					LineNumber: 4,
				},
			},
		},

		// Section header cases
		{
			name:     "section with default threshold",
			contents: "[Backend]\nbackend/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "backend/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
					Section:    &Section{Name: "Backend", MinApprovals: 1},
				},
			},
		},
		{
			name:     "section with explicit threshold",
			contents: "[Backend][2]\nbackend/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "backend/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
					Section:    &Section{Name: "Backend", MinApprovals: 2},
				},
			},
		},
		{
			name:     "optional section",
			contents: "^[Docs]\ndocs/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "docs/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
					Section:    &Section{Name: "Docs", MinApprovals: 1, Optional: true},
				},
			},
		},
		{
			name:     "optional section with explicit threshold",
			contents: "^[Docs][3]\ndocs/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "docs/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
					Section:    &Section{Name: "Docs", MinApprovals: 3, Optional: true},
				},
			},
		},
		{
			name:     "section with zero threshold",
			contents: "[Section][0]\nsrc/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "src/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
					Section:    &Section{Name: "Section", MinApprovals: 0},
				},
			},
		},
		{
			name:     "section name with spaces",
			contents: "[My Backend Team]\nbackend/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "backend/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
					Section:    &Section{Name: "My Backend Team", MinApprovals: 1},
				},
			},
		},
		{
			name:     "section header followed by blank line",
			contents: "[Backend]\n\nbackend/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "backend/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 3,
					Section:    &Section{Name: "Backend", MinApprovals: 1},
				},
			},
		},
		{
			name:     "two section headers reset section context",
			contents: "[Frontend]\nfe/** @user\n[Backend]\nbe/** @user2",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "fe/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
					Section:    &Section{Name: "Frontend", MinApprovals: 1},
				},
				{
					pattern:    mustBuildPattern(t, "be/**"),
					Owners:     []Owner{{Value: "user2", Type: "username"}},
					LineNumber: 4,
					Section:    &Section{Name: "Backend", MinApprovals: 1},
				},
			},
		},
		{
			name:     "section header with no rules below it",
			contents: "[Empty]\n[Backend]\nbe/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "be/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 3,
					Section:    &Section{Name: "Backend", MinApprovals: 1},
				},
			},
		},
		{
			name:     "rules before section header have nil section",
			contents: "* @user\n[Backend]\nbe/** @user2",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "*"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 1,
				},
				{
					pattern:    mustBuildPattern(t, "be/**"),
					Owners:     []Owner{{Value: "user2", Type: "username"}},
					LineNumber: 3,
					Section:    &Section{Name: "Backend", MinApprovals: 1},
				},
			},
		},
		{
			name:     "comment inside section does not change section context",
			contents: "[Backend]\n# comment\nbe/** @user",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "be/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 3,
					Section:    &Section{Name: "Backend", MinApprovals: 1},
				},
			},
		},
		{
			name:     "multiple rules under one section",
			contents: "[Backend]\nbe/** @user\napi/** @user2",
			expected: Ruleset{
				{
					pattern:    mustBuildPattern(t, "be/**"),
					Owners:     []Owner{{Value: "user", Type: "username"}},
					LineNumber: 2,
					Section:    &Section{Name: "Backend", MinApprovals: 1},
				},
				{
					pattern:    mustBuildPattern(t, "api/**"),
					Owners:     []Owner{{Value: "user2", Type: "username"}},
					LineNumber: 3,
					Section:    &Section{Name: "Backend", MinApprovals: 1},
				},
			},
		},

		// Error cases
		{
			name:     "malformed rule",
			contents: "malformed rule\n",
			err:      "line 1: invalid owner format 'rule' at position 11",
		},
		{
			name:     "whitespace-only section name is not a section header",
			contents: "[   ]\nbe/** @user",
			err:      "line 1: unexpected character ']' at position 5",
		},
	}

	for _, e := range examples {
		t.Run("parses "+e.name, func(t *testing.T) {
			reader := strings.NewReader(e.contents)
			actual, err := ParseFile(reader)
			if e.err != "" {
				assert.EqualError(t, err, e.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, e.expected, actual)
			}
		})
	}
}

func TestParseSectionHeader(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected *Section
	}{
		// Valid section headers
		{
			name:     "required section",
			line:     "[Backend]",
			expected: &Section{Name: "Backend", MinApprovals: 1},
		},
		{
			name:     "required section with explicit threshold",
			line:     "[Backend][2]",
			expected: &Section{Name: "Backend", MinApprovals: 2},
		},
		{
			name:     "optional section",
			line:     "^[Docs]",
			expected: &Section{Name: "Docs", MinApprovals: 1, Optional: true},
		},
		{
			name:     "optional section with explicit threshold",
			line:     "^[Docs][3]",
			expected: &Section{Name: "Docs", MinApprovals: 3, Optional: true},
		},
		{
			name:     "zero threshold",
			line:     "[Section][0]",
			expected: &Section{Name: "Section", MinApprovals: 0},
		},
		{
			name:     "section name with spaces",
			line:     "[My Backend Team]",
			expected: &Section{Name: "My Backend Team", MinApprovals: 1},
		},
		{
			name:     "mixed case name is preserved as-is",
			line:     "[BackEnd]",
			expected: &Section{Name: "BackEnd", MinApprovals: 1},
		},
		{
			name:     "name trimmed of surrounding whitespace",
			line:     "[ Backend ]",
			expected: &Section{Name: "Backend", MinApprovals: 1},
		},

		// Lines that are not section headers
		{name: "no closing bracket", line: "[foo", expected: nil},
		{name: "empty brackets", line: "[]", expected: nil},
		{name: "whitespace-only name", line: "[   ]", expected: nil},
		{name: "caret alone", line: "^", expected: nil},
		{name: "regular file rule", line: "/backend/** @user", expected: nil},
		{name: "comment line", line: "# comment", expected: nil},
		{name: "empty line", line: "", expected: nil},
	}

	for _, test := range tests {
		t.Run("parses "+test.name, func(t *testing.T) {
			section, ok := parseSectionHeader(test.line)
			if test.expected == nil {
				assert.False(t, ok)
				assert.Nil(t, section)
			} else {
				assert.True(t, ok)
				assert.Equal(t, test.expected, section)
			}
		})
	}
}

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
			name: "team owners file with parentheses",
			rule: "file(1).txt @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "file(1).txt"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "team owners file with one parentheses on the left",
			rule: "file(1.txt @user",
			expected: Rule{
				pattern: mustBuildPattern(t, "file(1.txt"),
				Owners:  []Owner{{Value: "user", Type: "username"}},
			},
		},
		{
			name: "team owners file with one parentheses on the right",
			rule: "file1).txt foo@example.com",
			expected: Rule{
				pattern: mustBuildPattern(t, "file1).txt"),
				Owners:  []Owner{{Value: "foo@example.com", Type: "email"}},
			},
		},
		{
			name: "team owners file with parentheses in the folder name",
			rule: "(folder)/file.txt @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "(folder)/file.txt"),
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
			name: "pattern with colon",
			rule: "services/foo:bar/**/* @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "services/foo:bar/**/*"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
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
		{
			name: "pattern with pipe character '|'",
			rule: "foo|bar|baz @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "foo|bar|baz"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "pattern with left curly brace '{'",
			rule: "foo{bar.txt @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "foo{bar.txt"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "pattern with right curly brace '}'",
			rule: "foo}bar.txt @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "foo}bar.txt"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "pattern with curly braces '{' and '}'",
			rule: "foo{bar}.txt @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "foo{bar}.txt"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "pattern with curly braces and pipe character",
			rule: "foo|{bar}.txt @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "foo|{bar}.txt"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "patterns with escaped brackets",
			rule: "nameWith\\[brackets\\] @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "nameWith\\[brackets\\]"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "pattern with tilde '~'",
			rule: "foobar~.txt @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "foobar~.txt"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},
		{
			name: "pattern with caret '^'",
			rule: "vllm/v1/worker/^cpu @org/team",
			expected: Rule{
				pattern: mustBuildPattern(t, "vllm/v1/worker/^cpu"),
				Owners:  []Owner{{Value: "org/team", Type: "team"}},
			},
		},

		// Error cases
		{
			name: "empty rule",
			rule: "",
			err:  "unexpected end of rule",
		},
		{
			name: "malformed owners",
			rule: "file.txt missing-at-sign",
			err:  "invalid owner format 'missing-at-sign' at position 10",
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
			actual, err := parseRule(e.rule, opts)
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
