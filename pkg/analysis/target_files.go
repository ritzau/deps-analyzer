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

	// Analyze cross-package dependencies to find incoming/outgoing file deps
	for _, dep := range crossPackageDeps {
		sourceTargetLabel := packageToTarget(dep.SourcePackage)
		targetTargetLabel := packageToTarget(dep.TargetPackage)

		// Incoming: other targets depending on this target
		if targetTargetLabel == targetLabel {
			details.IncomingFileDeps = append(details.IncomingFileDeps, FileDependencyDetail{
				SourceFile:   dep.SourceFile,
				TargetFile:   dep.TargetFile,
				SourceTarget: sourceTargetLabel,
				TargetTarget: targetTargetLabel,
			})
		}

		// Outgoing: this target depending on other targets
		if sourceTargetLabel == targetLabel {
			details.OutgoingFileDeps = append(details.OutgoingFileDeps, FileDependencyDetail{
				SourceFile:   dep.SourceFile,
				TargetFile:   dep.TargetFile,
				SourceTarget: sourceTargetLabel,
				TargetTarget: targetTargetLabel,
			})
		}
	}

	return details
}

// extractPackage extracts the package name from a target label
// e.g., "//util:util" -> "util", "//core/engine:engine" -> "core/engine"
func extractPackage(targetLabel string) string {
	// Remove leading "//"
	if len(targetLabel) > 2 && targetLabel[:2] == "//" {
		targetLabel = targetLabel[2:]
	}

	// Split on ":"
	for i := 0; i < len(targetLabel); i++ {
		if targetLabel[i] == ':' {
			return targetLabel[:i]
		}
	}

	return targetLabel
}

// packageToTarget converts a package name to a target label
// e.g., "util" -> "//util:util", "core/engine" -> "//core/engine:engine"
func packageToTarget(pkg string) string {
	// Extract the last component for the target name
	targetName := pkg
	for i := len(pkg) - 1; i >= 0; i-- {
		if pkg[i] == '/' {
			targetName = pkg[i+1:]
			break
		}
	}
	return "//" + pkg + ":" + targetName
}

// isHeaderFile checks if a file is a header file
func isHeaderFile(path string) bool {
	return len(path) > 2 && (path[len(path)-2:] == ".h" ||
		len(path) > 4 && path[len(path)-4:] == ".hpp")
}
