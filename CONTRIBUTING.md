# Contributing to deps-analyzer

## Development Setup

### Prerequisites

- Go 1.21 or later
- Bazel 7.0 or later
- Make
- clang-format (for C++ formatting)

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/ritzau/deps-analyzer.git
cd deps-analyzer

# Install development tools
make install-tools

# Set up git hooks (automatic formatting on commit)
make setup-hooks
```

## Code Formatting and Linting

This project uses automated code formatting and linting enforced via git hooks.

### Git Hooks (Automatic)

After running `make setup-hooks`, pre-commit hooks will automatically:

1. **Format code** before each commit:
   - Go: `gofmt` + `goimports`
   - C++: `clang-format`
   - JavaScript/TypeScript: `prettier`
   - Bazel/Starlark: `buildifier`

2. **Strip trailing whitespace** from all source files

3. **Ensure single trailing newline** at end of files

### Manual Formatting

Format all code manually:

```bash
make format
```

### Linting

Run Go linters:

```bash
make lint
```

This requires `golangci-lint`. Install with:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Code Style Guidelines

### Go

- Follow standard Go conventions
- Use `gofmt` formatting
- Organize imports with `goimports`
- Keep functions focused and small
- Document exported functions and types

### C++

- Use `clang-format` with default Google style
- Prefer modern C++ features (C++17+)
- Use RAII and smart pointers
- Follow Bazel naming conventions for BUILD files

### JavaScript/TypeScript

- Use `prettier` with default settings
- Prefer ES2020+ features
- Use JSDoc comments for type hints (until TypeScript migration)

### Bazel/Starlark

- Use `buildifier` for formatting
- Follow Bazel style guide
- Keep BUILD files organized and readable

## File Conventions

- **No trailing whitespace** - automatically stripped by git hooks
- **Single trailing newline** - automatically ensured by git hooks
- **UTF-8 encoding** for all source files

## Building

```bash
# Build the binary
make build

# Run tests
make test

# Clean build artifacts
make clean
```

## Running the Analyzer

```bash
# Basic usage
./deps-analyzer --web --workspace=/path/to/bazel/workspace

# Development mode (auto-restart on changes)
make dev
```

## Submitting Changes

1. Create a feature branch
2. Make your changes
3. Ensure git hooks are installed: `make setup-hooks`
4. Commit your changes (hooks will auto-format)
5. Push and create a pull request

The git hooks ensure all code is properly formatted before commit, maintaining consistent style across the codebase.

