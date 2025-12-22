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
	RegularDeps       []string `json:"regularDeps"`
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
	// Query for rule kind and attributes
	query := fmt.Sprintf("kind('cc_binary|cc_shared_library', %s)", label)
	cmd := exec.Command("bazel", "query", "--output=label_kind", query)
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed for %s: %w", label, err)
	}

	// Parse kind from output (format: "rule_kind rule label")
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected query output format: %s", string(output))
	}
	kind := parts[1]

	info := &BinaryInfo{
		Label: label,
		Kind:  kind,
	}

	// Get dynamic_deps if any
	dynamicDeps, _ := queryAttribute(workspace, label, "dynamic_deps")
	info.DynamicDeps = dynamicDeps

	// Get data dependencies (filter for shared libraries)
	dataDeps, _ := queryAttribute(workspace, label, "data")
	for _, dep := range dataDeps {
		// Check if this data dep is a shared library
		if isSharedLibrary(workspace, dep) {
			info.DataDeps = append(info.DataDeps, dep)
		}
	}

	// Get linkopts to extract system libraries
	linkopts, _ := queryLinkopts(workspace, label)
	info.SystemLibraries = extractSystemLibraries(linkopts)

	// Get regular cc_library deps
	regularDeps, _ := queryAttribute(workspace, label, "deps")
	info.RegularDeps = regularDeps

	return info, nil
}

// queryAttribute queries a specific attribute of a target
func queryAttribute(workspace string, label string, attr string) ([]string, error) {
	query := fmt.Sprintf("attr(%s, '', %s)", attr, label)
	cmd := exec.Command("bazel", "query", "--output=label", query)
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Attribute might not exist, return empty
		return nil, nil
	}

	// Try to get the actual deps by querying deps() with appropriate filters
	var depsQuery string
	switch attr {
	case "dynamic_deps":
		depsQuery = fmt.Sprintf("kind('cc_shared_library', deps(%s, 1))", label)
	case "data":
		depsQuery = fmt.Sprintf("attr(data, '', %s)", label)
	case "deps":
		depsQuery = fmt.Sprintf("kind('cc_library', deps(%s, 1))", label)
	default:
		return nil, nil
	}

	cmd = exec.Command("bazel", "query", "--output=label", depsQuery)
	cmd.Dir = workspace
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var results []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != label {
			results = append(results, line)
		}
	}

	return results, nil
}

// queryLinkopts gets linkopts for a target
func queryLinkopts(workspace string, label string) ([]string, error) {
	// Use cquery to get build information including linkopts
	cmd := exec.Command("bazel", "cquery",
		"--output=jsonproto",
		label)
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// Parse the JSON to extract linkopts
	// This is a simplified version - in reality we'd parse the proto JSON
	outputStr := string(output)

	// Look for linkopts patterns in the output
	var linkopts []string
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "-l") {
			// Extract -l flags
			fields := strings.Fields(line)
			for _, field := range fields {
				if strings.HasPrefix(field, "-l") {
					linkopts = append(linkopts, field)
				}
			}
		}
	}

	return linkopts, nil
}

// extractSystemLibraries extracts system library names from linkopts
func extractSystemLibraries(linkopts []string) []string {
	var sysLibs []string
	seen := make(map[string]bool)

	for _, opt := range linkopts {
		opt = strings.TrimSpace(opt)
		// Match -lfoo or -l foo patterns
		if strings.HasPrefix(opt, "-l") {
			lib := strings.TrimPrefix(opt, "-l")
			lib = strings.Trim(lib, `"'`)
			if lib != "" && !seen[lib] {
				seen[lib] = true
				sysLibs = append(sysLibs, lib)
			}
		}
	}

	return sysLibs
}

// isSharedLibrary checks if a label is a cc_shared_library
func isSharedLibrary(workspace string, label string) bool {
	cmd := exec.Command("bazel", "query", "--output=label_kind", label)
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "cc_shared_library")
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
