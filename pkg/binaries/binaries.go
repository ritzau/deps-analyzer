package binaries

import (
	"fmt"
	"os/exec"
	"strings"
)

// BinaryInfo represents a cc_binary or cc_shared_library
type BinaryInfo struct {
	Label             string   `json:"label"`
	Kind              string   `json:"kind"` // "cc_binary" or "cc_shared_library"
	DynamicDeps       []string `json:"dynamicDeps"`
	DataDeps          []string `json:"dataDeps"`
	SystemLibraries   []string `json:"systemLibraries"`
	RegularDeps       []string `json:"regularDeps"`      // Direct cc_library dependencies
	InternalTargets   []string `json:"internalTargets"` // All cc_library targets this binary depends on
}

// QueryAllBinaries finds all cc_binary and cc_shared_library targets
func QueryAllBinaries(workspace string) ([]string, error) {
	cmd := exec.Command("bazel", "query", "--output=label",
		"kind('cc_binary|cc_shared_library', //...)")
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed: %w\nOutput: %s", err, string(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var binaries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and Bazel status messages
		if line != "" && strings.HasPrefix(line, "//") {
			binaries = append(binaries, line)
		}
	}

	return binaries, nil
}

// GetBinaryInfo retrieves detailed information about a binary or shared library
func GetBinaryInfo(workspace string, label string) (*BinaryInfo, error) {
	// Query for rule kind
	cmd := exec.Command("bazel", "query", "--output=label_kind", label)
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed for %s: %w", label, err)
	}

	// Parse kind from output (format: "cc_binary rule //label")
	outputStr := string(output)
	// Filter out Loading/INFO lines, get only the result line
	lines := strings.Split(outputStr, "\n")
	var resultLine string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Loading:") && !strings.HasPrefix(line, "INFO:") {
			resultLine = line
			break
		}
	}

	parts := strings.Fields(resultLine)
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected query output format: %s", resultLine)
	}
	kind := parts[0] // First field is the rule kind (e.g., "cc_binary", "cc_shared_library")

	info := &BinaryInfo{
		Label: label,
		Kind:  kind,
	}

	// Get shared library dependencies (both dynamic_deps and from data)
	sharedLibDeps := querySharedLibraryDeps(workspace, label)

	// Separate into dynamic_deps and data_deps based on how they're referenced
	// For now, we'll use a heuristic: query deps to see what's linked
	linkedDeps := queryLinkedDeps(workspace, label)

	for _, dep := range sharedLibDeps {
		if contains(linkedDeps, dep) {
			info.DynamicDeps = append(info.DynamicDeps, dep)
		} else {
			info.DataDeps = append(info.DataDeps, dep)
		}
	}

	// Get system libraries from linkopts
	info.SystemLibraries = querySystemLibraries(workspace, label)

	// Get all cc_library targets this binary depends on (excluding shared libraries)
	info.InternalTargets = queryInternalTargets(workspace, label)

	return info, nil
}

// queryInternalTargets finds all cc_library targets this binary depends on
func queryInternalTargets(workspace string, label string) []string {
	// Query for all cc_library targets in the dependency tree
	cmd := exec.Command("bazel", "query",
		fmt.Sprintf("kind('cc_library', deps(%s))", label))
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	return parseLabels(string(output), label)
}

// querySharedLibraryDeps finds all cc_shared_library dependencies
func querySharedLibraryDeps(workspace string, label string) []string {
	// Query for all shared libraries this target depends on
	cmd := exec.Command("bazel", "query",
		fmt.Sprintf("kind('cc_shared_library', deps(%s))", label))
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	return parseLabels(string(output), label)
}

// queryLinkedDeps finds dependencies that are linked (not just data)
func queryLinkedDeps(workspace string, label string) []string {
	// Query direct deps only (depth 1) to find what's actually linked
	cmd := exec.Command("bazel", "query",
		fmt.Sprintf("deps(%s, 1)", label))
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	return parseLabels(string(output), label)
}

// querySystemLibraries extracts system libraries from linkopts
func querySystemLibraries(workspace string, label string) []string {
	// Use buildozer to read linkopts if available, otherwise return empty
	// For now, we'll use a simple heuristic based on common system libs

	// Try to get build file content and parse linkopts
	cmd := exec.Command("bazel", "query", "--output=build", label)
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	return extractSystemLibraries(string(output))
}

// extractSystemLibraries parses system libraries from build output
func extractSystemLibraries(buildOutput string) []string {
	var sysLibs []string
	seen := make(map[string]bool)

	lines := strings.Split(buildOutput, "\n")
	for _, line := range lines {
		// Look for linkopts lines containing -l flags
		if strings.Contains(line, "-l") {
			// Extract -l flags
			fields := strings.Fields(line)
			for _, field := range fields {
				field = strings.Trim(field, `"',[]`)
				if strings.HasPrefix(field, "-l") {
					lib := strings.TrimPrefix(field, "-l")
					if lib != "" && !seen[lib] {
						seen[lib] = true
						sysLibs = append(sysLibs, lib)
					}
				}
			}
		}
	}

	return sysLibs
}

// parseLabels extracts target labels from bazel query output
func parseLabels(output string, exclude string) []string {
	var labels []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, status messages, and the queried label itself
		if line != "" && strings.HasPrefix(line, "//") && line != exclude {
			labels = append(labels, line)
		}
	}
	return labels
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// GetAllBinariesInfo retrieves information for all binaries
func GetAllBinariesInfo(workspace string) ([]*BinaryInfo, error) {
	labels, err := QueryAllBinaries(workspace)
	if err != nil {
		return nil, err
	}

	var binaries []*BinaryInfo
	for _, label := range labels {
		info, err := GetBinaryInfo(workspace, label)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to get info for %s: %v\n", label, err)
			continue
		}
		binaries = append(binaries, info)
	}

	return binaries, nil
}
