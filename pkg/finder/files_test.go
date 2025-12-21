package finder

import (
	"path/filepath"
	"testing"
)

func TestFindSourceFiles(t *testing.T) {
	// Test against the real example workspace
	examplePath := filepath.Join("..", "..", "example")

	files, err := FindSourceFiles(examplePath)
	if err != nil {
		t.Fatalf("FindSourceFiles() error = %v", err)
	}

	// Expected: 16 source files (8 .cc + 8 .h including orphaned.cc)
	// util: strings.cc/h, file_io.cc/h, time.cc/h, math.cc/h, orphaned.cc
	// core: engine.cc/h, state.cc/h
	// plugins: renderer.cc/h
	// main: main.cc
	expectedMin := 15 // At minimum we should find all non-orphaned files
	if len(files) < expectedMin {
		t.Errorf("FindSourceFiles() found %d files, expected at least %d", len(files), expectedMin)
	}

	// Check that orphaned.cc is found
	foundOrphaned := false
	for _, f := range files {
		if filepath.Base(f) == "orphaned.cc" {
			foundOrphaned = true
			break
		}
	}
	if !foundOrphaned {
		t.Error("FindSourceFiles() did not find orphaned.cc")
	}

	// Verify no bazel-* files are included
	for _, f := range files {
		if filepath.Base(f) == "bazel-bin" || filepath.Base(f) == "bazel-out" {
			t.Errorf("FindSourceFiles() should not include bazel-* directories, found: %s", f)
		}
	}
}
