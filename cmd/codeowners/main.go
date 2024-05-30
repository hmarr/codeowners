package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmarr/codeowners"
	flag "github.com/spf13/pflag"
)

var ErrCheck = errors.New("unowned files exist")

func main() {
	var (
		ownerFilters   []string
		showUnowned    bool
		checkMode      bool
		codeownersPath string
		helpFlag       bool
	)
	flag.StringSliceVarP(&ownerFilters, "owner", "o", nil, "filter results by owner")
	flag.BoolVarP(&showUnowned, "unowned", "u", false, "only show unowned files (can be combined with -o)")
	flag.StringVarP(&codeownersPath, "file", "f", "", "CODEOWNERS file path")
	flag.BoolVarP(&helpFlag, "help", "h", false, "show this help message")
	flag.BoolVarP(&checkMode, "check", "c", false, "enable check mode and exist with a non-zero status code if unowned files exist")

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

	var checkError bool
	for _, startPath := range paths {
		// godirwalk only accepts directories, so we need to handle files separately
		if !isDir(startPath) {
			if err := printFileOwners(out, ruleset, startPath, ownerFilters, showUnowned, checkMode); err != nil {
				if errors.Is(err, ErrCheck) {
					checkError = true
					continue
				}

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
				err := printFileOwners(out, ruleset, path, ownerFilters, showUnowned, checkMode)
				if err != nil {
					if errors.Is(err, ErrCheck) {
						checkError = true
						return nil
					}
				}

				return err
			}
			return nil
		})

		if err != nil {
			if errors.Is(err, ErrCheck) {
				checkError = true
				continue
			}

			fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(1)
		}
	}

	if checkError {
		if showUnowned {
			out.Flush()
		}

		fmt.Fprintf(os.Stderr, "error: %v\n", ErrCheck.Error())

		os.Exit(1)
	}
}

func printFileOwners(out io.Writer, ruleset codeowners.Ruleset, path string, ownerFilters []string, showUnowned bool, checkMode bool) error {
	hasUnowned := false

	rule, err := ruleset.Match(path)
	if err != nil {
		return err
	}
	// If we didn't get a match, the file is unowned
	if rule == nil || rule.Owners == nil {
		// Unless explicitly requested, don't show unowned files if we're filtering by owner
		if len(ownerFilters) == 0 || showUnowned || checkMode {
			fmt.Fprintf(out, "%-70s  (unowned)\n", path)

			if checkMode {
				hasUnowned = true
			}
		}

		if hasUnowned {
			return ErrCheck
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

// isDir checks if there's a directory at the path specified.
func isDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
