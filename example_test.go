package codeowners_test

import (
	"bytes"
	"fmt"
    "testing"

	"github.com/hmarr/codeowners"
    "github.com/stretchr/testify/assert"
)

func TestExample(t *testing.T) {
	f := bytes.NewBufferString("src/**/*.c @acme/c-developers")
	ruleset, err := codeowners.ParseFile(f)
	if err != nil {
		panic(err)
	}

	match, err := ruleset.Match("src/foo.c")
	fmt.Println(match.Owners)

	match, err = ruleset.Match("src/foo.rs")
	fmt.Println(match)
	// Output:
	// [@acme/c-developers]
	// <nil>
}

func TestExampleParseFile(t *testing.T) {
	f := bytes.NewBufferString("src/**/*.go @acme/go-developers # Go code")
	ruleset, err := codeowners.ParseFile(f)
	if err != nil {
		panic(err)
	}
    assert.Len(t, ruleset, 1)
    assert.Equal(t, "src/**/*.go", ruleset[0].RawPattern())
    assert.Equal(t, "@acme/go-developers", ruleset[0].Owners[0].String())
    assert.Equal(t, "Go code", ruleset[0].Comment)
}

func TestExampleRuleset_Match(t *testing.T) {
	f := bytes.NewBufferString("src/**/*.go @acme/go-developers # Go code")
	ruleset, _ := codeowners.ParseFile(f)

	match, _ := ruleset.Match("src")
    assert.Nil(t, match)

	match, _ = ruleset.Match("src/foo.go")
    assert.NotNil(t, match)

	match, _ = ruleset.Match("src/foo/bar.go")
    assert.NotNil(t, match)

	match, _ = ruleset.Match("src/foo.rs")
    assert.Nil(t, match)
}
