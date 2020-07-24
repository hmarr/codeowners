package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hmarr/codeowners"
	"github.com/karrick/godirwalk"
	flag "github.com/spf13/pflag"
)

var (
	ownerFilter    *string = flag.StringP("owner", "o", "", "filter results by owner")
	codeownersPath *string = flag.StringP("file", "f", "CODEOWNERS", "CODEOWNERS file path")
	helpFlag       *bool   = flag.BoolP("help", "h", false, "show this help message")
)

func main() {
	flag.Parse()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: codeowners <path>...\n")
		flag.PrintDefaults()
	}

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	ruleset, err := loadCodeowners(*codeownersPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", *codeownersPath, err)
		os.Exit(1)
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	for _, startPath := range paths {
		err = godirwalk.Walk(startPath, &godirwalk.Options{
			Callback: func(path string, dirent *godirwalk.Dirent) error {
				if path == ".git" {
					return filepath.SkipDir
				}

				if !dirent.IsDir() {
					return printFileOwners(ruleset, path)
				}
				return nil
			},
			Unsorted: true,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(1)
		}
	}
}

func printFileOwners(ruleset codeowners.Ruleset, path string) error {
	rule, err := ruleset.Match(path)
	if rule == nil || err != nil {
		return err
	}

	for _, o := range rule.Owners {
		if *ownerFilter == "" || o.Value == *ownerFilter {
			fmt.Printf("%s:\t%s\n", path, o.String())
		}
	}
	return nil
}

func loadCodeowners(path string) (codeowners.Ruleset, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	return codeowners.ParseFile(file)
}
