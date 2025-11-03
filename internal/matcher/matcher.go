// Package matcher provides file pattern matching for include/exclude filtering.
package matcher

import (
	"path/filepath"

	"github.com/nchapman/myrient-dl/internal/parser"
)

// Matcher handles include/exclude pattern matching
type Matcher struct {
	includePatterns []string
	excludePatterns []string
}

// New creates a new Matcher with the given patterns
func New(include, exclude []string) *Matcher {
	return &Matcher{
		includePatterns: include,
		excludePatterns: exclude,
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
	// Check include patterns (OR logic - must match at least one)
	if len(m.includePatterns) > 0 {
		matchedAny := false
		for _, pattern := range m.includePatterns {
			if pattern == "" || pattern == "*" {
				matchedAny = true
				break
			}
			matched, err := filepath.Match(pattern, filename)
			if err != nil {
				continue // Skip invalid patterns
			}
			if matched {
				matchedAny = true
				break
			}
		}
		if !matchedAny {
			return false
		}
	}

	// Check exclude patterns (OR logic - excluded if matches any)
	for _, pattern := range m.excludePatterns {
		if pattern == "" {
			continue
		}
		matched, err := filepath.Match(pattern, filename)
		if err != nil {
			continue // Skip invalid patterns
		}
		if matched {
			return false // Exclude if any pattern matches
		}
	}

	return true
}
