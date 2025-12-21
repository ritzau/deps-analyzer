package analysis

import (
	"path/filepath"
	"strings"
)

// UncoveredFile represents a source file that is not part of any Bazel target
type UncoveredFile struct {
	Path    string // Absolute or relative path to the file
	Package string // Inferred package (e.g., "//util")
}

// FindUncoveredFiles compares all source files in the workspace with files
// covered by Bazel targets and returns the uncovered ones
func FindUncoveredFiles(allFiles []string, coveredFiles []string) []UncoveredFile {
	// Create a set of covered files for fast lookup
	// Normalize paths for comparison (handle both absolute and relative)
	coveredSet := make(map[string]bool)
	for _, file := range coveredFiles {
		normalized := normalizePath(file)
		coveredSet[normalized] = true
	}

	// Find uncovered files
	var uncovered []UncoveredFile
	for _, file := range allFiles {
		normalized := normalizePath(file)
		if !coveredSet[normalized] {
			pkg := inferPackage(file)
			uncovered = append(uncovered, UncoveredFile{
				Path:    file,
				Package: pkg,
			})
		}
	}

	return uncovered
}

// normalizePath normalizes a file path for comparison
// Handles both absolute paths and relative paths from different starting points
func normalizePath(path string) string {
	// Clean the path (removes redundant separators, etc.)
	path = filepath.Clean(path)

	// Convert to slash separators for consistency
	path = filepath.ToSlash(path)

	// Remove any leading "./" or "../"
	path = strings.TrimPrefix(path, "./")

	// For absolute paths in example/, extract just the relative part
	// e.g., "/path/to/example/util/file.cc" -> "util/file.cc"
	if idx := strings.Index(path, "example/"); idx != -1 {
		path = path[idx+len("example/"):]
	}

	return path
}

// inferPackage attempts to determine the Bazel package from a file path
// e.g., "util/strings.cc" -> "//util"
// e.g., "example/core/engine.cc" -> "//core"
func inferPackage(filePath string) string {
	// Normalize the path first
	filePath = normalizePath(filePath)

	// Get the directory
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "" {
		return "//"
	}

	// Convert to Bazel package format
	// e.g., "util" -> "//util"
	return "//" + filepath.ToSlash(dir)
}
