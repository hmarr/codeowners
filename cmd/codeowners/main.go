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

func main() {
	var (
		ownerFilters   []string
		showUnowned    bool
		codeownersPath string
		helpFlag       bool
	)
	flag.StringSliceVarP(&ownerFilters, "owner", "o", nil, "filter results by owner")
	flag.BoolVarP(&showUnowned, "unowned", "u", false, "only show unowned files (can be combined with -o)")
	flag.StringVarP(&codeownersPath, "file", "f", "", "CODEOWNERS file path")
	flag.BoolVarP(&helpFlag, "help", "h", false, "show this help message")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: codeowners <path>...\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	ruleset, err := loadCodeowners(codeownersPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	// Make the @ optional for GitHub teams and usernames
	for i := range ownerFilters {
		ownerFilters[i] = strings.TrimLeft(ownerFilters[i], "@")
	}

	for _, startPath := range paths {
		// godirwalk only accepts directories, so we need to handle files separately
		if !isDir(startPath) {
			if err := printFileOwners(ruleset, startPath, ownerFilters, showUnowned); err != nil {
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
					return printFileOwners(ruleset, path, ownerFilters, showUnowned)
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

func printFileOwners(ruleset codeowners.Ruleset, path string, ownerFilters []string, showUnowned bool) error {
	rule, err := ruleset.Match(path)
	if err != nil {
		return err
	}
	// If we didn't get a match, the file is unowned
	if rule == nil || rule.Owners == nil {
		// Unless explicitly requested, don't show unowned files if we're filtering by owner
		if len(ownerFilters) == 0 || showUnowned {
			fmt.Printf("%-70s  (unowned)\n", path)
		}
		return nil
	}

	// Figure out which of the owners we need to show according to the --owner filters
	owners := []string{}
	for _, o := range rule.Owners {
		// If there are no filters, show all owners
		filterMatch := len(ownerFilters) == 0 && !showUnowned
		for _, filter := range ownerFilters {
			if filter == o.Value {
				filterMatch = true
			}
		}
		if filterMatch {
			owners = append(owners, o.String())
		}
	}

	// If the owners slice is empty, no owners matched the filters so don't show anything
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
