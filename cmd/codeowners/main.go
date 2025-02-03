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
		showUnmatched  bool
		codeownersPath string
		helpFlag       bool
	)
	flag.StringSliceVarP(&ownerFilters, "owner", "o", nil, "filter results by owner")
	flag.BoolVarP(&showUnowned, "unowned", "u", false, "only show unowned files (can be combined with -o, implies -m)")
	flag.BoolVarP(&showUnmatched, "unmatched", "m", false, "only show files not matched by CODEOWNERS (can be combined with -o)")
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
		// godirwalk only accepts directories, so we need to handle files separately
		if !isDir(startPath) {
			if err := printFileOwners(out, ruleset, startPath, ownerFilters, showUnowned, showUnmatched); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v", err)
				os.Exit(1)
			}
			continue
		}

		err = filepath.WalkDir(startPath, func(path string, d os.DirEntry, err error) error {
			if path == ".git" {
				return filepath.SkipDir
			}

			// Only show code owners for files, not directories
			if !d.IsDir() {
				return printFileOwners(out, ruleset, path, ownerFilters, showUnowned, showUnmatched)
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(1)
		}
	}
}

func printFileOwners(out io.Writer, ruleset codeowners.Ruleset, path string, ownerFilters []string, showUnowned, showUnmatched bool) error {
	rule, err := ruleset.Match(path)
	if err != nil {
		return err
	}
	// If we didn't get a rule match, consider the file unmatched
	if rule == nil {
		if showUnmatched || len(ownerFilters) == 0 {
			fmt.Fprintf(out, "%-70s  (unmatched)\n", path)
		}
		return nil
	}

	// If we did get a match, but the rule has no owners, the file is unowned
	if rule.Owners == nil {
		if showUnowned || (len(ownerFilters) == 0 && !showUnmatched) {
			fmt.Fprintf(out, "%-70s  (unowned)\n", path)
		}
		return nil
	}

	// Figure out which of the owners we need to show according to the --owner filters
	ownersToShow := make([]string, 0, len(rule.Owners))
	for _, o := range rule.Owners {
		// If there are no filters, show all owners
		filterMatch := len(ownerFilters) == 0 && !showUnowned && !showUnmatched
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

// isDir checks if there's a directory at the path specified.
func isDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
