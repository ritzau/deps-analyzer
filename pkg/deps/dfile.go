package deps

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// FileDependency represents dependencies for a single source file
type FileDependency struct {
	SourceFile   string   // e.g., "util/math.cc"
	Dependencies []string // e.g., ["util/math.h", "util/strings.h"]
}

// ParseDFile parses a Makefile-style .d dependency file
// Format: target.o: dep1.cc dep2.h dep3.h ...
func ParseDFile(path string) (*FileDependency, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var sourceFile string
	var dependencies []string

	scanner := bufio.NewScanner(file)
	var currentLine strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Handle line continuations (backslash at end)
		if strings.HasSuffix(strings.TrimSpace(line), "\\") {
			currentLine.WriteString(strings.TrimSuffix(strings.TrimSpace(line), "\\"))
			currentLine.WriteString(" ")
			continue
		}

		currentLine.WriteString(line)
		fullLine := currentLine.String()
		currentLine.Reset()

		// Parse the dependency line
		// Format: "target.o: dep1 dep2 dep3"
		if idx := strings.Index(fullLine, ":"); idx != -1 {
			depsStr := strings.TrimSpace(fullLine[idx+1:])
			depParts := strings.Fields(depsStr)

			for _, dep := range depParts {
				// Skip external dependencies (system includes)
				// Only include workspace files (relative paths without absolute markers)
				if !isWorkspaceFile(dep) {
					continue
				}

				// The first workspace file is typically the source file
				if sourceFile == "" && (strings.HasSuffix(dep, ".cc") || strings.HasSuffix(dep, ".cpp")) {
					sourceFile = dep
				} else {
					// Add to dependencies (headers and other files)
					dependencies = append(dependencies, dep)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &FileDependency{
		SourceFile:   sourceFile,
		Dependencies: dependencies,
	}, nil
}

// isWorkspaceFile checks if a path is a workspace file (not system include)
func isWorkspaceFile(path string) bool {
	// Absolute paths are system includes
	if filepath.IsAbs(path) {
		return false
	}

	// External Bazel dependencies start with "external/"
	if strings.HasPrefix(path, "external/") {
		return false
	}

	// bazel-out paths are build artifacts, not source
	if strings.HasPrefix(path, "bazel-out/") {
		return false
	}

	// Workspace files are relative paths like "util/strings.h" or "core/engine.cc"
	return true
}
