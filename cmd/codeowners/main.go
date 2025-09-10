package main

import (
	"os"

	"github.com/hmarr/codeowners/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
