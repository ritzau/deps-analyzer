package symbols

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Symbol represents a symbol extracted from an object file
type Symbol struct {
	Name    string // Symbol name (e.g., "_Z3foov" or "foo")
	Type    string // Symbol type (T, U, D, B, etc.)
	Address string // Address (if applicable)
	File    string // Source file that defines/uses this symbol
}

// LinkageType describes how a symbol is linked
type LinkageType string

const (
	LinkageStatic  LinkageType = "static"  // Same binary, statically linked
	LinkageDynamic LinkageType = "dynamic" // Different binary, dynamically linked via .so
	LinkageCross   LinkageType = "cross"   // Different binary, no clear linkage
)

// SymbolDependency represents a dependency through a symbol
// e.g., file A uses symbol X which is defined in file B
type SymbolDependency struct {
	SourceFile   string      `json:"sourceFile"`   // File that uses the symbol
	TargetFile   string      `json:"targetFile"`   // File that defines the symbol
	Symbol       string      `json:"symbol"`       // The symbol name
	SourceTarget string      `json:"sourceTarget"` // Bazel target of source file
	TargetTarget string      `json:"targetTarget"` // Bazel target of target file
	Linkage      LinkageType `json:"linkage"`      // How the symbol is linked
	SourceBinary string      `json:"sourceBinary"` // Which binary/library uses it
	TargetBinary string      `json:"targetBinary"` // Which binary/library defines it
}

// ParseNMOutput parses the output of nm command for a single object file
// nm output format: [address] <type> <symbol>
// Example: 0000000000000000 T _Z3foov
//          U _Z3barv
func ParseNMOutput(objectFile string, nmOutput string) []Symbol {
	var symbols []Symbol
	scanner := bufio.NewScanner(strings.NewReader(nmOutput))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		var symbol Symbol
		symbol.File = objectFile

		// Format can be:
		// "U symbol_name" (undefined)
		// "address T symbol_name" (defined)
		if len(parts) == 2 {
			// Undefined symbol (no address)
			symbol.Type = parts[0]
			symbol.Name = parts[1]
		} else if len(parts) >= 3 {
			// Defined symbol (has address)
			symbol.Address = parts[0]
			symbol.Type = parts[1]
			symbol.Name = parts[2]
		}

		symbols = append(symbols, symbol)
	}

	return symbols
}

// RunNM runs nm on an object file and returns the parsed symbols
func RunNM(objectFile string) ([]Symbol, error) {
	cmd := exec.Command("nm", objectFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("nm failed for %s: %w", objectFile, err)
	}

	return ParseNMOutput(objectFile, string(output)), nil
}

// FindObjectFiles searches for .o files in the bazel output directories
// Typically in bazel-out/darwin-fastbuild/bin/... or similar
func FindObjectFiles(workspaceRoot string) ([]string, error) {
	var objectFiles []string

	// Common Bazel output paths
	bazelOutDirs := []string{
		filepath.Join(workspaceRoot, "bazel-out"),
		filepath.Join(workspaceRoot, "bazel-bin"),
	}

	for _, dir := range bazelOutDirs {
		// Use find command to locate .o files
		cmd := exec.Command("find", dir, "-name", "*.o", "-type", "f")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Directory might not exist, continue
			continue
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				objectFiles = append(objectFiles, line)
			}
		}
	}

	return objectFiles, nil
}

// BuildSymbolGraph analyzes all object files and builds symbol dependencies
// It also determines which binary/library each object file belongs to and the linkage type
func BuildSymbolGraph(workspaceRoot string, fileToTarget map[string]string, targetToKind map[string]string) ([]SymbolDependency, error) {
	// Find all .o files
	objectFiles, err := FindObjectFiles(workspaceRoot)
	if err != nil {
		return nil, err
	}

	if len(objectFiles) == 0 {
		return nil, fmt.Errorf("no object files found in %s", workspaceRoot)
	}

	// Map symbol names to the files that define them
	symbolDefinitions := make(map[string]string) // symbol -> defining file

	// Map files to their undefined symbols
	fileUndefinedSymbols := make(map[string][]string) // file -> undefined symbols

	// Process all object files
	for _, objFile := range objectFiles {
		symbols, err := RunNM(objFile)
		if err != nil {
			// Skip files we can't process
			continue
		}

		// Convert object file path to source file path
		sourceFile := objectFileToSourceFile(objFile, workspaceRoot)

		for _, sym := range symbols {
			if sym.Type == "U" {
				// Undefined symbol - this file needs it
				fileUndefinedSymbols[sourceFile] = append(fileUndefinedSymbols[sourceFile], sym.Name)
			} else if isDefinedSymbol(sym.Type) {
				// Defined symbol - this file provides it
				symbolDefinitions[sym.Name] = sourceFile
			}
		}
	}

	// Build dependencies: file A depends on file B if A uses symbol defined in B
	var symbolDeps []SymbolDependency

	for sourceFile, undefinedSyms := range fileUndefinedSymbols {
		for _, symName := range undefinedSyms {
			if definingFile, ok := symbolDefinitions[symName]; ok {
				// Found where this symbol is defined
				if sourceFile != definingFile {
					dep := SymbolDependency{
						SourceFile: sourceFile,
						TargetFile: definingFile,
						Symbol:     symName,
					}

					// Add target labels and determine linkage type
					if fileToTarget != nil {
						if srcTarget, ok := fileToTarget[sourceFile]; ok {
							dep.SourceTarget = srcTarget
							dep.SourceBinary = srcTarget // Use target as binary identifier
						}
						if tgtTarget, ok := fileToTarget[definingFile]; ok {
							dep.TargetTarget = tgtTarget
							dep.TargetBinary = tgtTarget // Use target as binary identifier
						}

						// Determine linkage type
						if dep.SourceTarget == dep.TargetTarget {
							// Same target = static linkage within same binary
							dep.Linkage = LinkageStatic
						} else if targetToKind != nil {
							// Different targets - check if target is a shared library
							sourceKind := targetToKind[dep.SourceTarget]
							targetKind := targetToKind[dep.TargetTarget]

							if targetKind == "cc_shared_library" || sourceKind == "cc_shared_library" {
								dep.Linkage = LinkageDynamic
							} else {
								// Different binaries, not shared library
								dep.Linkage = LinkageCross
							}
						} else {
							dep.Linkage = LinkageCross
						}
					}

					symbolDeps = append(symbolDeps, dep)
				}
			}
		}
	}

	return symbolDeps, nil
}

// objectFileToSourceFile converts an object file path to its source file path
// e.g., "bazel-out/darwin-fastbuild/bin/util/_objs/util/strings.o" -> "util/strings.cc"
func objectFileToSourceFile(objPath string, workspaceRoot string) string {
	// Extract the relative path and convert .o to source extension
	// This is a heuristic and may need adjustment based on actual Bazel structure
	base := filepath.Base(objPath)
	name := strings.TrimSuffix(base, ".o")

	// Try to extract package path from the object file path
	// Bazel typically puts objects in paths like: bazel-out/.../bin/package/_objs/target/file.o
	parts := strings.Split(objPath, string(filepath.Separator))

	var packagePath string
	for i, part := range parts {
		if part == "bin" && i+1 < len(parts) {
			// Everything after "bin" until "_objs" is the package path
			for j := i + 1; j < len(parts); j++ {
				if parts[j] == "_objs" {
					break
				}
				if packagePath != "" {
					packagePath += "/"
				}
				packagePath += parts[j]
			}
			break
		}
	}

	if packagePath != "" {
		// Try common C++ extensions
		for _, ext := range []string{".cc", ".cpp", ".c"} {
			candidate := filepath.Join(packagePath, name+ext)
			return candidate
		}
	}

	// Fallback: just use the base name with .cc
	return name + ".cc"
}

// isDefinedSymbol returns true if the symbol type indicates a definition
func isDefinedSymbol(symType string) bool {
	// T: text (code) section
	// D: initialized data
	// B: uninitialized data (BSS)
	// R: read-only data
	// W: weak symbol
	// Lowercase versions are local symbols
	switch symType {
	case "T", "t", "D", "d", "B", "b", "R", "r", "W", "w":
		return true
	default:
		return false
	}
}
