package main

import (
	"os"

	"github.com/robandpdx/gh-codeowners/internal/app"
)

// This binary is intended for distribution as a GitHub CLI extension. When
// installed (repository name must be gh-codeowners), users can run:
//
//	gh codeowners [flags]
//
// which invokes this executable.
func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
