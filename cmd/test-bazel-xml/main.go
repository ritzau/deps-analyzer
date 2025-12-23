package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ritzau/deps-analyzer/pkg/bazel"
	"github.com/ritzau/deps-analyzer/pkg/model"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: test-bazel-xml <workspace-path>")
	}

	workspacePath := os.Args[1]

	fmt.Println("Querying Bazel workspace...")
	module, err := bazel.QueryWorkspace(workspacePath)
	if err != nil {
		log.Fatalf("Failed to query workspace: %v", err)
	}

	// Add compile-time dependencies from .d files
	fmt.Println("Adding compile-time dependencies from .d files...")
	if err := bazel.AddCompileDependencies(module, workspacePath); err != nil {
		// Don't fail if .d files aren't available, just warn
		fmt.Printf("Warning: Could not add compile dependencies: %v\n", err)
	}

	// Add symbol-level dependencies from nm analysis
	fmt.Println("Adding symbol-level dependencies from nm analysis...")
	if err := bazel.AddSymbolDependencies(module, workspacePath); err != nil {
		// Don't fail if object files aren't available, just warn
		fmt.Printf("Warning: Could not add symbol dependencies: %v\n", err)
	}

	fmt.Printf("\nFound %d targets:\n", len(module.Targets))
	for _, target := range module.Targets {
		fmt.Printf("  %s (%s)\n", target.Label, target.Kind)
		if len(target.Sources) > 0 {
			fmt.Printf("    Sources: %v\n", target.Sources)
		}
		if len(target.Linkopts) > 0 {
			fmt.Printf("    Linkopts: %v\n", target.Linkopts)
		}
	}

	fmt.Printf("\nFound %d dependencies:\n", len(module.Dependencies))
	// Group by type
	byType := make(map[model.DependencyType][]model.Dependency)
	for _, dep := range module.Dependencies {
		byType[dep.Type] = append(byType[dep.Type], dep)
	}

	// Print in a specific order for consistency
	depTypeOrder := []model.DependencyType{
		model.DependencyStatic,
		model.DependencyDynamic,
		model.DependencyData,
		model.DependencyCompile,
		model.DependencySymbol,
	}

	for _, depType := range depTypeOrder {
		if deps, ok := byType[depType]; ok {
			fmt.Printf("  %s: %d\n", depType, len(deps))
			for _, dep := range deps {
				fmt.Printf("    %s -> %s\n", dep.From, dep.To)
			}
		}
	}

	// Show package-level dependencies
	packages := module.GetPackages()
	fmt.Printf("\nFound %d packages:\n", len(packages))
	for _, pkg := range packages {
		fmt.Printf("  %s (%d targets)\n", pkg.Path, len(pkg.Targets))
	}

	// Get all package dependencies
	fmt.Println("\nPackage-to-package dependencies:")
	pkgDeps := module.GetAllPackageDependencies()
	for _, pkgDep := range pkgDeps {
		edgeCount := 0
		for _, edges := range pkgDep.Dependencies {
			edgeCount += len(edges)
		}
		fmt.Printf("  %s -> %s (%d edges)\n", pkgDep.From, pkgDep.To, edgeCount)

		// Show breakdown by type
		for depType, edges := range pkgDep.Dependencies {
			fmt.Printf("    %s: %d\n", depType, len(edges))
		}
	}

	// Show dependency issues
	if len(module.Issues) > 0 {
		fmt.Printf("\n⚠️  Found %d dependency issues:\n", len(module.Issues))
		for _, issue := range module.Issues {
			fmt.Printf("  [%s] %s -> %s\n", issue.Severity, issue.From, issue.To)
			fmt.Printf("    Issue: %s\n", issue.Issue)
			fmt.Printf("    Types: %v\n", issue.Types)
			fmt.Printf("    %s\n", issue.Description)
		}
	} else {
		fmt.Println("\n✓ No dependency issues found")
	}

	// Write to JSON for inspection
	jsonData, err := json.MarshalIndent(module, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile("module.json", jsonData, 0644); err != nil {
		log.Fatalf("Failed to write JSON: %v", err)
	}

	fmt.Println("\nModule data written to module.json")
}
