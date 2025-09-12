package app

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/robandpdx/codeowners"
	pflag "github.com/spf13/pflag"
)

// Run executes the codeowners CLI logic. It returns an exit code suitable for
// passing to os.Exit. All user-facing output (apart from errors) is written to stdout.
// Errors are written to stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	var (
		ownerFilters   []string
		showUnowned    bool
		codeownersPath string
		helpFlag       bool
	)

	fs := pflag.NewFlagSet("codeowners", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringSliceVarP(&ownerFilters, "owner", "o", nil, "filter results by owner")
	fs.BoolVarP(&showUnowned, "unowned", "u", false, "only show unowned files (can be combined with -o)")
	fs.StringVarP(&codeownersPath, "file", "f", "", "CODEOWNERS file path")
	fs.BoolVarP(&helpFlag, "help", "h", false, "show this help message")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: codeowners <path>...\n")
		fs.PrintDefaults()
	}

	// Allow tests to set the standard library flag.CommandLine behavior if desired
	// but isolate from global flags for cleanliness.
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp { // standard library compatibility
			return 0
		}
		return 2
	}

	if helpFlag {
		fs.Usage()
		return 0
	}

	paths := fs.Args()
	// If no positional arguments AND no functional flags were provided, show help (don't attempt to load CODEOWNERS).
	if len(paths) == 0 && len(ownerFilters) == 0 && !showUnowned && codeownersPath == "" {
		fs.Usage()
		return 0
	}
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	ruleset, err := loadCodeowners(codeownersPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	// Make the @ optional for GitHub teams and usernames
	for i := range ownerFilters {
		ownerFilters[i] = strings.TrimLeft(ownerFilters[i], "@")
	}

	out := bufio.NewWriter(stdout)
	defer out.Flush()

	for _, startPath := range paths {
		if !isDir(startPath) { // handle single file path
			if err := printFileOwners(out, ruleset, startPath, ownerFilters, showUnowned); err != nil {
				fmt.Fprintf(stderr, "error: %v", err)
				return 1
			}
			continue
		}

		err = filepath.WalkDir(startPath, func(path string, d os.DirEntry, err error) error {
			if path == ".git" {
				return filepath.SkipDir
			}
			if !d.IsDir() {
				return printFileOwners(out, ruleset, path, ownerFilters, showUnowned)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(stderr, "error: %v", err)
			return 1
		}
	}
	return 0
}

func printFileOwners(out io.Writer, ruleset codeowners.Ruleset, path string, ownerFilters []string, showUnowned bool) error {
	rule, err := ruleset.Match(path)
	if err != nil {
		return err
	}
	if rule == nil || rule.Owners == nil { // unowned file
		if len(ownerFilters) == 0 || showUnowned {
			fmt.Fprintf(out, "%-70s  (unowned)\n", path)
		}
		return nil
	}

	ownersToShow := make([]string, 0, len(rule.Owners))
	for _, o := range rule.Owners {
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
	if len(ownersToShow) > 0 { // only output if something matched filters
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
