.PHONY: build run test clean install install-tools build-frontend watch-frontend dev format lint setup-hooks

# Build the analyzer binary (with frontend compilation)
build: install-tools build-frontend
	go build -o deps-analyzer ./cmd/deps-analyzer

# Install all build and development tools
install-tools:
	@echo "Installing build and development tools..."
	@go install github.com/evanw/esbuild/cmd/esbuild@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/bazelbuild/buildtools/buildifier@latest
	@go install github.com/evilmartians/lefthook@latest
	@echo "Tools installed successfully!"

# Build frontend TypeScript files (if they exist)
build-frontend:
	@echo "Building frontend..."
	@if [ -d "pkg/web/static/src" ]; then \
		esbuild pkg/web/static/src/app.ts \
			--bundle \
			--outfile=pkg/web/static/app.js \
			--target=es2020 \
			--sourcemap; \
	else \
		echo "No TypeScript source directory found, skipping frontend build"; \
	fi

# Run the analyzer in web mode
run: build
	./deps-analyzer --workspace=./example --web --port=8080 --watch

# Run all tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f deps-analyzer
	rm -f analyzer
	# Uncomment these after migrating to TypeScript:
	# rm -f pkg/web/static/app.js
	# rm -f pkg/web/static/app.js.map

# Install to GOPATH/bin
install:
	go install ./cmd/deps-analyzer

# Development mode with watch (run in separate terminals)
dev: install-tools
	@echo "Starting development mode..."
	@echo "Run 'make watch-frontend' in another terminal to auto-compile TypeScript"
	cd example && ../deps-analyzer --web

# Watch frontend files for changes (run in separate terminal during development)
watch-frontend:
	@if [ -d "pkg/web/static/src" ]; then \
		esbuild pkg/web/static/src/app.ts \
			--bundle \
			--outfile=pkg/web/static/app.js \
			--target=es2020 \
			--sourcemap \
			--watch; \
	else \
		echo "No TypeScript source directory found"; \
		exit 1; \
	fi

# Set up git hooks with lefthook
setup-hooks: install-tools
	@echo "Setting up git hooks..."
	@$(shell go env GOPATH)/bin/lefthook install
	@echo "Git hooks installed! Hooks will run on git commit."

# Format all code
format: install-tools
	@echo "Formatting all code..."
	@$(shell go env GOPATH)/bin/lefthook run format

# Run linters
lint: install-tools
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

.DEFAULT_GOAL := build
