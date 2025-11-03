// Package matcher provides file pattern matching for include/exclude filtering.
package matcher

import (
	"path/filepath"

	"github.com/nchapman/myrient-dl/internal/parser"
)

// Matcher handles include/exclude pattern matching
type Matcher struct {
	includePattern string
	excludePattern string
}

// New creates a new Matcher with the given patterns
func New(include, exclude string) *Matcher {
	return &Matcher{
		includePattern: include,
		excludePattern: exclude,
	}
}

// Filter applies include/exclude patterns to a list of files
func (m *Matcher) Filter(files []parser.FileInfo) []parser.FileInfo {
	var filtered []parser.FileInfo

	for _, file := range files {
		if m.matches(file.Name) {
			filtered = append(filtered, file)
		}
	}

	return filtered
}

// matches checks if a filename matches the include/exclude criteria
func (m *Matcher) matches(filename string) bool {
	// Check include pattern
	if m.includePattern != "" && m.includePattern != "*" {
		matched, err := filepath.Match(m.includePattern, filename)
		if err != nil || !matched {
			return false
		}
	}

	// Check exclude pattern
	if m.excludePattern != "" {
		matched, err := filepath.Match(m.excludePattern, filename)
		if err != nil {
			return true // If pattern is invalid, don't exclude
		}
		if matched {
			return false
		}
	}

	return true
}
