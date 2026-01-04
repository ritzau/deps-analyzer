package bazel

import (
	"encoding/xml"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ritzau/deps-analyzer/pkg/deps"
	"github.com/ritzau/deps-analyzer/pkg/model"
	"github.com/ritzau/deps-analyzer/pkg/symbols"
)

// QueryResult represents the XML output from bazel query
type QueryResult struct {
	XMLName xml.Name  `xml:"query"`
	Rules   []RuleXML `xml:"rule"`
}

// RuleXML represents a single rule in the XML output
type RuleXML struct {
	Class    string      `xml:"class,attr"`
	Name     string      `xml:"name,attr"`
	Location string      `xml:"location,attr"`
	Lists    []ListXML   `xml:"list"`
	Strings  []StringXML `xml:"string"`
}

// ListXML represents a list attribute in the XML
type ListXML struct {
	Name    string      `xml:"name,attr"`
	Labels  []LabelXML  `xml:"label"`
	Strings []StringXML `xml:"string"`
}

// LabelXML represents a label in the XML
type LabelXML struct {
	Value string `xml:"value,attr"`
}

// StringXML represents a string value in the XML
type StringXML struct {
	Value string `xml:"value,attr"`
}

// QueryWorkspace queries all cc_* targets and their dependencies
func QueryWorkspace(workspacePath string) (*model.Module, error) {
	// Query all cc_binary, cc_shared_library, and cc_library targets
	cmd := exec.Command("bazel", "query",
		"kind('cc_binary|cc_shared_library|cc_library', //...)",
		"--output=xml")
	cmd.Dir = workspacePath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed: %w\nOutput: %s", err, string(output))
	}

	// Bazel outputs XML 1.1, but Go's XML parser only supports 1.0
	// Replace the version declaration
	xmlStr := string(output)
	xmlStr = strings.Replace(xmlStr, `<?xml version="1.1"`, `<?xml version="1.0"`, 1)

	// Parse XML
	var result QueryResult
	if err := xml.Unmarshal([]byte(xmlStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Build module structure
	module := &model.Module{
		Targets:      make(map[string]*model.Target),
		Dependencies: make([]model.Dependency, 0),
		Issues:       make([]model.DependencyIssue, 0),
	}

	// Get workspace/module name
	workspaceName, err := GetWorkspaceName(workspacePath)
	if err != nil {
		// Log warning but don't fail - use default
		fmt.Printf("Warning: could not determine workspace name: %v\n", err)
		workspaceName = filepath.Base(workspacePath)
	}
	module.Name = workspaceName

	// Get absolute path to workspace
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		absPath = workspacePath // Fall back to original path if absolute path fails
	}
	module.WorkspacePath = absPath

	// First pass: create all targets
	for _, rule := range result.Rules {
		target := parseTarget(rule)
		if target != nil {
			module.Targets[target.Label] = target
		}
	}

	// Collect all external dependencies referenced by workspace targets
	externalDeps := collectExternalDependencies(result.Rules)

	// Query external dependencies and add them to the module
	var externalRules []RuleXML
	if len(externalDeps) > 0 {
		externalTargets, rules, err := queryExternalTargets(workspacePath, externalDeps)
		if err != nil {
			// Log warning but don't fail - external deps are optional
			fmt.Printf("Warning: failed to query external dependencies: %v\n", err)
		} else {
			// Add external targets to module
			for _, target := range externalTargets {
				module.Targets[target.Label] = target
			}
			externalRules = rules
		}
	}

	// Second pass: create typed dependencies from workspace targets
	for _, rule := range result.Rules {
		deps := parseDependencies(rule, module.Targets)
		module.Dependencies = append(module.Dependencies, deps...)
	}

	// Third pass: create typed dependencies from external targets
	for _, rule := range externalRules {
		deps := parseDependencies(rule, module.Targets)
		module.Dependencies = append(module.Dependencies, deps...)
	}

	return module, nil
}

// collectExternalDependencies extracts all external dependency labels from rules
func collectExternalDependencies(rules []RuleXML) []string {
	externalDeps := make(map[string]bool)

	for _, rule := range rules {
		for _, list := range rule.Lists {
			// Check deps, dynamic_deps, and data lists
			if list.Name == "deps" || list.Name == "dynamic_deps" || list.Name == "data" {
				for _, label := range list.Labels {
					// External dependencies start with @
					if strings.HasPrefix(label.Value, "@") {
						// Skip bazel_tools and other system repos
						if !strings.HasPrefix(label.Value, "@bazel_tools//") &&
							!strings.HasPrefix(label.Value, "@local_config_") &&
							!strings.HasPrefix(label.Value, "@platforms//") {
							externalDeps[label.Value] = true
						}
					}
				}
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(externalDeps))
	for dep := range externalDeps {
		result = append(result, dep)
	}
	return result
}

// queryExternalTargets queries Bazel for details about external targets
// Returns targets, rules, and error
func queryExternalTargets(workspacePath string, externalLabels []string) ([]*model.Target, []RuleXML, error) {
	if len(externalLabels) == 0 {
		return nil, nil, nil
	}

	// Build query expression: label1 + label2 + label3...
	queryExpr := strings.Join(externalLabels, " + ")

	cmd := exec.Command("bazel", "query", "--output=xml", queryExpr)
	cmd.Dir = workspacePath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("bazel query for external targets failed: %w\nOutput: %s", err, string(output))
	}

	// Parse XML
	xmlStr := string(output)
	xmlStr = strings.Replace(xmlStr, `<?xml version="1.1"`, `<?xml version="1.0"`, 1)

	var result QueryResult
	if err := xml.Unmarshal([]byte(xmlStr), &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse external targets XML: %w", err)
	}

	// Parse targets
	targets := make([]*model.Target, 0, len(result.Rules))
	for _, rule := range result.Rules {
		target := parseTarget(rule)
		if target != nil {
			targets = append(targets, target)
		}
	}

	return targets, result.Rules, nil
}

// parseTarget converts RuleXML to Target
func parseTarget(rule RuleXML) *model.Target {
	// Only process cc_binary, cc_shared_library, cc_library
	kind := model.TargetKind(rule.Class)
	if kind != model.TargetKindBinary && kind != model.TargetKindSharedLibrary && kind != model.TargetKindLibrary {
		return nil
	}

	label := rule.Name
	parts := strings.Split(label, ":")
	packagePath := label
	targetName := ""
	if len(parts) == 2 {
		packagePath = parts[0]
		targetName = parts[1]
	}

	target := &model.Target{
		Label:   label,
		Kind:    kind,
		Package: packagePath,
		Name:    targetName,
	}

	// Skip file parsing for external targets (labels starting with @)
	// External packages don't have file-level dependency information
	isExternalTarget := strings.HasPrefix(label, "@")

	// Extract attributes from lists
	for _, list := range rule.Lists {
		switch list.Name {
		case "srcs":
			if !isExternalTarget {
				for _, label := range list.Labels {
					if strings.HasSuffix(label.Value, ".cc") {
						target.Sources = append(target.Sources, label.Value)
					} else if strings.HasSuffix(label.Value, ".h") || strings.HasSuffix(label.Value, ".hpp") {
						target.Headers = append(target.Headers, label.Value)
					}
				}
			}
		case "hdrs":
			if !isExternalTarget {
				for _, label := range list.Labels {
					if strings.HasSuffix(label.Value, ".h") || strings.HasSuffix(label.Value, ".hpp") {
						target.Headers = append(target.Headers, label.Value)
					}
				}
			}
		case "linkopts":
			for _, str := range list.Strings {
				target.Linkopts = append(target.Linkopts, str.Value)
			}
		case "visibility":
			for _, label := range list.Labels {
				target.Visibility = append(target.Visibility, label.Value)
			}
		}
	}

	return target
}

// parseDependencies creates typed dependency edges for a target
func parseDependencies(rule RuleXML, targets map[string]*model.Target) []model.Dependency {
	fromLabel := rule.Name
	var deps []model.Dependency

	for _, list := range rule.Lists {
		switch list.Name {
		case "deps":
			// Regular deps - determine type based on target kind
			for _, label := range list.Labels {
				depType := determineDependencyType(label.Value, targets)
				deps = append(deps, model.Dependency{
					From: fromLabel,
					To:   label.Value,
					Type: depType,
				})
			}

		case "dynamic_deps":
			// Explicit dynamic dependencies
			for _, label := range list.Labels {
				deps = append(deps, model.Dependency{
					From: fromLabel,
					To:   label.Value,
					Type: model.DependencyDynamic,
				})
			}

		case "data":
			// Data dependencies (runtime)
			for _, label := range list.Labels {
				deps = append(deps, model.Dependency{
					From: fromLabel,
					To:   label.Value,
					Type: model.DependencyData,
				})
			}
		}
	}

	return deps
}

// determineDependencyType determines if a dep is static or dynamic based on target kind
func determineDependencyType(depLabel string, targets map[string]*model.Target) model.DependencyType {
	depTarget, exists := targets[depLabel]
	if !exists {
		// If we don't know the target, assume static (cc_library)
		return model.DependencyStatic
	}

	switch depTarget.Kind {
	case model.TargetKindLibrary:
		return model.DependencyStatic
	case model.TargetKindSharedLibrary:
		return model.DependencyDynamic
	case model.TargetKindBinary:
		// Depending on a binary is unusual, treat as data
		return model.DependencyData
	default:
		return model.DependencyStatic
	}
}

// AddCompileDependencies adds compile-time dependencies from .d files to the module
func AddCompileDependencies(module *model.Module, workspacePath string) error {
	// Parse all .d files
	fileDeps, err := deps.ParseAllDFiles(workspacePath)
	if err != nil {
		return fmt.Errorf("parsing .d files: %w", err)
	}

	// Build a map from file paths to targets
	fileToTarget := make(map[string]*model.Target)
	for _, target := range module.Targets {
		// Map source files to their target
		for _, src := range target.Sources {
			// Normalize the path - src is like "//main:main.cc"
			// We need to extract just the file path part
			filePath := NormalizeSourcePath(src)
			fileToTarget[filePath] = target
		}
		// Map header files to their target
		for _, hdr := range target.Headers {
			filePath := NormalizeSourcePath(hdr)
			fileToTarget[filePath] = target
		}
	}

	// Process each file dependency
	for _, fileDep := range fileDeps {
		// Find which target owns the source file
		sourceTarget := findTargetForFile(fileDep.SourceFile, fileToTarget)
		if sourceTarget == nil {
			continue // Skip files not associated with any target
		}

		// For each header dependency, find which target owns it
		for _, depFile := range fileDep.Dependencies {
			depTarget := findTargetForFile(depFile, fileToTarget)
			if depTarget == nil {
				continue // Skip external or unknown dependencies
			}

			// Skip dependencies within the same target
			if sourceTarget.Label == depTarget.Label {
				continue
			}

			// Check if this compile dependency already exists
			exists := false
			for _, dep := range module.Dependencies {
				if dep.From == sourceTarget.Label && dep.To == depTarget.Label && dep.Type == model.DependencyCompile {
					exists = true
					break
				}
			}

			// Add the compile dependency if it doesn't exist
			if !exists {
				module.Dependencies = append(module.Dependencies, model.Dependency{
					From: sourceTarget.Label,
					To:   depTarget.Label,
					Type: model.DependencyCompile,
				})
			}
		}
	}

	return nil
}

// NormalizeSourcePath converts a Bazel label source path to a workspace-relative path
// Example: "//main:main.cc" -> "main/main.cc"
func NormalizeSourcePath(labelPath string) string {
	// Remove leading "//" if present
	path := strings.TrimPrefix(labelPath, "//")

	// If there's a colon, it's a label like "//main:main.cc"
	if idx := strings.Index(path, ":"); idx != -1 {
		pkg := path[:idx]
		file := path[idx+1:]
		return filepath.Join(pkg, file)
	}

	// Otherwise it's already a file path
	return path
}

// findTargetForFile finds the target that owns a given file path
func findTargetForFile(filePath string, fileToTarget map[string]*model.Target) *model.Target {
	// Try exact match first
	if target, ok := fileToTarget[filePath]; ok {
		return target
	}

	// Try normalizing the file path (handle different formats)
	normalized := filepath.Clean(filePath)
	if target, ok := fileToTarget[normalized]; ok {
		return target
	}

	// Try matching by base path components
	// .d files might have paths like "util/math.cc" while targets have "//util:math.cc"
	for candidatePath, target := range fileToTarget {
		if strings.HasSuffix(candidatePath, filePath) || strings.HasSuffix(filePath, filepath.Base(candidatePath)) {
			// Check if the package matches
			if strings.Contains(candidatePath, filepath.Dir(filePath)) {
				return target
			}
		}
	}

	return nil
}

// AddSymbolDependencies adds symbol-level dependencies from nm analysis to the module
// It also detects and reports issues like duplicate symbols (both static and dynamic linkage)
func AddSymbolDependencies(module *model.Module, workspacePath string) error {
	// Build file-to-target and target-to-kind maps
	fileToTarget := make(map[string]string)
	targetToKind := make(map[string]string)

	for _, target := range module.Targets {
		targetToKind[target.Label] = string(target.Kind)

		// Map source files to their target
		for _, src := range target.Sources {
			filePath := NormalizeSourcePath(src)
			fileToTarget[filePath] = target.Label
		}
	}

	// Run symbol analysis
	symbolDeps, err := symbols.BuildSymbolGraph(workspacePath, fileToTarget, targetToKind)
	if err != nil {
		return fmt.Errorf("building symbol graph: %w", err)
	}

	// Track dependencies by source->target pair to detect conflicts
	depPairs := make(map[string][]model.DependencyType) // "from->to" -> list of types

	// Add symbol dependencies to module
	for _, symDep := range symbolDeps {
		if symDep.SourceTarget == "" || symDep.TargetTarget == "" {
			continue // Skip if we couldn't map to targets
		}

		// Skip dependencies within the same target
		if symDep.SourceTarget == symDep.TargetTarget {
			continue
		}

		// Check if this symbol dependency already exists
		exists := false
		for _, dep := range module.Dependencies {
			if dep.From == symDep.SourceTarget && dep.To == symDep.TargetTarget && dep.Type == model.DependencySymbol {
				exists = true
				break
			}
		}

		// Add the symbol dependency if it doesn't exist
		if !exists {
			module.Dependencies = append(module.Dependencies, model.Dependency{
				From: symDep.SourceTarget,
				To:   symDep.TargetTarget,
				Type: model.DependencySymbol,
			})
		}

		// Track this dependency type for conflict detection
		key := symDep.SourceTarget + " -> " + symDep.TargetTarget
		depPairs[key] = append(depPairs[key], model.DependencySymbol)
	}

	// Detect conflicts: Check if any dependency pair has both static/symbol and dynamic types
	for _, dep := range module.Dependencies {
		key := dep.From + " -> " + dep.To
		depPairs[key] = append(depPairs[key], dep.Type)
	}

	// Look for problematic combinations
	for key, types := range depPairs {
		hasStatic := false
		hasDynamic := false
		hasSymbol := false

		for _, t := range types {
			switch t {
			case model.DependencyStatic, model.DependencySymbol:
				if t == model.DependencyStatic {
					hasStatic = true
				}
				if t == model.DependencySymbol {
					hasSymbol = true
				}
			case model.DependencyDynamic:
				hasDynamic = true
			}
		}

		// Issue: Both static and dynamic linkage to the same target
		if (hasStatic || hasSymbol) && hasDynamic {
			parts := strings.Split(key, " -> ")
			if len(parts) == 2 {
				typeList := make([]string, 0)
				if hasStatic {
					typeList = append(typeList, "static")
				}
				if hasSymbol {
					typeList = append(typeList, "symbol")
				}
				if hasDynamic {
					typeList = append(typeList, "dynamic")
				}

				module.Issues = append(module.Issues, model.DependencyIssue{
					From:     parts[0],
					To:       parts[1],
					Issue:    "duplicate_linkage",
					Types:    typeList,
					Severity: "warning",
					Description: fmt.Sprintf("Target %s has both static and dynamic linkage to %s. "+
						"This can cause duplicate symbols and runtime issues. "+
						"Symbols may be included both statically (via deps) and dynamically (via dynamic_deps/shared library).",
						parts[0], parts[1]),
				})
			}
		}
	}

	return nil
}

// QueryAllSourceFiles returns all source files covered by Bazel targets
// This is a compatibility function for the old code
func QueryAllSourceFiles(workspacePath string) ([]string, error) {
	module, err := QueryWorkspace(workspacePath)
	if err != nil {
		return nil, err
	}

	// Collect all source files from all targets
	sourceFiles := make([]string, 0)
	seen := make(map[string]bool)

	for _, target := range module.Targets {
		for _, src := range target.Sources {
			normalized := NormalizeSourcePath(src)
			if !seen[normalized] {
				seen[normalized] = true
				sourceFiles = append(sourceFiles, normalized)
			}
		}
	}

	return sourceFiles, nil
}

// BuildFileToTargetMap creates a mapping from file paths to target labels
// This is a compatibility function for the old code
func BuildFileToTargetMap(workspacePath string) (map[string]string, error) {
	module, err := QueryWorkspace(workspacePath)
	if err != nil {
		return nil, err
	}

	fileToTarget := make(map[string]string)

	for _, target := range module.Targets {
		// Map source files
		for _, src := range target.Sources {
			filePath := NormalizeSourcePath(src)
			fileToTarget[filePath] = target.Label
		}
		// Map header files
		for _, hdr := range target.Headers {
			filePath := NormalizeSourcePath(hdr)
			fileToTarget[filePath] = target.Label
		}
	}

	return fileToTarget, nil
}

