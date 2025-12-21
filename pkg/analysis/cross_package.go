package analysis

import (
	"path/filepath"
	"strings"

	"github.com/ritzau/deps-analyzer/pkg/graph"
)

// CrossPackageDep represents a file dependency that crosses package boundaries
type CrossPackageDep struct {
	SourceFile    string `json:"sourceFile"`    // e.g., "core/engine.cc"
	TargetFile    string `json:"targetFile"`    // e.g., "util/strings.h"
	SourcePackage string `json:"sourcePackage"` // e.g., "//core"
	TargetPackage string `json:"targetPackage"` // e.g., "//util"
	SourceTarget  string `json:"sourceTarget"`  // e.g., "//core:core" (full target label if known)
	TargetTarget  string `json:"targetTarget"`  // e.g., "//util:util" (full target label if known)
}

// FindCrossPackageDeps identifies file dependencies that cross package boundaries
// Deprecated: Use FindCrossPackageDepsWithTargets for more accurate target labels
func FindCrossPackageDeps(fg *graph.FileGraph) []CrossPackageDep {
	return FindCrossPackageDepsWithTargets(fg, nil)
}

// FindCrossPackageDepsWithTargets identifies file dependencies that cross package boundaries
// and includes full target labels when fileToTarget mapping is provided
func FindCrossPackageDepsWithTargets(fg *graph.FileGraph, fileToTarget map[string]string) []CrossPackageDep {
	var crossDeps []CrossPackageDep

	edges := fg.Edges()
	for _, edge := range edges {
		sourceFile := edge[0]
		targetFile := edge[1]

		// Determine packages
		sourcePackage := fileToPackage(sourceFile)
		targetPackage := fileToPackage(targetFile)

		// If packages differ, this is a cross-package dependency
		if sourcePackage != targetPackage {
			dep := CrossPackageDep{
				SourceFile:    sourceFile,
				TargetFile:    targetFile,
				SourcePackage: sourcePackage,
				TargetPackage: targetPackage,
			}

			// Add full target labels if mapping is provided
			if fileToTarget != nil {
				if sourceTarget, ok := fileToTarget[sourceFile]; ok {
					dep.SourceTarget = sourceTarget
				}
				if targetTarget, ok := fileToTarget[targetFile]; ok {
					dep.TargetTarget = targetTarget
				}
			}

			crossDeps = append(crossDeps, dep)
		}
	}

	return crossDeps
}

// fileToPackage converts a file path to its Bazel package
// e.g., "util/strings.h" -> "//util"
// e.g., "core/engine.cc" -> "//core"
func fileToPackage(filePath string) string {
	dir := filepath.Dir(filePath)
	if dir == "." {
		return "//"
	}

	// Get the first directory component
	parts := strings.Split(dir, string(filepath.Separator))
	if len(parts) > 0 {
		return "//" + parts[0]
	}

	return "//"
}
