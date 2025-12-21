package bazel

import (
	"path/filepath"
	"testing"
)

func TestQueryAllCCTargets(t *testing.T) {
	examplePath := filepath.Join("..", "..", "example")

	targets, err := QueryAllCCTargets(examplePath)
	if err != nil {
		t.Fatalf("QueryAllCCTargets() error = %v", err)
	}

	// Expected: 4 cc_* targets (util, core, plugins, main)
	expectedCount := 4
	if len(targets) != expectedCount {
		t.Errorf("QueryAllCCTargets() found %d targets, expected %d", len(targets), expectedCount)
	}

	// Check that we found expected targets
	expectedLabels := []string{"//util:util", "//core:core", "//plugins:renderer_plugin", "//main:test_app"}
	foundLabels := make(map[string]bool)
	for _, target := range targets {
		foundLabels[target.Label] = true
	}

	for _, expected := range expectedLabels {
		if !foundLabels[expected] {
			t.Errorf("QueryAllCCTargets() missing expected target: %s", expected)
		}
	}
}

func TestQuerySourceFilesForTarget(t *testing.T) {
	examplePath := filepath.Join("..", "..", "example")

	tests := []struct {
		name          string
		target        string
		shouldContain []string
		shouldNotContain []string
	}{
		{
			name:   "util target",
			target: "//util:util",
			shouldContain: []string{
				"util/strings.cc",
				"util/strings.h",
				"util/math.cc",
				"util/math.h",
			},
			shouldNotContain: []string{
				"util/orphaned.cc", // Not in BUILD.bazel
			},
		},
		{
			name:   "core target",
			target: "//core:core",
			shouldContain: []string{
				"core/engine.cc",
				"core/engine.h",
				"core/state.cc",
				"core/state.h",
			},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := QuerySourceFilesForTarget(examplePath, tt.target)
			if err != nil {
				t.Fatalf("QuerySourceFilesForTarget() error = %v", err)
			}

			fileSet := make(map[string]bool)
			for _, f := range files {
				fileSet[f] = true
			}

			for _, expected := range tt.shouldContain {
				if !fileSet[expected] {
					t.Errorf("QuerySourceFilesForTarget() missing file: %s", expected)
				}
			}

			for _, notExpected := range tt.shouldNotContain {
				if fileSet[notExpected] {
					t.Errorf("QuerySourceFilesForTarget() should not contain: %s", notExpected)
				}
			}
		})
	}
}

func TestQueryAllSourceFiles(t *testing.T) {
	examplePath := filepath.Join("..", "..", "example")

	files, err := QueryAllSourceFiles(examplePath)
	if err != nil {
		t.Fatalf("QueryAllSourceFiles() error = %v", err)
	}

	// Should find at least 15 files (all non-orphaned files)
	if len(files) < 15 {
		t.Errorf("QueryAllSourceFiles() found %d files, expected at least 15", len(files))
	}

	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	// Should contain files from util
	if !fileSet["util/strings.cc"] {
		t.Error("QueryAllSourceFiles() missing util/strings.cc")
	}

	// Should NOT contain orphaned.cc
	if fileSet["util/orphaned.cc"] {
		t.Error("QueryAllSourceFiles() should not contain util/orphaned.cc")
	}
}

func TestLabelToPath(t *testing.T) {
	tests := []struct {
		label    string
		expected string
	}{
		{"//util:strings.cc", "util/strings.cc"},
		{"//core:engine.h", "core/engine.h"},
		{"//main:main.cc", "main/main.cc"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			result := labelToPath(tt.label)
			if result != tt.expected {
				t.Errorf("labelToPath(%s) = %s, expected %s", tt.label, result, tt.expected)
			}
		})
	}
}
