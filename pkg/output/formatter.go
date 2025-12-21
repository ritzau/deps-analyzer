package output

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/ritzau/deps-analyzer/pkg/analysis"
)

// PrintCoverageReport prints a nicely formatted coverage report with colors
func PrintCoverageReport(workspace string, totalFiles, coveredFiles int, uncovered []analysis.UncoveredFile) {
	// Color definitions
	bold := color.New(color.Bold)
	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)

	// Header
	bold.Println("Bazel C++ Analyzer - Coverage Report")
	bold.Println("=====================================")
	fmt.Printf("Workspace: %s\n", workspace)
	fmt.Printf("Scanned: %d source files\n", totalFiles)

	// Coverage stats with colors
	if len(uncovered) == 0 {
		green.Printf("Covered: %d files\n", coveredFiles)
		green.Printf("Uncovered: 0 files\n")
	} else {
		fmt.Printf("Covered: %d files\n", coveredFiles)
		yellow.Printf("Uncovered: %d file(s)\n", len(uncovered))
	}
	fmt.Println()

	// Uncovered files list
	if len(uncovered) > 0 {
		red.Println("UNCOVERED FILES:")
		for _, uf := range uncovered {
			yellow.Printf("  %s\n", uf.Path)
			cyan.Printf("    Package: %s\n", uf.Package)
			fmt.Printf("    Suggestion: Add to BUILD.bazel or remove if unused\n")
			fmt.Println()
		}
	}

	// Summary with color based on coverage percentage
	percentage := 100.0
	if totalFiles > 0 {
		percentage = float64(coveredFiles) / float64(totalFiles) * 100.0
	}

	summaryColor := green
	if percentage < 100.0 {
		summaryColor = yellow
	}
	if percentage < 80.0 {
		summaryColor = red
	}

	summaryColor.Printf("Summary: %.0f%% coverage (%d/%d files)\n", percentage, coveredFiles, totalFiles)

	// Success check mark if 100%
	if percentage == 100.0 {
		green.Println("✓ All files are covered by Bazel targets!")
	}
}
