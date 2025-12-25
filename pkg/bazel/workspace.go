package bazel

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GetWorkspaceName attempts to determine the workspace/module name from:
// 1. `bazel mod graph` command (if using Bazel modules/bzlmod)
// 2. Directory name as fallback
func GetWorkspaceName(workspacePath string) (string, error) {
	// Try to get module name from `bazel mod graph`
	moduleName, err := extractModuleNameFromBazel(workspacePath)
	if err == nil && moduleName != "" {
		return moduleName, nil
	}

	// Fallback: use directory name
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", err
	}

	dirName := filepath.Base(absPath)

	// Convert "." to actual directory name
	if dirName == "." {
		// Get parent directory's name
		parent := filepath.Dir(absPath)
		dirName = filepath.Base(parent)
	}

	return dirName, nil
}

// extractModuleNameFromBazel runs `bazel mod graph` and extracts the root module name
// Output format: <root> (module_name@version)
func extractModuleNameFromBazel(workspacePath string) (string, error) {
	cmd := exec.Command("bazel", "mod", "graph")
	cmd.Dir = workspacePath

	output, err := cmd.Output()
	if err != nil {
		return "", err // bazel mod graph failed (maybe not using bzlmod)
	}

	// Parse output: look for "<root> (module_name@version)"
	// Regex to match: <root> (name@version) or just (name)
	rootRegex := regexp.MustCompile(`<root>\s+\(([^@)]+)`)

	if matches := rootRegex.FindStringSubmatch(string(output)); len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	return "", nil
}
