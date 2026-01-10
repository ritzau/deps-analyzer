package bazel

import (
	"context"
	"fmt"
	"os/exec"
)

// Executor handles the execution of Bazel commands
type Executor struct{}

// NewExecutor creates a new Bazel executor
func NewExecutor() *Executor {
	return &Executor{}
}

// RunQuery executes a Bazel query and returns the raw XML output.
// It respects the provided context for cancellation.
func (e *Executor) RunQuery(ctx context.Context, workspacePath string, query string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "bazel", "query", query, "--output=xml")
	cmd.Dir = workspacePath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bazel query failed: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}
