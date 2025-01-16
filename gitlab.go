package codeowners

import "regexp"

var (
	gitLabUsernameRegexp = regexp.MustCompile(`\A@(([a-zA-Z0-9\-_]+)([._][a-zA-Z0-9\-_]+)*)\z`)
	gitLabGroupRegexp    = regexp.MustCompile(`\A@(([a-zA-Z0-9\-_]+)([/][a-zA-Z0-9\-_]+)+)\z`)
	gitLabRoleNameRegexp = regexp.MustCompile(`\A@@(([a-zA-Z0-9\-_]+)([._][a-zA-Z0-9\-_]+)*)\z`)
)

func matchCustomOwner(s, t string, rgx *regexp.Regexp) (Owner, error) {
	match := rgx.FindStringSubmatch(s)
	if match == nil || len(match) < 2 {
		return Owner{}, ErrNoMatch
	}

	return Owner{Value: match[1], Type: t}, nil
}

func GitLabOwnerMatchers() []OwnerMatcher {
	return []OwnerMatcher{
		OwnerMatchFunc(func(s string) (Owner, error) {
			return matchCustomOwner(s, UsernameOwner, gitLabUsernameRegexp)
		}),
		OwnerMatchFunc(func(s string) (Owner, error) {
			return matchCustomOwner(s, GroupOwner, gitLabGroupRegexp)
		}),
		OwnerMatchFunc(func(s string) (Owner, error) {
			return matchCustomOwner(s, RoleOwner, gitLabRoleNameRegexp)
		}),
		OwnerMatchFunc(MatchEmailOwner),
	}
}
