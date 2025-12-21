package analysis

import (
	"github.com/ritzau/deps-analyzer/pkg/graph"
)

// FileInTarget represents a file belonging to a target
type FileInTarget struct {
	Path string `json:"path"`
	Type string `json:"type"` // "source" or "header"
}

// FileDependency represents a dependency between files across targets
type FileDependencyDetail struct {
	SourceFile   string `json:"sourceFile"`
	TargetFile   string `json:"targetFile"`
	SourceTarget string `json:"sourceTarget"`
	TargetTarget string `json:"targetTarget"`
}

// TargetFileDetails contains detailed file-level information for a target
type TargetFileDetails struct {
	TargetLabel       string                 `json:"targetLabel"`
	Files             []FileInTarget         `json:"files"`
	IncomingFileDeps  []FileDependencyDetail `json:"incomingFileDeps"`  // Files from other targets depending on this target's files
	OutgoingFileDeps  []FileDependencyDetail `json:"outgoingFileDeps"`  // This target's files depending on files in other targets
}

// GetTargetFileDetails analyzes file-level dependencies for a specific target
func GetTargetFileDetails(targetLabel string, fileGraph *graph.FileGraph, crossPackageDeps []CrossPackageDep) *TargetFileDetails {
	details := &TargetFileDetails{
		TargetLabel:      targetLabel,
		Files:            make([]FileInTarget, 0),
		IncomingFileDeps: make([]FileDependencyDetail, 0),
		OutgoingFileDeps: make([]FileDependencyDetail, 0),
	}

	// Extract package name from target label (e.g., "//util:util" -> "util")
	targetPackage := extractPackage(targetLabel)

	// Find all files in the file graph and categorize them
	for _, node := range fileGraph.Nodes() {
		filePackage := fileToPackage(node.Path)

		// Check if this file belongs to the target's package
		if filePackage == targetPackage {
			fileType := "source"
			if isHeaderFile(node.Path) {
				fileType = "header"
			}
			details.Files = append(details.Files, FileInTarget{
				Path: node.Path,
				Type: fileType,
			})
		}
	}

	// Create a map of files in this target for quick lookup
	filesInTarget := make(map[string]bool)
	for _, file := range details.Files {
		filesInTarget[file.Path] = true
	}

	// Analyze cross-package dependencies to find incoming/outgoing file deps
	for _, dep := range crossPackageDeps {
		// Incoming: other targets depending on this target's files
		if filesInTarget[dep.TargetFile] {
			// Use full target label if available, otherwise fall back to package
			sourceTarget := dep.SourceTarget
			if sourceTarget == "" {
				sourceTarget = dep.SourcePackage
			}

			details.IncomingFileDeps = append(details.IncomingFileDeps, FileDependencyDetail{
				SourceFile:   dep.SourceFile,
				TargetFile:   dep.TargetFile,
				SourceTarget: sourceTarget,
				TargetTarget: targetLabel, // Use actual target label for current target
			})
		}

		// Outgoing: this target's files depending on other targets
		if filesInTarget[dep.SourceFile] {
			// Use full target label if available, otherwise fall back to package
			targetTarget := dep.TargetTarget
			if targetTarget == "" {
				targetTarget = dep.TargetPackage
			}

			details.OutgoingFileDeps = append(details.OutgoingFileDeps, FileDependencyDetail{
				SourceFile:   dep.SourceFile,
				TargetFile:   dep.TargetFile,
				SourceTarget: targetLabel, // Use actual target label for current target
				TargetTarget: targetTarget,
			})
		}
	}

	return details
}

// extractPackage extracts the package name from a target label
// e.g., "//util:util" -> "//util", "//core/engine:engine" -> "//core/engine"
func extractPackage(targetLabel string) string {
	// Split on ":" to get the package part
	for i := 0; i < len(targetLabel); i++ {
		if targetLabel[i] == ':' {
			return targetLabel[:i]
		}
	}

	return targetLabel
}

// isHeaderFile checks if a file is a header file
func isHeaderFile(path string) bool {
	return len(path) > 2 && (path[len(path)-2:] == ".h" ||
		len(path) > 4 && path[len(path)-4:] == ".hpp")
}
