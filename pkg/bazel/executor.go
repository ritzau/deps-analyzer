package bazel

import (
	"context"
	"fmt"
	"os/exec"
)

// Executor handles the execution of Bazel commands
type Executor interface {
	RunQuery(ctx context.Context, workspacePath string, query string) ([]byte, error)
}

// DefaultExecutor is the default implementation of Executor that runs actual commands
type DefaultExecutor struct{}

// NewExecutor creates a new default Bazel executor
func NewExecutor() Executor {
	return &DefaultExecutor{}
}

// RunQuery executes a Bazel query and returns the raw XML output.
// It respects the provided context for cancellation.
func (e *DefaultExecutor) RunQuery(ctx context.Context, workspacePath string, query string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "bazel", "query", query, "--output=xml")
	cmd.Dir = workspacePath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}
