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
		include       string
		exclude       string
		expectedLen   int
		expectedNames []string
	}{
		{
			name:          "include all",
			include:       "*",
			exclude:       "",
			expectedLen:   6,
			expectedNames: []string{"mario.zip", "mario_beta.zip", "sonic.zip", "sonic_proto.zip", "readme.txt", "game.rar"},
		},
		{
			name:          "include only zip files",
			include:       "*.zip",
			exclude:       "",
			expectedLen:   4,
			expectedNames: []string{"mario.zip", "mario_beta.zip", "sonic.zip", "sonic_proto.zip"},
		},
		{
			name:          "include mario files",
			include:       "mario*",
			exclude:       "",
			expectedLen:   2,
			expectedNames: []string{"mario.zip", "mario_beta.zip"},
		},
		{
			name:          "exclude beta files",
			include:       "*",
			exclude:       "*beta*",
			expectedLen:   5,
			expectedNames: []string{"mario.zip", "sonic.zip", "sonic_proto.zip", "readme.txt", "game.rar"},
		},
		{
			name:          "include zip but exclude proto",
			include:       "*.zip",
			exclude:       "*proto*",
			expectedLen:   3,
			expectedNames: []string{"mario.zip", "mario_beta.zip", "sonic.zip"},
		},
		{
			name:          "include sonic exclude proto",
			include:       "sonic*",
			exclude:       "*proto*",
			expectedLen:   1,
			expectedNames: []string{"sonic.zip"},
		},
		{
			name:          "no matches",
			include:       "zelda*",
			exclude:       "",
			expectedLen:   0,
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
	m := New("*", "")
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
	m := New("[invalid", "")
	result := m.Filter(files)

	// With invalid pattern, should return no matches (pattern fails to match)
	if len(result) != 0 {
		t.Errorf("expected no matches with invalid pattern, got %d", len(result))
	}
}
