package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmarr/codeowners"
	"github.com/karrick/godirwalk"
	flag "github.com/spf13/pflag"
)

var (
	ownerFilter    *string = flag.StringP("owner", "o", "", "filter results by owner")
	codeownersPath *string = flag.StringP("file", "f", "", "CODEOWNERS file path")
	helpFlag       *bool   = flag.BoolP("help", "h", false, "show this help message")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: codeowners <path>...\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	ruleset, err := loadCodeowners(*codeownersPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	// Make the @ optional for GitHub teams and usernames
	*ownerFilter = strings.TrimLeft(*ownerFilter, "@")

	for _, startPath := range paths {
		// godirwalk only accepts directories, so we need to handle files separately
		if !isDir(startPath) {
			if err := printFileOwners(ruleset, startPath); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v", err)
				os.Exit(1)
			}
			continue
		}

		err = godirwalk.Walk(startPath, &godirwalk.Options{
			Callback: func(path string, dirent *godirwalk.Dirent) error {
				if path == ".git" {
					return filepath.SkipDir
				}

				// Only show code owners for files, not directories
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
	if err != nil {
		return err
	}
	if rule == nil {
		if *ownerFilter == "" {
			fmt.Printf("%-70s  (unowned)\n", path)
		}
		return nil
	}

	owners := []string{}
	for _, o := range rule.Owners {
		if *ownerFilter == "" || o.Value == *ownerFilter {
			owners = append(owners, o.String())
		}
	}

	if len(owners) > 0 {
		fmt.Printf("%-70s  %s\n", path, strings.Join(owners, " "))
	}
	return nil
}

func loadCodeowners(path string) (codeowners.Ruleset, error) {
	if path == "" {
		return codeowners.LoadFileFromStandardLocation()
	}
	return codeowners.LoadFile(path)
}

// isDir checks if there's a directory at the path specified.
func isDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
