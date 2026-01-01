package watcher

import (
	"context"
	"fmt"
	"github.com/ritzau/deps-analyzer/pkg/logging"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ChangeType represents the type of file change detected
type ChangeType int

const (
	ChangeTypeBuildFile ChangeType = iota
	ChangeTypeDFile
	ChangeTypeOFile
)

// ChangeEvent represents a batch of file system changes
type ChangeEvent struct {
	Type      ChangeType
	Paths     []string
	Timestamp time.Time
}

// FileWatcher watches a Bazel workspace for file changes
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	workspace string
	events    chan ChangeEvent
	done      chan struct{}
	mu        sync.Mutex
}

// NewFileWatcher creates a new file system watcher for a Bazel workspace
func NewFileWatcher(workspace string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:   watcher,
		workspace: workspace,
		events:    make(chan ChangeEvent, 100),
		done:      make(chan struct{}),
	}

	return fw, nil
}

// Start begins watching for file changes
func (fw *FileWatcher) Start(ctx context.Context) error {
	// Find and watch all directories containing BUILD files
	if err := fw.watchBuildFiles(); err != nil {
		logging.Warn("failed to watch BUILD files", "error", err)
	}

	// Watch bazel-out directory if it exists
	if err := fw.watchBazelOut(); err != nil {
		logging.Warn("failed to watch bazel-out", "error", err)
	}

	logging.Info("started watching workspace", "path", fw.workspace)

	// Process events
	go fw.processEvents(ctx)

	return nil
}

// watchBuildFiles finds and watches all directories containing BUILD files
func (fw *FileWatcher) watchBuildFiles() error {
	buildDirs := make(map[string]bool)

	err := filepath.Walk(fw.workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip bazel-* symlink directories
		if info.IsDir() && strings.HasPrefix(info.Name(), "bazel-") {
			return filepath.SkipDir
		}

		// Check if this is a BUILD file
		if !info.IsDir() && (info.Name() == "BUILD" || info.Name() == "BUILD.bazel") {
			dir := filepath.Dir(path)
			buildDirs[dir] = true
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk workspace: %w", err)
	}

	// Add all directories to watcher
	for dir := range buildDirs {
		if err := fw.watcher.Add(dir); err != nil {
			logging.Warn("failed to watch directory", "path", dir, "error", err)
		}
	}

	logging.Info("monitoring directories for BUILD files", "count", len(buildDirs))
	return nil
}

// watchBazelOut watches the bazel-out directory for artifact changes
func (fw *FileWatcher) watchBazelOut() error {
	bazelOut := filepath.Join(fw.workspace, "bazel-out")

	// Check if bazel-out exists
	if _, err := os.Stat(bazelOut); os.IsNotExist(err) {
		logging.Info("[WATCHER] bazel-out directory does not exist yet, skipping")
		return nil
	}

	// Resolve symlink if necessary
	resolvedPath, err := filepath.EvalSymlinks(bazelOut)
	if err != nil {
		return fmt.Errorf("failed to resolve bazel-out symlink: %w", err)
	}

	// Watch the resolved directory non-recursively
	if err := fw.watcher.Add(resolvedPath); err != nil {
		return fmt.Errorf("failed to watch bazel-out: %w", err)
	}

	logging.Info("monitoring bazel-out", "path", resolvedPath)
	return nil
}

// processEvents processes file system events and batches them by type
func (fw *FileWatcher) processEvents(ctx context.Context) {
	// Batch events to avoid sending one event per file
	var buildFiles []string
	var dFiles []string
	var oFiles []string

	flushTimer := time.NewTimer(100 * time.Millisecond)
	flushTimer.Stop()

	flush := func() {
		if len(buildFiles) > 0 {
			fw.events <- ChangeEvent{
				Type:      ChangeTypeBuildFile,
				Paths:     buildFiles,
				Timestamp: time.Now(),
			}
			buildFiles = nil
		}
		if len(dFiles) > 0 {
			fw.events <- ChangeEvent{
				Type:      ChangeTypeDFile,
				Paths:     dFiles,
				Timestamp: time.Now(),
			}
			dFiles = nil
		}
		if len(oFiles) > 0 {
			fw.events <- ChangeEvent{
				Type:      ChangeTypeOFile,
				Paths:     oFiles,
				Timestamp: time.Now(),
			}
			oFiles = nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			fw.watcher.Close()
			close(fw.events)
			close(fw.done)
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Filter to only relevant file types
			name := filepath.Base(event.Name)

			if name == "BUILD" || name == "BUILD.bazel" {
				buildFiles = append(buildFiles, event.Name)
				flushTimer.Reset(100 * time.Millisecond)
			} else if strings.HasSuffix(name, ".d") {
				dFiles = append(dFiles, event.Name)
				flushTimer.Reset(100 * time.Millisecond)
			} else if strings.HasSuffix(name, ".o") {
				oFiles = append(oFiles, event.Name)
				flushTimer.Reset(100 * time.Millisecond)
			}

		case <-flushTimer.C:
			flush()

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			logging.Error("watcher error", "error", err)
		}
	}
}

// Events returns the channel of change events
func (fw *FileWatcher) Events() <-chan ChangeEvent {
	return fw.events
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	close(fw.done)
	return fw.watcher.Close()
}
