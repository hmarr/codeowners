# codeowners

![build](https://github.com/hmarr/codeowners/workflows/build/badge.svg)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/hmarr/codeowners)](https://pkg.go.dev/github.com/hmarr/codeowners)

A CLI and Go library for GitHub's [CODEOWNERS file](https://docs.github.com/en/github/creating-cloning-and-archiving-repositories/about-code-owners#codeowners-syntax).

## Command line usage

By default, the command line tool will walk the directory tree, printing the code owners of any files that are found.

You can pass the `--owner` flag to filter results by a specific owner.

To limit the files the tool looks at, provide one or more paths as arguments.

```
$ codeowners --help
usage: codeowners <path>...
  -f, --file string     CODEOWNERS file path
  -h, --help            show this help message
  -o, --owner strings   filter results by owner
```

## Go library usage

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hmarr/codeowners"
)

func main() {
	file, err := os.Open("CODEOWNERS")
	if err != nil {
		log.Fatal(err)
	}

	ruleset, err := codeowners.ParseFile(file)
	if err != nil {
		log.Fatal(err)
	}

	rule, err := ruleset.Match("path/to/file")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Owners: %v\n", rule.Owners)
}
```
