package codeowners

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type parseOption func(*parseOptions)

type parseOptions struct {
	ownerMatchers []OwnerMatcher
}

func WithOwnerMatchers(mm []OwnerMatcher) parseOption {
	return func(opts *parseOptions) {
		opts.ownerMatchers = mm
	}
}

type OwnerMatcher interface {
	// Matches give string agains a pattern e.g. a regexp.
	// Should return ErrNoMatch if the pattern doesn't match.
	Match(s string) (Owner, error)
}

type ErrInvalidOwnerFormat struct {
	Owner string
}

func (err ErrInvalidOwnerFormat) Error() string {
	return fmt.Sprintf("invalid owner format '%s'", err.Owner)
}

var ErrNoMatch = errors.New("no match")

var (
	emailRegexp    = regexp.MustCompile(`\A[A-Z0-9a-z\._%\+\-]+@[A-Za-z0-9\.\-]+\.[A-Za-z]{2,6}\z`)
	teamRegexp     = regexp.MustCompile(`\A@([a-zA-Z0-9\-]+\/[a-zA-Z0-9_\-]+)\z`)
	usernameRegexp = regexp.MustCompile(`\A@([a-zA-Z0-9\-_]+)\z`)
	groupRegexp    = regexp.MustCompile(`\A@(([a-zA-Z0-9\-_]+)(/[a-zA-Z0-9\-_]+)*)\z`)
)

var DefaultOwnerMatchers = []OwnerMatcher{
	OwnerMatchFunc(MatchEmailOwner),
	OwnerMatchFunc(MatchTeamOwner),
	OwnerMatchFunc(MatchUsernameOwner),
	OwnerMatchFunc(MatchGroupOwner),
}

type OwnerMatchFunc func(s string) (Owner, error)

func (f OwnerMatchFunc) Match(s string) (Owner, error) {
	return f(s)
}

func MatchEmailOwner(s string) (Owner, error) {
	match := emailRegexp.FindStringSubmatch(s)
	if match == nil {
		return Owner{}, ErrNoMatch
	}

	return Owner{Value: match[0], Type: EmailOwner}, nil
}

func MatchTeamOwner(s string) (Owner, error) {
	match := teamRegexp.FindStringSubmatch(s)
	if match == nil {
		return Owner{}, ErrNoMatch
	}

	return Owner{Value: match[1], Type: TeamOwner}, nil
}

func MatchUsernameOwner(s string) (Owner, error) {
	match := usernameRegexp.FindStringSubmatch(s)
	if match == nil {
		return Owner{}, ErrNoMatch
	}

	return Owner{Value: match[1], Type: UsernameOwner}, nil
}

func MatchGroupOwner(s string) (Owner, error) {
	match := groupRegexp.FindStringSubmatch(s)
	if match == nil {
		return Owner{}, ErrNoMatch
	}

	return Owner{Value: match[1], Type: GroupOwner}, nil
}

// ParseFile parses a CODEOWNERS file, returning a set of rules.
// To override the default owner matchers, pass WithOwnerMatchers() as an option.
func ParseFile(f io.Reader, options ...parseOption) (Ruleset, error) {
	opts := parseOptions{ownerMatchers: DefaultOwnerMatchers}
	for _, opt := range options {
		opt(&opts)
	}

	sectionOwners := []Owner{}
	rules := Ruleset{}
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		// Ignore blank lines and comments
		if len(strings.TrimSpace(line)) == 0 || line[0] == '#' {
			continue
		}

		if isSectionBraces(rune(line[0])) {
			section, err := parseSection(line, opts)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}

			sectionOwners = section.Owners

			continue
		}

		rule, err := parseRule(line, opts, sectionOwners)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		rule.LineNumber = lineNo
		rules = append(rules, rule)
	}
	return rules, nil
}

const (
	statePattern = iota + 1
	stateOwners
	stateSection
)

// parseSection parses a single line of a CODEOWNERS file, returning a Rule struct
func parseSection(ruleStr string, opts parseOptions) (Section, error) {
	s := Section{}

	state := stateSection
	escaped := false
	buf := bytes.Buffer{}
	for i, ch := range strings.TrimSpace(ruleStr) {
		// Comments consume the rest of the line and stop further parsing
		if ch == '#' {
			s.Comment = strings.TrimSpace(ruleStr[i+1:])
			break
		}

		switch state {
		case stateSection:
			switch {
			case ch == '\\':
				// Escape the next character (important for whitespace while parsing), but
				// don't lose the backslash as it's part of the pattern
				escaped = true
				buf.WriteRune(ch)
				continue
			case isSectionBraces(ch):
				continue

			case isSectionChar(ch) || (isWhitespace(ch) && escaped):
				// Keep any valid pattern characters and escaped whitespace
				buf.WriteRune(ch)

			case isWhitespace(ch) && !escaped:
				s.Name = buf.String()
				buf.Reset()
				state = stateOwners

			default:
				return s, fmt.Errorf("section: unexpected character '%c' at position %d", ch, i+1)
			}

		case stateOwners:
			switch {
			case isWhitespace(ch):
				// Whitespace means we've reached the end of the owner or we're just chomping
				// through whitespace before or after owner declarations
				if buf.Len() > 0 {
					ownerStr := buf.String()
					owner, err := newOwner(ownerStr, opts.ownerMatchers)
					if err != nil {
						return s, fmt.Errorf("section: %w at position %d", err, i+1-len(ownerStr))
					}

					s.Owners = append(s.Owners, owner)
					buf.Reset()
				}

			case isOwnersChar(ch):
				// Write valid owner characters to the buffer
				buf.WriteRune(ch)

			default:
				return s, fmt.Errorf("section: unexpected character '%c' at position %d", ch, i+1)
			}
		}
	}

	escaped = false

	// We've finished consuming the line, but we might still have content in the buffer
	// if the line didn't end with a separator (whitespace)
	switch state {
	case stateSection:
		s.Name = buf.String()

	case stateOwners:
		// If there's an owner left in the buffer, don't leave it behind
		if buf.Len() > 0 {
			ownerStr := buf.String()
			owner, err := newOwner(ownerStr, opts.ownerMatchers)
			if err != nil {
				return s, fmt.Errorf("%s at position %d", err.Error(), len(ruleStr)+1-len(ownerStr))
			}

			s.Owners = append(s.Owners, owner)
		}

	}

	return s, nil
}

// parseRule parses a single line of a CODEOWNERS file, returning a Rule struct
func parseRule(ruleStr string, opts parseOptions, inheritedOwners []Owner) (Rule, error) {
	r := Rule{}

	state := statePattern
	escaped := false
	buf := bytes.Buffer{}
	for i, ch := range strings.TrimSpace(ruleStr) {
		// Comments consume the rest of the line and stop further parsing
		if ch == '#' {
			r.Comment = strings.TrimSpace(ruleStr[i+1:])
			break
		}

		switch state {
		case statePattern:
			switch {
			case ch == '\\':
				// Escape the next character (important for whitespace while parsing), but
				// don't lose the backslash as it's part of the pattern
				escaped = true
				buf.WriteRune(ch)
				continue

			case isWhitespace(ch) && !escaped:
				// Unescaped whitespace means this is the end of the pattern
				pattern, err := newPattern(buf.String())
				if err != nil {
					return r, err
				}
				r.pattern = pattern
				buf.Reset()
				state = stateOwners

			case isPatternChar(ch) || (isWhitespace(ch) && escaped):
				// Keep any valid pattern characters and escaped whitespace
				buf.WriteRune(ch)

			default:
				return r, fmt.Errorf("unexpected character '%c' at position %d", ch, i+1)
			}
			// Escaping only applies to one character
			escaped = false

		case stateOwners:
			switch {
			case isWhitespace(ch):
				// Whitespace means we've reached the end of the owner or we're just chomping
				// through whitespace before or after owner declarations
				if buf.Len() > 0 {
					ownerStr := buf.String()
					owner, err := newOwner(ownerStr, opts.ownerMatchers)
					if err != nil {
						return r, fmt.Errorf("%w at position %d", err, i+1-len(ownerStr))
					}
					r.Owners = append(r.Owners, owner)
					buf.Reset()
				}

			case isOwnersChar(ch):
				// Write valid owner characters to the buffer
				buf.WriteRune(ch)

			default:
				return r, fmt.Errorf("unexpected character '%c' at position %d", ch, i+1)
			}
		}
	}

	// We've finished consuming the line, but we might still have content in the buffer
	// if the line didn't end with a separator (whitespace)
	switch state {
	case statePattern:
		if buf.Len() == 0 { // We should have non-empty pattern
			return r, fmt.Errorf("unexpected end of rule")
		}

		pattern, err := newPattern(buf.String())
		if err != nil {
			return r, err
		}
		r.pattern = pattern

	case stateOwners:
		// If there's an owner left in the buffer, don't leave it behind
		if buf.Len() > 0 {
			ownerStr := buf.String()
			owner, err := newOwner(ownerStr, opts.ownerMatchers)
			if err != nil {
				return r, fmt.Errorf("%s at position %d", err.Error(), len(ruleStr)+1-len(ownerStr))
			}
			r.Owners = append(r.Owners, owner)
		}
	}

	if len(r.Owners) == 0 {
		r.Owners = inheritedOwners
	}

	return r, nil
}

// newOwner figures out which kind of owner this is and returns an Owner struct
func newOwner(s string, mm []OwnerMatcher) (Owner, error) {
	for _, m := range mm {
		o, err := m.Match(s)
		if errors.Is(err, ErrNoMatch) {
			continue
		} else if err != nil {
			return Owner{}, err
		}

		return o, nil
	}

	return Owner{}, ErrInvalidOwnerFormat{
		Owner: s,
	}
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isAlphanumeric(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

// isPatternChar matches characters that are allowed in patterns
func isPatternChar(ch rune) bool {
	switch ch {
	case '*', '?', '.', '/', '@', '_', '+', '-', '\\':
		return true
	}
	return isAlphanumeric(ch)
}

// isOwnersChar matches characters that are allowed in owner definitions
func isOwnersChar(ch rune) bool {
	switch ch {
	case '.', '@', '/', '_', '%', '+', '-':
		return true
	}
	return isAlphanumeric(ch)
}

// isSectionChar matches characters that are allowed in owner definitions
func isSectionChar(ch rune) bool {
	switch ch {
	case '.', '@', '/', '_', '%', '+', '-':
		return true
	}
	return isAlphanumeric(ch)
}

// isSectionBraces matches characters that are allowed in section definitions
func isSectionBraces(ch rune) bool {
	switch ch {
	case '[', ']':
		return true
	}
	return false
}
