package finder

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// FindSourceFiles walks the workspace directory and returns all .cc and .h files,
// excluding bazel-* directories and other build artifacts.
func FindSourceFiles(workspaceRoot string) ([]string, error) {
	var sourceFiles []string

	err := filepath.WalkDir(workspaceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip bazel-* directories (symlinks to build outputs)
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, "bazel-") || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a C++ source or header file
		ext := filepath.Ext(path)
		if ext == ".cc" || ext == ".h" {
			sourceFiles = append(sourceFiles, path)
		}

		return nil
	})

	return sourceFiles, err
}
