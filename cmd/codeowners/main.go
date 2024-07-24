package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmarr/codeowners"
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

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for _, startPath := range paths {
		err = filepath.WalkDir(startPath, func(path string, d os.DirEntry, err error) error {
			if d.IsDir() {
				if path == ".git" {
					return filepath.SkipDir
				}

				// Don't show code owners for directories.
				return nil
			}

			return printFileOwners(out, ruleset, path, ownerFilters, showUnowned)
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(1)
		}
	}
}

func printFileOwners(out io.Writer, ruleset codeowners.Ruleset, path string, ownerFilters []string, showUnowned bool) error {
	rule, err := ruleset.Match(path)
	if err != nil {
		return err
	}
	// If we didn't get a match, the file is unowned
	if rule == nil || rule.Owners == nil {
		// Unless explicitly requested, don't show unowned files if we're filtering by owner
		if len(ownerFilters) == 0 || showUnowned {
			fmt.Fprintf(out, "%-70s  (unowned)\n", path)
		}
		return nil
	}

	// Figure out which of the owners we need to show according to the --owner filters
	ownersToShow := make([]string, 0, len(rule.Owners))
	for _, o := range rule.Owners {
		// If there are no filters, show all owners
		filterMatch := len(ownerFilters) == 0 && !showUnowned
		for _, filter := range ownerFilters {
			if filter == o.Value {
				filterMatch = true
			}
		}
		if filterMatch {
			ownersToShow = append(ownersToShow, o.String())
		}
	}

	// If the owners slice is empty, no owners matched the filters so don't show anything
	if len(ownersToShow) > 0 {
		fmt.Fprintf(out, "%-70s  %s\n", path, strings.Join(ownersToShow, " "))
	}
	return nil
}

func loadCodeowners(path string) (codeowners.Ruleset, error) {
	if path == "" {
		return codeowners.LoadFileFromStandardLocation()
	}
	return codeowners.LoadFile(path)
}
