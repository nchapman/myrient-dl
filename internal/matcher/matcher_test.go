package matcher

import (
	"testing"

	"github.com/nchapman/myrient-dl/internal/parser"
)

func TestMatcher_Filter(t *testing.T) {
	files := []parser.FileInfo{
		{Name: "mario.zip", URL: "http://example.com/mario.zip", Size: 1000},
		{Name: "mario_beta.zip", URL: "http://example.com/mario_beta.zip", Size: 2000},
		{Name: "sonic.zip", URL: "http://example.com/sonic.zip", Size: 3000},
		{Name: "sonic_proto.zip", URL: "http://example.com/sonic_proto.zip", Size: 4000},
		{Name: "readme.txt", URL: "http://example.com/readme.txt", Size: 500},
		{Name: "game.rar", URL: "http://example.com/game.rar", Size: 1500},
	}

	tests := []struct {
		name          string
		include       []string
		exclude       []string
		expectedLen   int
		expectedNames []string
	}{
		{
			name:          "include all",
			include:       []string{"*"},
			exclude:       []string{},
			expectedLen:   6,
			expectedNames: []string{"mario.zip", "mario_beta.zip", "sonic.zip", "sonic_proto.zip", "readme.txt", "game.rar"},
		},
		{
			name:          "include only zip files",
			include:       []string{"*.zip"},
			exclude:       []string{},
			expectedLen:   4,
			expectedNames: []string{"mario.zip", "mario_beta.zip", "sonic.zip", "sonic_proto.zip"},
		},
		{
			name:          "include mario files",
			include:       []string{"mario*"},
			exclude:       []string{},
			expectedLen:   2,
			expectedNames: []string{"mario.zip", "mario_beta.zip"},
		},
		{
			name:          "exclude beta files",
			include:       []string{"*"},
			exclude:       []string{"*beta*"},
			expectedLen:   5,
			expectedNames: []string{"mario.zip", "sonic.zip", "sonic_proto.zip", "readme.txt", "game.rar"},
		},
		{
			name:          "include zip but exclude proto",
			include:       []string{"*.zip"},
			exclude:       []string{"*proto*"},
			expectedLen:   3,
			expectedNames: []string{"mario.zip", "mario_beta.zip", "sonic.zip"},
		},
		{
			name:          "include sonic exclude proto",
			include:       []string{"sonic*"},
			exclude:       []string{"*proto*"},
			expectedLen:   1,
			expectedNames: []string{"sonic.zip"},
		},
		{
			name:          "no matches",
			include:       []string{"zelda*"},
			exclude:       []string{},
			expectedLen:   0,
			expectedNames: []string{},
		},
		{
			name:          "multiple includes OR logic",
			include:       []string{"*.zip", "*.rar"},
			exclude:       []string{},
			expectedLen:   5,
			expectedNames: []string{"mario.zip", "mario_beta.zip", "sonic.zip", "sonic_proto.zip", "game.rar"},
		},
		{
			name:          "multiple includes with wildcard patterns",
			include:       []string{"mario*", "sonic*"},
			exclude:       []string{},
			expectedLen:   4,
			expectedNames: []string{"mario.zip", "mario_beta.zip", "sonic.zip", "sonic_proto.zip"},
		},
		{
			name:          "multiple excludes OR logic",
			include:       []string{"*"},
			exclude:       []string{"*beta*", "*proto*"},
			expectedLen:   4,
			expectedNames: []string{"mario.zip", "sonic.zip", "readme.txt", "game.rar"},
		},
		{
			name:          "multiple includes and multiple excludes",
			include:       []string{"*.zip", "*.rar"},
			exclude:       []string{"*beta*", "*proto*"},
			expectedLen:   3,
			expectedNames: []string{"mario.zip", "sonic.zip", "game.rar"},
		},
		{
			name:          "pattern with comma in name",
			include:       []string{"*,*"},
			exclude:       []string{},
			expectedLen:   0, // None of our test files have commas
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.include, tt.exclude)
			result := m.Filter(files)

			if len(result) != tt.expectedLen {
				t.Errorf("expected %d files, got %d", tt.expectedLen, len(result))
			}

			// Check that we got the expected files
			for i, expectedName := range tt.expectedNames {
				if i >= len(result) {
					t.Errorf("missing expected file: %s", expectedName)
					continue
				}
				if result[i].Name != expectedName {
					t.Errorf("expected file %s at index %d, got %s", expectedName, i, result[i].Name)
				}
			}
		})
	}
}

func TestMatcher_EmptyList(t *testing.T) {
	m := New([]string{"*"}, []string{})
	result := m.Filter([]parser.FileInfo{})

	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %d files", len(result))
	}
}

func TestMatcher_InvalidPattern(t *testing.T) {
	files := []parser.FileInfo{
		{Name: "test.zip", URL: "http://example.com/test.zip", Size: 1000},
	}

	// Invalid glob pattern - should not panic
	m := New([]string{"[invalid"}, []string{})
	result := m.Filter(files)

	// With invalid pattern, should return no matches (pattern fails to match)
	if len(result) != 0 {
		t.Errorf("expected no matches with invalid pattern, got %d", len(result))
	}
}

func TestMatcher_FilesWithCommas(t *testing.T) {
	// Test that patterns can handle filenames with commas naturally
	files := []parser.FileInfo{
		{Name: "3, 2, 1, Smurf! My First Racing Game (Europe) (En,Fr,De,Es,It,Nl).zip", URL: "http://example.com/file.zip", Size: 5000},
		{Name: "Game (En,Fr).zip", URL: "http://example.com/game.zip", Size: 2000},
		{Name: "Normal.zip", URL: "http://example.com/normal.zip", Size: 1000},
	}

	tests := []struct {
		name          string
		include       []string
		exclude       []string
		expectedLen   int
		expectedNames []string
	}{
		{
			name:        "match files with commas using wildcard",
			include:     []string{"*,*"},
			exclude:     []string{},
			expectedLen: 2,
			expectedNames: []string{
				"3, 2, 1, Smurf! My First Racing Game (Europe) (En,Fr,De,Es,It,Nl).zip",
				"Game (En,Fr).zip",
			},
		},
		{
			name:        "match specific pattern with comma",
			include:     []string{"*En,Fr*"},
			exclude:     []string{},
			expectedLen: 2,
			expectedNames: []string{
				"3, 2, 1, Smurf! My First Racing Game (Europe) (En,Fr,De,Es,It,Nl).zip",
				"Game (En,Fr).zip",
			},
		},
		{
			name:        "exclude files with commas",
			include:     []string{"*.zip"},
			exclude:     []string{"*,*"},
			expectedLen: 1,
			expectedNames: []string{
				"Normal.zip",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.include, tt.exclude)
			result := m.Filter(files)

			if len(result) != tt.expectedLen {
				t.Errorf("expected %d files, got %d", tt.expectedLen, len(result))
			}

			for i, expectedName := range tt.expectedNames {
				if i >= len(result) {
					t.Errorf("missing expected file: %s", expectedName)
					continue
				}
				if result[i].Name != expectedName {
					t.Errorf("expected file %s at index %d, got %s", expectedName, i, result[i].Name)
				}
			}
		})
	}
}
