package bazel

import (
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ritzau/deps-analyzer/pkg/model"
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
func QueryWorkspace(workspacePath string) (*model.Workspace, error) {
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

	// Build workspace structure
	workspace := &model.Workspace{
		Targets:      make(map[string]*model.Target),
		Dependencies: make([]model.Dependency, 0),
	}

	// First pass: create all targets
	for _, rule := range result.Rules {
		target := parseTarget(rule)
		if target != nil {
			workspace.Targets[target.Label] = target
		}
	}

	// Second pass: create typed dependencies
	for _, rule := range result.Rules {
		deps := parseDependencies(rule, workspace.Targets)
		workspace.Dependencies = append(workspace.Dependencies, deps...)
	}

	return workspace, nil
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

	// Extract attributes from lists
	for _, list := range rule.Lists {
		switch list.Name {
		case "srcs":
			for _, label := range list.Labels {
				if strings.HasSuffix(label.Value, ".cc") {
					target.Sources = append(target.Sources, label.Value)
				}
			}
		case "hdrs":
			for _, label := range list.Labels {
				if strings.HasSuffix(label.Value, ".h") || strings.HasSuffix(label.Value, ".hpp") {
					target.Headers = append(target.Headers, label.Value)
				}
			}
		case "deps":
			for _, label := range list.Labels {
				target.Deps = append(target.Deps, label.Value)
			}
		case "dynamic_deps":
			for _, label := range list.Labels {
				target.DynamicDeps = append(target.DynamicDeps, label.Value)
			}
		case "data":
			for _, label := range list.Labels {
				target.Data = append(target.Data, label.Value)
			}
		case "linkopts":
			for _, str := range list.Strings {
				target.Linkopts = append(target.Linkopts, str.Value)
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
