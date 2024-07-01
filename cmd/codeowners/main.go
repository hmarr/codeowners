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

type Codeowners struct {
	ownerFilters   []string
	showUnowned    bool
	codeownersPath string
	helpFlag       bool
	sections       bool
}

func main() {
	var (
		c        Codeowners
		helpFlag bool
	)

	flag.StringSliceVarP(&c.ownerFilters, "owner", "o", nil, "filter results by owner")
	flag.BoolVarP(&c.showUnowned, "unowned", "u", false, "only show unowned files (can be combined with -o)")
	flag.StringVarP(&c.codeownersPath, "file", "f", "", "CODEOWNERS file path")
	flag.BoolVarP(&helpFlag, "help", "h", false, "show this help message")
	flag.BoolVarP(&c.sections, "sections", "", false, "support sections and inheritance")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: codeowners <path>...\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	ruleset, err := c.loadCodeowners()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	// Make the @ optional for GitHub teams and usernames
	for i := range c.ownerFilters {
		c.ownerFilters[i] = strings.TrimLeft(c.ownerFilters[i], "@")
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for _, startPath := range paths {
		// godirwalk only accepts directories, so we need to handle files separately
		if !isDir(startPath) {
			if err := c.printFileOwners(out, ruleset, startPath); err != nil {
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
				return c.printFileOwners(out, ruleset, path)
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(1)
		}
	}
}

func (c Codeowners) printFileOwners(out io.Writer, ruleset codeowners.Ruleset, path string) error {
	rule, err := ruleset.Match(path)
	if err != nil {
		return err
	}
	// If we didn't get a match, the file is unowned
	if rule == nil || rule.Owners == nil {
		// Unless explicitly requested, don't show unowned files if we're filtering by owner
		if len(c.ownerFilters) == 0 || c.showUnowned {
			fmt.Fprintf(out, "%-70s  (unowned)\n", path)
		}
		return nil
	}

	// Figure out which of the owners we need to show according to the --owner filters
	ownersToShow := make([]string, 0, len(rule.Owners))
	for _, o := range rule.Owners {
		// If there are no filters, show all owners
		filterMatch := len(c.ownerFilters) == 0 && !c.showUnowned
		for _, filter := range c.ownerFilters {
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

func (c Codeowners) loadCodeowners() (codeowners.Ruleset, error) {
	var parseOptions []codeowners.ParseOption
	if c.sections {
		parseOptions = append(parseOptions, codeowners.WithSectionSupport())
	}

	if c.codeownersPath == "" {
		return codeowners.LoadFileFromStandardLocation(parseOptions...)
	}
	return codeowners.LoadFile(c.codeownersPath, parseOptions...)
}

// isDir checks if there's a directory at the path specified.
func isDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
