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
	workspace, err := bazel.QueryWorkspace(workspacePath)
	if err != nil {
		log.Fatalf("Failed to query workspace: %v", err)
	}

	fmt.Printf("\nFound %d targets:\n", len(workspace.Targets))
	for _, target := range workspace.Targets {
		fmt.Printf("  %s (%s)\n", target.Label, target.Kind)
		if len(target.Sources) > 0 {
			fmt.Printf("    Sources: %v\n", target.Sources)
		}
		if len(target.Deps) > 0 {
			fmt.Printf("    Deps: %v\n", target.Deps)
		}
		if len(target.DynamicDeps) > 0 {
			fmt.Printf("    DynamicDeps: %v\n", target.DynamicDeps)
		}
		if len(target.Data) > 0 {
			fmt.Printf("    Data: %v\n", target.Data)
		}
		if len(target.Linkopts) > 0 {
			fmt.Printf("    Linkopts: %v\n", target.Linkopts)
		}
	}

	fmt.Printf("\nFound %d dependencies:\n", len(workspace.Dependencies))
	// Group by type
	byType := make(map[model.DependencyType][]model.Dependency)
	for _, dep := range workspace.Dependencies {
		byType[dep.Type] = append(byType[dep.Type], dep)
	}

	for depType, deps := range byType {
		fmt.Printf("  %s: %d\n", depType, len(deps))
		for _, dep := range deps {
			fmt.Printf("    %s -> %s\n", dep.From, dep.To)
		}
	}

	// Show package-level dependencies
	fmt.Printf("\nFound %d packages:\n", len(workspace.Packages))
	for _, pkg := range workspace.Packages {
		fmt.Printf("  %s (%d targets)\n", pkg.Path, len(pkg.Targets))
	}

	// Get all package dependencies
	fmt.Println("\nPackage-to-package dependencies:")
	pkgDeps := workspace.GetAllPackageDependencies()
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

	// Write to JSON for inspection
	jsonData, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile("workspace.json", jsonData, 0644); err != nil {
		log.Fatalf("Failed to write JSON: %v", err)
	}

	fmt.Println("\nWorkspace data written to workspace.json")
}
