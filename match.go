package codeowners

import (
	"fmt"
	"regexp"
	"strings"
)

type pattern struct {
	pattern string
	regex   *regexp.Regexp
}

// newPattern creates a new pattern struct from a gitignore-style pattern string
func newPattern(patternStr string) (pattern, error) {
	patternRegex, err := buildPatternRegex(patternStr)
	if err != nil {
		return pattern{}, err
	}

	return pattern{
		pattern: patternStr,
		regex:   patternRegex,
	}, nil
}

// match tests if the path provided matches the pattern
func (p pattern) match(testPath string) (bool, error) {
	return p.regex.MatchString(testPath), nil
}

// buildPatternRegex compiles a new regexp object from a gitignore-style pattern string
func buildPatternRegex(pattern string) (*regexp.Regexp, error) {
	var re strings.Builder

	// The pattern is anchored if it starts with a slash, or has a slash before the
	// final character
	slashPos := strings.IndexByte(pattern, '/')
	anchored := slashPos != -1 && slashPos != len(pattern)-1
	if anchored {
		// Patterns with a non-terminal slash can only match from the start of the string
		re.WriteString(`\A`)
	} else {
		// Patterns without a non-terminal slash can match anywhere, but still need to
		// consider string and path-segment boundaries
		re.WriteString(`(?:\A|/)`)
	}

	// For consistency, strip leading and trailing slashes from the pattern, but
	// keep track of whether it's a directory-only pattern (has a trailing slash)
	matchesDir := pattern[len(pattern)-1] == '/'
	patternRunes := []rune(strings.Trim(pattern, "/"))

	inCharClass := false
	escaped := false
	for i := 0; i < len(patternRunes); i++ {
		ch := patternRunes[i]

		// If the previous character was a backslash, treat this as a literal
		if escaped {
			re.WriteString(regexp.QuoteMeta(string(ch)))
			escaped = false
			continue
		}

		switch ch {
		case '\\':
			// Escape the next character
			escaped = true

		case '*':
			// Check for double-asterisk wildcards (^**/, /**/, /**$)
			if i+1 < len(patternRunes) && patternRunes[i+1] == '*' {
				leftAnchored := i == 0
				leadingSlash := i > 0 && patternRunes[i-1] == '/'
				rightAnchored := i+2 == len(patternRunes)
				trailingSlash := i+2 < len(patternRunes) && patternRunes[i+2] == '/'

				if (leftAnchored || leadingSlash) && (rightAnchored || trailingSlash) {
					re.WriteString(`.*`)

					// Leading (**/) and middle (/**/) wildcards have two extra characters to
					// skip, and with trailing wildcards (/**) we're at the end anyway
					i += 2
					break
				}
			}

			// If it's not a double-asterisk, treat it as a regular wildcard
			re.WriteString(`[^/]*`)

		case '?':
			// Single-character wildcard
			re.WriteString(`[^/]`)

		case '[':
			// Open a character class
			inCharClass = true
			re.WriteRune(ch)

		case ']':
			// Close the character class if we're in one, or treat as a literal
			if inCharClass {
				re.WriteRune(ch)
				inCharClass = false
			} else {
				re.WriteString(regexp.QuoteMeta(string(ch)))
			}

		default:
			// Escape literal characters so they don't interfere with the regex
			re.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}

	if inCharClass {
		return nil, fmt.Errorf("unterminated character class in pattern %s", pattern)
	}

	if matchesDir {
		// This will match either a directory that's prefix of a path provided, or
		// a suffix if we assume that tested directories always have a trailing slash
		re.WriteString(`/`)
	} else {
		// End the match either at the end of the string or at a slash (in the case that
		// we've matched a directory)
		re.WriteString(`(?:\z|/)`)
	}

	return regexp.Compile(re.String())
}
