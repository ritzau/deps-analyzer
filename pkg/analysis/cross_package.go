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
}

// FindCrossPackageDeps identifies file dependencies that cross package boundaries
func FindCrossPackageDeps(fg *graph.FileGraph) []CrossPackageDep {
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
			crossDeps = append(crossDeps, CrossPackageDep{
				SourceFile:    sourceFile,
				TargetFile:    targetFile,
				SourcePackage: sourcePackage,
				TargetPackage: targetPackage,
			})
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
