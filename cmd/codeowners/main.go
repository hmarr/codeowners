package main

import (
	"bufio"
	"encoding/json"
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
		jsonOutput     bool
		helpFlag       bool
	)
	flag.StringSliceVarP(&ownerFilters, "owner", "o", nil, "filter results by owner")
	flag.BoolVarP(&showUnowned, "unowned", "u", false, "only show unowned files (can be combined with -o)")
	flag.StringVarP(&codeownersPath, "file", "f", "", "CODEOWNERS file path")
	flag.BoolVarP(&jsonOutput, "json", "j", false, "output a JSON array with each file's owners and matching rule")
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

	var p printer = &textPrinter{out: out}
	if jsonOutput {
		p = &jsonPrinter{out: out}
	}

	for _, startPath := range paths {
		// godirwalk only accepts directories, so we need to handle files separately
		if !isDir(startPath) {
			if err := printFileOwners(p, ruleset, startPath, ownerFilters, showUnowned); err != nil {
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
				return printFileOwners(p, ruleset, path, ownerFilters, showUnowned)
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(1)
		}
	}

	if err := p.finish(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}

func printFileOwners(p printer, ruleset codeowners.Ruleset, path string, ownerFilters []string, showUnowned bool) error {
	rule, err := ruleset.Match(path)
	if err != nil {
		return err
	}
	// If we didn't get a match, the file is unowned
	if rule == nil || rule.Owners == nil {
		// Unless explicitly requested, don't show unowned files if we're filtering by owner
		if len(ownerFilters) == 0 || showUnowned {
			return p.print(path, rule, nil)
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
		return p.print(path, rule, ownersToShow)
	}
	return nil
}

// printer writes one result per file. owners is empty for unowned files, and
// rule is the matching rule (nil if no rule matched).
type printer interface {
	print(path string, rule *codeowners.Rule, owners []string) error
	finish() error
}

// textPrinter writes the default column-aligned output.
type textPrinter struct {
	out io.Writer
}

func (t *textPrinter) print(path string, _ *codeowners.Rule, owners []string) error {
	if len(owners) == 0 {
		_, err := fmt.Fprintf(t.out, "%-70s  (unowned)\n", path)
		return err
	}
	_, err := fmt.Fprintf(t.out, "%-70s  %s\n", path, strings.Join(owners, " "))
	return err
}

func (t *textPrinter) finish() error { return nil }

// jsonPrinter streams a JSON array, one object per file, so large trees don't
// have to be held in memory.
type jsonPrinter struct {
	out     io.Writer
	started bool
}

type jsonRule struct {
	Pattern string `json:"pattern"`
	Line    int    `json:"line"`
}

type jsonEntry struct {
	Path   string    `json:"path"`
	Owners []string  `json:"owners"`
	Rule   *jsonRule `json:"rule"`
}

func (j *jsonPrinter) print(path string, rule *codeowners.Rule, owners []string) error {
	entry := jsonEntry{Path: filepath.ToSlash(path), Owners: owners}
	if entry.Owners == nil {
		entry.Owners = []string{}
	}
	if rule != nil {
		entry.Rule = &jsonRule{Pattern: rule.RawPattern(), Line: rule.LineNumber}
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	sep := ",\n  "
	if !j.started {
		sep = "[\n  "
		j.started = true
	}
	_, err = fmt.Fprintf(j.out, "%s%s", sep, b)
	return err
}

func (j *jsonPrinter) finish() error {
	if !j.started {
		_, err := fmt.Fprint(j.out, "[]\n")
		return err
	}
	_, err := fmt.Fprint(j.out, "\n]\n")
	return err
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
