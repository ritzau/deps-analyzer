package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ritzau/deps-analyzer/pkg/logging"
)

// FindDFiles finds all .d dependency files in the bazel-out directory
func FindDFiles(workspaceRoot string) ([]string, error) {
	var dfiles []string

	// Search in bazel-out directory
	bazelOutPath := filepath.Join(workspaceRoot, "bazel-out")

	// Resolve symlink if bazel-out is a symlink
	resolvedPath, err := filepath.EvalSymlinks(bazelOutPath)
	if err != nil {
		// If bazel-out doesn't exist or can't be resolved, return empty list (not an error)
		if os.IsNotExist(err) {
			return dfiles, nil
		}
		return nil, fmt.Errorf("resolving bazel-out symlink: %w", err)
	}

	logging.Debug("searching for .d files", "path", resolvedPath)

	err = filepath.Walk(resolvedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors for individual files
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include .d files that don't have extra suffixes
		// We want "math.d" but not "math.ii.d" or "math.s.d"
		if filepath.Ext(path) == ".d" {
			base := filepath.Base(path)
			// Check if it's a simple .d file (e.g., "math.d" not "math.ii.d")
			if strings.Count(base, ".") == 1 {
				dfiles = append(dfiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking bazel-out directory: %w", err)
	}

	logging.Debug("found .d files", "count", len(dfiles))
	return dfiles, nil
}

// ParseAllDFiles finds and parses all .d files in the workspace
func ParseAllDFiles(workspaceRoot string) ([]*FileDependency, error) {
	dfiles, err := FindDFiles(workspaceRoot)
	if err != nil {
		return nil, err
	}

	// Parse
	var deps []*FileDependency
	for _, dfile := range dfiles {
		dep, err := ParseDFile(dfile)
		if err != nil {
			logging.Debug("failed to parse dfile", "path", dfile, "error", err)
			continue
		}

		// Only include if we found a source file
		if dep.SourceFile != "" {
			deps = append(deps, dep)
		} else {
			logging.Debug("parsed dfile but no source file found", "path", dfile)
		}
	}

	logging.Debug("successfully parsed d files", "count", len(deps))
	return deps, nil
}

// Client abstracts the finding and parsing of .d files
type Client interface {
	ParseAllDFiles(workspaceRoot string) ([]*FileDependency, error)
}

// DefaultClient uses the actual filesystem
type DefaultClient struct{}

// NewClient creates a new default client
func NewClient() Client {
	return &DefaultClient{}
}

func (c *DefaultClient) ParseAllDFiles(workspaceRoot string) ([]*FileDependency, error) {
	return ParseAllDFiles(workspaceRoot)
}
