package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hmarr/codeowners"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCodeowners = `* @org/everyone
/src/ @org/backend @alice
/docs/ docs@example.com
/vendor/
`

var testPaths = []string{"README.md", "src/api/server.go", "docs/guide.md", "vendor/lib.go"}

func runPrinter(t *testing.T, p printer, out *bytes.Buffer, rules string, paths []string, filters []string, unowned bool) string {
	t.Helper()
	ruleset, err := codeowners.ParseFile(strings.NewReader(rules))
	require.NoError(t, err)
	for _, path := range paths {
		require.NoError(t, printFileOwners(p, ruleset, path, filters, unowned))
	}
	require.NoError(t, p.finish())
	return out.String()
}

func runJSON(t *testing.T, rules string, paths []string, filters []string, unowned bool) []jsonEntry {
	t.Helper()
	var out bytes.Buffer
	raw := runPrinter(t, &jsonPrinter{out: &out}, &out, rules, paths, filters, unowned)
	var entries []jsonEntry
	require.NoError(t, json.Unmarshal([]byte(raw), &entries), raw)
	return entries
}

func TestTextOutputUnchanged(t *testing.T) {
	var out bytes.Buffer
	got := runPrinter(t, &textPrinter{out: &out}, &out, testCodeowners, testPaths, nil, false)
	want := "README.md" + strings.Repeat(" ", 61) + "  @org/everyone\n" +
		"src/api/server.go" + strings.Repeat(" ", 53) + "  @org/backend @alice\n" +
		"docs/guide.md" + strings.Repeat(" ", 57) + "  docs@example.com\n" +
		"vendor/lib.go" + strings.Repeat(" ", 57) + "  (unowned)\n"
	assert.Equal(t, want, got)
}

func TestJSONOutput(t *testing.T) {
	entries := runJSON(t, testCodeowners, testPaths, nil, false)
	assert.Equal(t, []jsonEntry{
		{Path: "README.md", Owners: []string{"@org/everyone"}, Rule: &jsonRule{Pattern: "*", Line: 1}},
		{Path: "src/api/server.go", Owners: []string{"@org/backend", "@alice"}, Rule: &jsonRule{Pattern: "/src/", Line: 2}},
		{Path: "docs/guide.md", Owners: []string{"docs@example.com"}, Rule: &jsonRule{Pattern: "/docs/", Line: 3}},
		// A rule with no owners explicitly unowns files; the rule is still reported.
		{Path: "vendor/lib.go", Owners: []string{}, Rule: &jsonRule{Pattern: "/vendor/", Line: 4}},
	}, entries)
}

func TestJSONUnmatchedFileHasNullRule(t *testing.T) {
	var out bytes.Buffer
	raw := runPrinter(t, &jsonPrinter{out: &out}, &out, "/src/ @org/backend\n", []string{"README.md"}, nil, false)
	assert.Equal(t, "[\n  {\"path\":\"README.md\",\"owners\":[],\"rule\":null}\n]\n", raw)
}

func TestJSONRespectsOwnerFilter(t *testing.T) {
	entries := runJSON(t, testCodeowners, testPaths, []string{"alice"}, false)
	require.Len(t, entries, 1)
	assert.Equal(t, "src/api/server.go", entries[0].Path)
	assert.Equal(t, []string{"@alice"}, entries[0].Owners)
}

func TestJSONRespectsUnownedFlag(t *testing.T) {
	entries := runJSON(t, testCodeowners, testPaths, nil, true)
	require.Len(t, entries, 1)
	assert.Equal(t, "vendor/lib.go", entries[0].Path)
	assert.Empty(t, entries[0].Owners)
}

func TestJSONNoResultsIsEmptyArray(t *testing.T) {
	var out bytes.Buffer
	raw := runPrinter(t, &jsonPrinter{out: &out}, &out, testCodeowners, testPaths, []string{"nobody"}, false)
	assert.Equal(t, "[]\n", raw)
}

func TestJSONEscapesPaths(t *testing.T) {
	entries := runJSON(t, "* @org/everyone\n", []string{`odd "name" file.txt`}, nil, false)
	require.Len(t, entries, 1)
	assert.Equal(t, `odd "name" file.txt`, entries[0].Path)
}
