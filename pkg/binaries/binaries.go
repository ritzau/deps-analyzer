package binaries

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/ritzau/deps-analyzer/pkg/model"
)

// BinaryInfo represents a cc_binary or cc_shared_library
type BinaryInfo struct {
	Label             string              `json:"label"`
	Kind              string              `json:"kind"` // "cc_binary" or "cc_shared_library"
	DynamicDeps       []string            `json:"dynamicDeps"`
	DataDeps          []string            `json:"dataDeps"`
	SystemLibraries   []string            `json:"systemLibraries"`
	RegularDeps       []string            `json:"regularDeps"`      // Direct cc_library dependencies
	InternalTargets   []string            `json:"internalTargets"` // All cc_library targets this binary depends on
	OverlappingDeps   map[string][]string `json:"overlappingDeps"` // Map of binary -> overlapping cc_library targets (potential duplicate symbols)
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
	fmt.Printf("  - Querying rule kind...\n")
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
	fmt.Printf("  - Querying shared library dependencies...\n")
	sharedLibDeps := querySharedLibraryDeps(workspace, label)

	// Separate into dynamic_deps and data_deps based on how they're referenced
	// For now, we'll use a heuristic: query deps to see what's linked
	fmt.Printf("  - Querying linked dependencies...\n")
	linkedDeps := queryLinkedDeps(workspace, label)

	for _, dep := range sharedLibDeps {
		if contains(linkedDeps, dep) {
			info.DynamicDeps = append(info.DynamicDeps, dep)
		} else {
			info.DataDeps = append(info.DataDeps, dep)
		}
	}

	// Get system libraries from linkopts
	fmt.Printf("  - Querying system libraries...\n")
	info.SystemLibraries = querySystemLibraries(workspace, label)

	// Get all cc_library targets this binary depends on (excluding shared libraries)
	fmt.Printf("  - Querying internal cc_library targets...\n")
	info.InternalTargets = queryInternalTargets(workspace, label)

	// Get direct cc_library dependencies (depth 1)
	fmt.Printf("  - Querying direct dependencies...\n")
	info.RegularDeps = queryDirectDeps(workspace, label)

	return info, nil
}

// queryDirectDeps finds direct cc_library dependencies (depth 1)
func queryDirectDeps(workspace string, label string) []string {
	// Query for direct cc_library dependencies only
	cmd := exec.Command("bazel", "query",
		fmt.Sprintf("kind('cc_library', deps(%s, 1))", label))
	cmd.Dir = workspace
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	return parseLabels(string(output), label)
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
	fmt.Println("Querying for all cc_binary and cc_shared_library targets...")
	labels, err := QueryAllBinaries(workspace)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d binaries to analyze\n", len(labels))

	var binaries []*BinaryInfo
	for i, label := range labels {
		fmt.Printf("[%d/%d] Analyzing binary: %s\n", i+1, len(labels), label)
		info, err := GetBinaryInfo(workspace, label)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to get info for %s: %v\n", label, err)
			continue
		}
		binaries = append(binaries, info)
	}

	// Compute overlapping dependencies (potential duplicate symbols)
	fmt.Println("Computing overlapping dependencies...")
	computeOverlappingDeps(binaries)

	return binaries, nil
}

// computeOverlappingDeps finds cc_library targets that are linked into multiple binaries
// This can cause duplicate symbols if a binary loads a shared library that both depend on the same cc_library
func computeOverlappingDeps(binaries []*BinaryInfo) {
	for i, binary := range binaries {
		if binary.Kind != "cc_binary" {
			continue // Only check for cc_binary loading shared libraries
		}

		binary.OverlappingDeps = make(map[string][]string)

		// Check each dynamic dependency
		for _, depLabel := range binary.DynamicDeps {
			// Find the shared library
			var sharedLib *BinaryInfo
			for _, b := range binaries {
				if b.Label == depLabel {
					sharedLib = b
					break
				}
			}

			if sharedLib == nil {
				continue
			}

			// Find overlapping cc_library targets
			binaryTargets := toSet(binary.InternalTargets)
			var overlapping []string

			for _, target := range sharedLib.InternalTargets {
				if binaryTargets[target] {
					overlapping = append(overlapping, target)
				}
			}

			if len(overlapping) > 0 {
				binary.OverlappingDeps[depLabel] = overlapping
			}
		}

		binaries[i] = binary
	}
}

// toSet converts a slice to a set (map[string]bool)
func toSet(slice []string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range slice {
		set[item] = true
	}
	return set
}

// DeriveBinaryInfoFromModule creates BinaryInfo for all binaries and shared libraries from the Module
// This is much faster than running separate Bazel queries for each binary
func DeriveBinaryInfoFromModule(module *model.Module) []*BinaryInfo {
	var result []*BinaryInfo

	// Process each binary and shared library target
	for _, target := range module.Targets {
		if target.Kind != model.TargetKindBinary && target.Kind != model.TargetKindSharedLibrary {
			continue
		}

		info := &BinaryInfo{
			Label:           target.Label,
			Kind:            string(target.Kind),
			DynamicDeps:     make([]string, 0),
			DataDeps:        make([]string, 0),
			SystemLibraries: extractSystemLibrariesFromLinkopts(target.Linkopts),
			RegularDeps:     make([]string, 0),
			InternalTargets: make([]string, 0),
			OverlappingDeps: make(map[string][]string),
		}

		// Collect dependencies from module.Dependencies
		allLibraries := make(map[string]bool)          // All transitive cc_library dependencies
		dynamicLibs := make(map[string][]string)       // Track which libraries are in which dynamic deps

		for _, dep := range module.Dependencies {
			if dep.From != target.Label {
				continue
			}

			depTarget := module.Targets[dep.To]
			if depTarget == nil {
				continue
			}

			// Categorize by dependency type
			switch dep.Type {
			case model.DependencyDynamic:
				info.DynamicDeps = append(info.DynamicDeps, dep.To)
				// Collect libraries from this dynamic dep for overlap detection
				dynamicLibs[dep.To] = getTransitiveLibraries(module, dep.To)
			case model.DependencyData:
				info.DataDeps = append(info.DataDeps, dep.To)
			case model.DependencyStatic:
				if depTarget.Kind == model.TargetKindLibrary {
					info.RegularDeps = append(info.RegularDeps, dep.To)
				}
			}
		}

		// Get all transitive cc_library dependencies
		visited := make(map[string]bool)
		collectAllLibraries(module, target.Label, visited, allLibraries)
		for lib := range allLibraries {
			if lib != target.Label {
				info.InternalTargets = append(info.InternalTargets, lib)
			}
		}

		result = append(result, info)
	}

	// Compute overlapping dependencies
	computeOverlappingDeps(result)

	return result
}

// extractSystemLibrariesFromLinkopts extracts system libraries from linkopts
func extractSystemLibrariesFromLinkopts(linkopts []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, opt := range linkopts {
		if strings.HasPrefix(opt, "-l") {
			lib := strings.TrimPrefix(opt, "-l")
			if lib != "" && !seen[lib] {
				seen[lib] = true
				result = append(result, lib)
			}
		}
	}

	return result
}

// getTransitiveLibraries gets all transitive cc_library dependencies of a target
func getTransitiveLibraries(module *model.Module, targetLabel string) []string {
	visited := make(map[string]bool)
	libraries := make(map[string]bool)
	collectAllLibraries(module, targetLabel, visited, libraries)

	result := make([]string, 0, len(libraries))
	for lib := range libraries {
		if lib != targetLabel {
			result = append(result, lib)
		}
	}
	return result
}

// collectAllLibraries recursively collects all cc_library dependencies
func collectAllLibraries(module *model.Module, targetLabel string, visited map[string]bool, libraries map[string]bool) {
	if visited[targetLabel] {
		return
	}
	visited[targetLabel] = true

	target := module.Targets[targetLabel]
	if target != nil && target.Kind == model.TargetKindLibrary {
		libraries[targetLabel] = true
	}

	// Recursively collect from dependencies
	for _, dep := range module.Dependencies {
		if dep.From == targetLabel && dep.Type == model.DependencyStatic {
			collectAllLibraries(module, dep.To, visited, libraries)
		}
	}
}
