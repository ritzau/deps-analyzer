package analysis

import (
	"path/filepath"
	"testing"
)

func TestFindUncoveredFiles(t *testing.T) {
	tests := []struct {
		name          string
		allFiles      []string
		coveredFiles  []string
		wantUncovered int
		wantContain   string // File that should be in uncovered list
	}{
		{
			name: "simple case with one uncovered file",
			allFiles: []string{
				"util/strings.cc",
				"util/math.cc",
				"util/orphaned.cc",
			},
			coveredFiles: []string{
				"util/strings.cc",
				"util/math.cc",
			},
			wantUncovered: 1,
			wantContain:   "util/orphaned.cc",
		},
		{
			name: "all files covered",
			allFiles: []string{
				"util/strings.cc",
				"util/math.cc",
			},
			coveredFiles: []string{
				"util/strings.cc",
				"util/math.cc",
			},
			wantUncovered: 0,
		},
		{
			name: "handles absolute paths",
			allFiles: []string{
				"/path/to/example/util/strings.cc",
				"/path/to/example/util/orphaned.cc",
			},
			coveredFiles: []string{
				"util/strings.cc",
			},
			wantUncovered: 1,
			wantContain:   "/path/to/example/util/orphaned.cc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uncovered := FindUncoveredFiles(tt.allFiles, tt.coveredFiles)

			if len(uncovered) != tt.wantUncovered {
				t.Errorf("FindUncoveredFiles() found %d uncovered files, want %d", len(uncovered), tt.wantUncovered)
			}

			if tt.wantContain != "" {
				found := false
				for _, uf := range uncovered {
					if uf.Path == tt.wantContain {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("FindUncoveredFiles() should contain %s, but didn't", tt.wantContain)
				}
			}
		})
	}
}

func TestInferPackage(t *testing.T) {
	tests := []struct {
		filePath string
		want     string
	}{
		{"util/strings.cc", "//util"},
		{"core/engine.h", "//core"},
		{"example/util/math.cc", "//util"},
		{"/absolute/path/example/plugins/renderer.cc", "//plugins"},
		{"main/main.cc", "//main"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			got := inferPackage(tt.filePath)
			if got != tt.want {
				t.Errorf("inferPackage(%s) = %s, want %s", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"util/strings.cc", "util/strings.cc"},
		{"./util/strings.cc", "util/strings.cc"},
		{"/path/to/example/util/strings.cc", "util/strings.cc"},
		{"example/util/strings.cc", "util/strings.cc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePath(tt.input)
			if got != tt.want {
				t.Errorf("normalizePath(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

// Integration test using real data
func TestFindUncoveredFilesIntegration(t *testing.T) {
	// Simulate real scenario from example workspace
	allFiles := []string{
		"example/util/strings.cc",
		"example/util/strings.h",
		"example/util/math.cc",
		"example/util/math.h",
		"example/util/file_io.cc",
		"example/util/file_io.h",
		"example/util/time.cc",
		"example/util/time.h",
		"example/util/orphaned.cc", // This is the uncovered file
		"example/core/engine.cc",
		"example/core/engine.h",
	}

	coveredFiles := []string{
		"util/strings.cc",
		"util/strings.h",
		"util/math.cc",
		"util/math.h",
		"util/file_io.cc",
		"util/file_io.h",
		"util/time.cc",
		"util/time.h",
		"core/engine.cc",
		"core/engine.h",
	}

	uncovered := FindUncoveredFiles(allFiles, coveredFiles)

	// Should find exactly 1 uncovered file (orphaned.cc)
	if len(uncovered) != 1 {
		t.Errorf("Expected 1 uncovered file, got %d", len(uncovered))
	}

	if len(uncovered) > 0 {
		if !contains(uncovered[0].Path, "orphaned.cc") {
			t.Errorf("Expected uncovered file to be orphaned.cc, got %s", uncovered[0].Path)
		}

		if uncovered[0].Package != "//util" {
			t.Errorf("Expected package to be //util, got %s", uncovered[0].Package)
		}
	}
}

func contains(s, substr string) bool {
	return filepath.Base(s) == filepath.Base(substr)
}
