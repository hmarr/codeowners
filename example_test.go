package codeowners_test

import (
	"bytes"
	"fmt"

	"github.com/hmarr/codeowners"
)

func Example() {
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

func ExampleParseFile() {
	f := bytes.NewBufferString("src/**/*.[hc] @acme/c-developers # C headers and source")
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
	// src/**/*.[hc]
	// @acme/c-developers
	// C headers and source
}

func ExampleRuleset_Match() {
	f := bytes.NewBufferString("src/**/*.[hc] @acme/c-developers # C headers and source")
	ruleset, _ := codeowners.ParseFile(f)

	match, _ := ruleset.Match("src")
	fmt.Println("src", match != nil)

	match, _ = ruleset.Match("src/foo.c")
	fmt.Println("src/foo.c", match != nil)

	match, _ = ruleset.Match("src/foo/bar.h")
	fmt.Println("src/foo/bar.h", match != nil)

	match, _ = ruleset.Match("src/foo.rs")
	fmt.Println("src/foo.rs", match != nil)
	// Output:
	// src false
	// src/foo.c true
	// src/foo/bar.h true
	// src/foo.rs false
}
