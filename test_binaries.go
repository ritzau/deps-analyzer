package main

import (
	"fmt"
	"github.com/ritzau/deps-analyzer/pkg/binaries"
)

func main() {
	workspace := "./example"
	
	bins, err := binaries.GetAllBinariesInfo(workspace)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	for _, bin := range bins {
		fmt.Printf("\n=== %s (%s) ===\n", bin.Label, bin.Kind)
		if len(bin.DynamicDeps) > 0 {
			fmt.Printf("  Dynamic deps: %v\n", bin.DynamicDeps)
		}
		if len(bin.DataDeps) > 0 {
			fmt.Printf("  Data deps: %v\n", bin.DataDeps)
		}
		if len(bin.SystemLibraries) > 0 {
			fmt.Printf("  System libs: %v\n", bin.SystemLibraries)
		}
	}
}
