package bazel

import (
	"context"
)

// MockExecutor is a mock implementation of Executor for testing
type MockExecutor struct {
	MockOutput []byte
	MockError  error
}

func (m *MockExecutor) RunQuery(ctx context.Context, workspacePath string, query string) ([]byte, error) {
	return m.MockOutput, m.MockError
}
