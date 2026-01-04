package bazel

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverSourceFiles finds all .cc and .h files using git ls-files
// It respects .gitignore and includes both tracked and untracked-but-not-ignored files
func DiscoverSourceFiles(workspaceRoot string) (map[string]bool, error) {
	discovered := make(map[string]bool)

	// Get tracked files
	trackedFiles, err := runGitLsFiles(workspaceRoot, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracked files: %w", err)
	}

	// Get untracked but not ignored files
	untrackedFiles, err := runGitLsFiles(workspaceRoot, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get untracked files: %w", err)
	}

	// Merge both lists
	allFiles := append(trackedFiles, untrackedFiles...)

	// Find all package directories (directories with BUILD files)
	packageDirs, err := findPackageDirectories(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find package directories: %w", err)
	}

	// Filter for C++ source files in package directories
	for _, file := range allFiles {
		// Check if it's a C++ source file
		if !isCppSourceFile(file) {
			continue
		}

		// Check if file is in a package directory
		fileDir := filepath.Dir(file)
		if fileDir == "." {
			fileDir = ""
		}

		if isInPackage(fileDir, packageDirs) {
			discovered[file] = true
		}
	}

	return discovered, nil
}

// FindUncoveredFiles compares discovered files against tracked files
// Returns files that exist in the workspace but are not included in any target
func FindUncoveredFiles(discovered map[string]bool, fileToTarget map[string]string) []string {
	var uncovered []string

	for file := range discovered {
		if _, exists := fileToTarget[file]; !exists {
			uncovered = append(uncovered, file)
		}
	}

	// Sort for consistent output
	sort.Strings(uncovered)
	return uncovered
}

// runGitLsFiles executes git ls-files and returns the list of files
func runGitLsFiles(workspaceRoot string, untrackedOnly bool) ([]string, error) {
	var cmd *exec.Cmd
	if untrackedOnly {
		// Get untracked files that are not ignored
		cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	} else {
		// Get tracked files
		cmd = exec.Command("git", "ls-files")
	}
	cmd.Dir = workspaceRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}

	return files, scanner.Err()
}

// findPackageDirectories finds all directories containing BUILD or BUILD.bazel files
func findPackageDirectories(workspaceRoot string) (map[string]bool, error) {
	packages := make(map[string]bool)

	// Find all BUILD files using git ls-files (faster than walking filesystem)
	cmd := exec.Command("git", "ls-files", "BUILD", "BUILD.bazel", "**/BUILD", "**/BUILD.bazel")
	cmd.Dir = workspaceRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to find BUILD files: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		buildFile := strings.TrimSpace(scanner.Text())
		if buildFile == "" {
			continue
		}

		// Get directory containing BUILD file
		dir := filepath.Dir(buildFile)
		if dir == "." {
			dir = ""
		}
		packages[dir] = true
	}

	return packages, scanner.Err()
}

// isCppSourceFile checks if a file has a C++ source extension
func isCppSourceFile(file string) bool {
	ext := strings.ToLower(filepath.Ext(file))
	return ext == ".cc" || ext == ".h" || ext == ".hpp"
}

// isInPackage checks if a directory is in a package or its subdirectories
func isInPackage(fileDir string, packageDirs map[string]bool) bool {
	// Check exact match
	if packageDirs[fileDir] {
		return true
	}

	// Check if file is in a subdirectory of a package
	for pkgDir := range packageDirs {
		if pkgDir == "" {
			// Root package contains everything
			return true
		}
		if strings.HasPrefix(fileDir, pkgDir+"/") {
			return true
		}
	}

	return false
}
