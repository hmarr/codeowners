package codeowners

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type patternTest struct {
	Name    string          `json:"name"`
	Pattern string          `json:"pattern"`
	Paths   map[string]bool `json:"paths"`
	Focus   bool            `json:"focus"`
}

func TestMatch(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/patterns.json")
	require.NoError(t, err)

	var tests []patternTest
	err = json.Unmarshal(data, &tests)
	require.NoError(t, err)

	focus := false
	for _, test := range tests {
		if test.Focus {
			focus = true
		}
	}

	for _, test := range tests {
		if test.Focus != focus {
			continue
		}

		t.Run(test.Name, func(t *testing.T) {
			for path, shouldMatch := range test.Paths {
				pattern, err := newPattern(test.Pattern)
				require.NoError(t, err)

				// Debugging tips:
				// - Print the generated regex: `fmt.Println(pattern.regex.String())`
				// - Only run a single case by adding `"focus" : true` to the test in the JSON file

				actual, err := pattern.match(path)
				require.NoError(t, err)

				if shouldMatch {
					assert.True(t, actual, "expected pattern %s to match path %s", test.Pattern, path)
				} else {
					assert.False(t, actual, "expected pattern %s to not match path %s", test.Pattern, path)
				}
			}
		})
	}
}
