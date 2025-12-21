package bazel

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Target represents a Bazel build target
type Target struct {
	Label string // e.g., "//util:util"
	Kind  string // e.g., "cc_library"
}

// QueryAllCCTargets returns all C++ targets in the workspace
func QueryAllCCTargets(workspaceRoot string) ([]Target, error) {
	// Use --output=label_kind to get both label and kind
	cmd := exec.Command("bazel", "query", `kind("cc_.* rule", //...)`, "--output=label_kind")
	cmd.Dir = workspaceRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed: %w\nOutput: %s", err, string(output))
	}

	var targets []Target
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and Bazel informational output
		if line == "" {
			continue
		}
		// Format is: "kind_name rule //label"
		// e.g., "cc_library rule //util:util"
		parts := strings.Fields(line)
		if len(parts) >= 3 && strings.HasPrefix(parts[2], "//") {
			targets = append(targets, Target{
				Label: parts[2],
				Kind:  parts[0], // e.g., "cc_binary", "cc_library"
			})
		}
	}

	return targets, nil
}

// QueryDeps returns the dependencies of a specific target (only cc_* rules in workspace)
func QueryDeps(workspaceRoot, targetLabel string) ([]string, error) {
	// Query for cc_* rule dependencies of this target
	query := fmt.Sprintf(`kind("cc_.* rule", deps(%s)) - %s`, targetLabel, targetLabel)
	cmd := exec.Command("bazel", "query", query)
	cmd.Dir = workspaceRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed: %w\nOutput: %s", err, string(output))
	}

	var deps []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and Bazel informational output
		if line == "" || !strings.HasPrefix(line, "//") {
			continue
		}
		deps = append(deps, line)
	}

	return deps, nil
}

// QuerySourceFiles returns all source files (from the workspace, not external)
// that are part of any cc_* target in the workspace
func QueryAllSourceFiles(workspaceRoot string) ([]string, error) {
	// Get all CC targets first
	targets, err := QueryAllCCTargets(workspaceRoot)
	if err != nil {
		return nil, err
	}

	// Collect all source files from all targets
	sourceFilesMap := make(map[string]bool)
	for _, target := range targets {
		files, err := QuerySourceFilesForTarget(workspaceRoot, target.Label)
		if err != nil {
			return nil, fmt.Errorf("failed to query files for %s: %w", target.Label, err)
		}
		for _, file := range files {
			sourceFilesMap[file] = true
		}
	}

	// Convert map to slice
	var sourceFiles []string
	for file := range sourceFilesMap {
		sourceFiles = append(sourceFiles, file)
	}

	return sourceFiles, nil
}

// QuerySourceFilesForTarget returns source files for a specific target
func QuerySourceFilesForTarget(workspaceRoot, targetLabel string) ([]string, error) {
	// Query for source files in this target's dependencies
	// Filter to only workspace files (starting with //)
	query := fmt.Sprintf(`filter("^//", kind("source file", deps(%s)))`, targetLabel)
	cmd := exec.Command("bazel", "query", query)
	cmd.Dir = workspaceRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed: %w\nOutput: %s", err, string(output))
	}

	var sourceFiles []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and Bazel informational output
		if line == "" || !strings.HasPrefix(line, "//") {
			continue
		}
		// Convert Bazel label to file path
		// e.g., "//util:strings.cc" -> "util/strings.cc"
		filePath := labelToPath(line)
		sourceFiles = append(sourceFiles, filePath)
	}

	return sourceFiles, nil
}

// labelToPath converts a Bazel label to a file path
// e.g., "//util:strings.cc" -> "util/strings.cc"
// e.g., "//core:engine.h" -> "core/engine.h"
func labelToPath(label string) string {
	// Remove leading "//"
	label = strings.TrimPrefix(label, "//")

	// Split on ":"
	parts := strings.SplitN(label, ":", 2)
	if len(parts) == 2 {
		// Package and file: "util:strings.cc" -> "util/strings.cc"
		return filepath.Join(parts[0], parts[1])
	}

	// No colon, just return as-is (shouldn't happen for source files)
	return label
}

// BuildFileToTargetMap creates a mapping from file paths to their owning target labels
// Returns a map where keys are file paths (e.g., "util/strings.cc") and values are target labels (e.g., "//util:util")
func BuildFileToTargetMap(workspaceRoot string) (map[string]string, error) {
	// Get all CC targets
	targets, err := QueryAllCCTargets(workspaceRoot)
	if err != nil {
		return nil, err
	}

	fileToTarget := make(map[string]string)

	// For each target, get its source files and map them to the target
	for _, target := range targets {
		files, err := QuerySourceFilesForTarget(workspaceRoot, target.Label)
		if err != nil {
			// Log but don't fail - some targets might not have source files
			continue
		}

		for _, filePath := range files {
			fileToTarget[filePath] = target.Label
		}
	}

	return fileToTarget, nil
}
