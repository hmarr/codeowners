package codeowners_test

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/hmarr/codeowners"
)

func Example() {
	f := bytes.NewBufferString(`src/**/*.c @acme/c-developers
# The following line should be ignored; it contains only spaces and tabs` +
		" \t\nsrc/**/*.go @acme/go-developers")
	ruleset, err := codeowners.ParseFile(f)
	if err != nil {
		panic(err)
	}

	match, err := ruleset.Match("src/foo.c")
	fmt.Println(match.Owners)

	match, err = ruleset.Match("src/foo.rs")
	fmt.Println(match)

	match, err = ruleset.Match("src/go/bar/bar.go")
	fmt.Println(match.Owners)
	// Output:
	// [@acme/c-developers]
	// <nil>
	// [@acme/go-developers]
}

func ExampleParseFile() {
	f := bytes.NewBufferString("src/**/*.go @acme/go-developers # Go code")
	ruleset, err := codeowners.ParseFile(f)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(ruleset))
	fmt.Println(ruleset[0].RawPattern())
	fmt.Println(ruleset[0].Owners[0].String())
	fmt.Println(ruleset[0].Comment)
	// Output:
	// 1
	// src/**/*.go
	// @acme/go-developers
	// Go code
}

func ExampleParseFile_customOwnerMatchers() {
	validUsernames := []string{"the-a-team", "the-b-team"}
	usernameRegexp := regexp.MustCompile(`\A@([a-zA-Z0-9\-]+)\z`)

	f := bytes.NewBufferString("src/**/*.go @the-a-team # Go code")
	ownerMatchers := []codeowners.OwnerMatcher{
		codeowners.OwnerMatchFunc(codeowners.MatchEmailOwner),
		codeowners.OwnerMatchFunc(func(s string) (codeowners.Owner, error) {
			// Custom owner matcher that only matches valid usernames
			match := usernameRegexp.FindStringSubmatch(s)
			if match == nil {
				return codeowners.Owner{}, codeowners.ErrNoMatch
			}

			for _, t := range validUsernames {
				if t == match[1] {
					return codeowners.Owner{Value: match[1], Type: codeowners.TeamOwner}, nil
				}
			}
			return codeowners.Owner{}, codeowners.ErrNoMatch
		}),
	}
	ruleset, err := codeowners.ParseFile(f, codeowners.WithOwnerMatchers(ownerMatchers))
	if err != nil {
		panic(err)
	}
	fmt.Println(len(ruleset))
	fmt.Println(ruleset[0].RawPattern())
	fmt.Println(ruleset[0].Owners[0].String())
	fmt.Println(ruleset[0].Comment)
	// Output:
	// 1
	// src/**/*.go
	// @the-a-team
	// Go code
}

func ExampleRuleset_Match() {
	f := bytes.NewBufferString("src/**/*.go @acme/go-developers # Go code")
	ruleset, _ := codeowners.ParseFile(f)

	match, _ := ruleset.Match("src")
	fmt.Println("src", match != nil)

	match, _ = ruleset.Match("src/foo.go")
	fmt.Println("src/foo.go", match != nil)

	match, _ = ruleset.Match("src/foo/bar.go")
	fmt.Println("src/foo/bar.go", match != nil)

	match, _ = ruleset.Match("src/foo.rs")
	fmt.Println("src/foo.rs", match != nil)
	// Output:
	// src false
	// src/foo.go true
	// src/foo/bar.go true
	// src/foo.rs false
}
